import re

with open("runtime/wolf_http_engine.c", "r") as f:
    content = f.read()

# Make wolf_engine_send_response not call write(), but return the buffer instead for io_uring, or call write() if not io_uring.
# Wait, we can just change wolf_engine_send_response to:
# int wolf_engine_send_response(WolfConnCtx* ctx, char** out_buf, int* out_len)
send_response_new = """static int wolf_engine_send_response(WolfConnCtx* ctx, char** out_buf, int* out_len) {
    const char* status_text = "OK";
    switch (ctx->status_code) {
        case 201: status_text = "Created"; break;
        case 204: status_text = "No Content"; break;
        case 301: status_text = "Moved Permanently"; break;
        case 302: status_text = "Found"; break;
        case 400: status_text = "Bad Request"; break;
        case 401: status_text = "Unauthorized"; break;
        case 403: status_text = "Forbidden"; break;
        case 404: status_text = "Not Found"; break;
        case 429: status_text = "Too Many Requests"; break;
        case 500: status_text = "Internal Server Error"; break;
        case 503: status_text = "Service Unavailable"; break;
        default:  break;
    }

    int body_len = ctx->res_body ? (int)strlen(ctx->res_body) : 0;
    
    int total_size = 128;
    for (int i = 0; i < ctx->res_header_count; i++) {
        total_size += strlen(ctx->res_header_keys[i]) + strlen(ctx->res_header_vals[i]) + 4;
    }
    total_size += body_len;

    char* response = (char*)wolf_arena_alloc(ctx->arena, total_size + 64);
    if (!response) return -1;
    
    char* ptr = response;
    ptr += snprintf(ptr, 128, "HTTP/1.1 %d %s\\r\\n", ctx->status_code, status_text);
    
    for (int i = 0; i < ctx->res_header_count; i++) {
        ptr += sprintf(ptr, "%s: %s\\r\\n", ctx->res_header_keys[i], ctx->res_header_vals[i]);
    }
    
    ptr += sprintf(ptr, "Content-Length: %d\\r\\nConnection: keep-alive\\r\\n\\r\\n", body_len);
    
    if (body_len > 0) {
        memcpy(ptr, ctx->res_body, body_len);
        ptr += body_len;
    }
    
    int final_len = ptr - response;
    
    if (out_buf) *out_buf = response;
    if (out_len) *out_len = final_len;
    
    return 0;
}
"""

content = re.sub(r'static int wolf_engine_send_response\(WolfConnCtx\* ctx\) \{.*?return 0;\n\}', send_response_new, content, flags=re.DOTALL)


callbacks_and_thread = """#if defined(WOLF_HAS_IO_URING)

static void on_send_complete(int client_fd, void* ctx_ptr, int bytes_written) {
    WolfConnCtx* ctx = (WolfConnCtx*)ctx_ptr;
    WolfCore* core = ctx->core;
    if (bytes_written >= 0) {
        __atomic_fetch_add(&core->requests_total, 1, __ATOMIC_RELAXED);
        __atomic_fetch_add(&core->bytes_in,  ctx->bytes_in,                              __ATOMIC_RELAXED);
        __atomic_fetch_add(&core->bytes_out, ctx->res_body ? strlen(ctx->res_body) : 0, __ATOMIC_RELAXED);
    }
    __atomic_fetch_sub(&core->requests_active, 1, __ATOMIC_RELAXED);
    close(client_fd);
    wolf_req_arena_flush();
    wolf_core_free_ctx(ctx);
}

static void on_recv_complete(int client_fd, void* ctx_ptr, int bytes_read) {
    WolfConnCtx* ctx = (WolfConnCtx*)ctx_ptr;
    WolfCore* core = ctx->core;
    WolfCoreArgs* args = (WolfCoreArgs*)core->args;

    if (bytes_read <= 0) {
        if (bytes_read < 0 && (errno == EAGAIN || errno == EWOULDBLOCK || bytes_read == -ETIME)) {
            const char* timeout_resp = "HTTP/1.1 408 Request Timeout\\r\\nContent-Length: 0\\r\\nConnection: close\\r\\n\\r\\n";
            write(client_fd, timeout_resp, strlen(timeout_resp));
        }
        close(client_fd);
        wolf_core_free_ctx(ctx);
        return;
    }
    ctx->bytes_in = bytes_read;
    ctx->read_buf[bytes_read] = '\\0';
    
    wolf_engine_parse_request(ctx, ctx->read_buf, bytes_read);

    /* GET /health Observability Bypass */
    if (ctx->method && strcmp(ctx->method, "GET") == 0 && ctx->path && strcmp(ctx->path, "/health") == 0) {
        char health_buf[4096];
        extern int wolf_get_active_requests(void);
        int active = wolf_get_active_requests();
        
        int n = snprintf(health_buf, sizeof(health_buf), "{\\"status\\":\\"ok\\",\\"active_requests\\":%d,\\"metrics\\":{", active);
        
        int first = 1;
        extern wolf_metric_t wolf_metrics_registry[];
        for (int i = 0; i < WOLF_MAX_METRICS; i++) {
            const char* k = atomic_load(&wolf_metrics_registry[i].key_ptr);
            if (k) {
                if (!first) {
                    if (n < sizeof(health_buf) - 1) health_buf[n++] = ',';
                }
                first = 0;
                n += snprintf(health_buf + n, sizeof(health_buf) - n, "\\"%s\\":%lld", 
                              wolf_metrics_registry[i].name, 
                              (long long)atomic_load(&wolf_metrics_registry[i].count));
            }
        }
        n += snprintf(health_buf + n, sizeof(health_buf) - n, "}}");
        
        ctx->res_body = wolf_arena_strdup(ctx->arena, health_buf);
        ctx->status_code = 200;
        ctx->res_header_keys[0] = "Content-Type";
        ctx->res_header_vals[0] = "application/json";
        ctx->res_header_keys[1] = "Access-Control-Allow-Origin";
        ctx->res_header_vals[1] = "*";
        ctx->res_header_count = 2;
        goto handle_send;
    }

    if (ctx->is_websocket) {
        wolf_engine_ws_handshake(ctx);
        extern int wolf_engine_register_ws_fd(int, const char*, const char*, const char*, const char*, const char*);
        wolf_engine_register_ws_fd(ctx->client_fd,
                                   ctx->method, ctx->path, ctx->query,
                                   ctx->ws_key,
                                   ctx->client_ip);
        ctx->client_fd = -1;
        wolf_core_free_ctx(ctx);
        return;
    }

    int64_t ctx_id = (int64_t)(ctx - wolf_core_ctxs);
    ctx->request_id = ctx_id;
    ctx->oom_triggered = 0;
    ctx->arena_used = ctx->arena ? ctx->arena->total_allocated + ctx->arena->pos : 0;
    ctx->arena_cap = WOLF_MAX_REQUEST_MEMORY;
    extern void wolf_set_current_context(void*, void*);
    wolf_set_current_context((void*)(intptr_t)ctx_id, (void*)(intptr_t)ctx_id);

    __atomic_fetch_add(&core->requests_active, 1, __ATOMIC_RELAXED);
    if (args->http_handler) {
        __atomic_store_n(&wolf_active_ctx, ctx, __ATOMIC_RELEASE);
        wolf_req_arena_init();
        wolf_closure_t* closure = (wolf_closure_t*)args->http_handler;
        typedef void* (*wolf_closure_fn_t)(void* env, int64_t req_id, int64_t res_id);
        wolf_closure_fn_t fn = (wolf_closure_fn_t)closure->fn;

        if (!wolf_closure_valid(closure)) {
            ctx->status_code = 500;
            __atomic_store_n(&wolf_active_ctx, NULL, __ATOMIC_RELEASE);
            goto handle_send;
        }

        ctx->oom_triggered = 0;
        if (setjmp(ctx->oom_jump) == 0) {
            fn(closure->env, ctx_id, ctx_id);
        } else {
            ctx->status_code = 500;
            ctx->res_body = "500 Internal Server Error (OOM)";
        }

        __atomic_store_n(&wolf_active_ctx, NULL, __ATOMIC_RELEASE);

        if (wolf_req_oom_check()) {
            ctx->status_code = 503;
            if (!ctx->res_body) ctx->res_body = "Service Unavailable";
            wolf_req_oom_clear();
        }
    }

handle_send:
    ;
    char* out_buf = NULL;
    int out_len = 0;
    wolf_engine_send_response(ctx, &out_buf, &out_len);
    if (out_buf && out_len > 0) {
        wolf_uring_submit_send(core->sentinel->uring, client_fd, out_buf, out_len, on_send_complete, ctx, ctx->arena);
        wolf_uring_flush(core->sentinel->uring);
    } else {
        on_send_complete(client_fd, ctx, -1);
    }
}

static void on_accept_complete(int server_fd, void* core_ptr, int client_fd) {
    if (client_fd < 0) return;
    WolfCore* core = (WolfCore*)core_ptr;
    
    struct timeval tv = { WOLF_REQUEST_TIMEOUT_SEC, 0 };
    setsockopt(client_fd, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv));
    int opt = 1;
    setsockopt(client_fd, IPPROTO_TCP, TCP_NODELAY, &opt, sizeof(opt));
    
    WolfArena* arena = wolf_arena_acquire(core->arena_pool);
    WolfConnCtx* ctx = wolf_core_alloc_ctx(core, client_fd, arena);
    if (!ctx) {
        const char* busy = "HTTP/1.1 503 Service Unavailable\\r\\nContent-Length: 0\\r\\n\\r\\n";
        write(client_fd, busy, strlen(busy));
        close(client_fd);
        wolf_arena_reset(arena);
        __atomic_fetch_add(&core->requests_total, 1, __ATOMIC_RELAXED);
        return;
    }
    
    ctx->core = core;
    ctx->read_buf = (char*)wolf_arena_alloc(arena, WOLF_MAX_REQUEST_SIZE);
    
    wolf_uring_submit_recv(core->sentinel->uring, client_fd, ctx->read_buf, WOLF_MAX_REQUEST_SIZE - 1, on_recv_complete, ctx, arena);
    wolf_uring_flush(core->sentinel->uring);
}
#endif

static void* wolf_core_thread(void* arg) {
    WolfCoreArgs* args = (WolfCoreArgs*)arg;
    WolfCore*     core = args->core;
    core->args = args;

    wolf_ctx_freelist_init();

    stack_t altstack;
    altstack.ss_sp = malloc(SIGSTKSZ);
    if (altstack.ss_sp) {
        altstack.ss_size = SIGSTKSZ;
        altstack.ss_flags = 0;
        sigaltstack(&altstack, NULL);
    }

    wolf_pin_to_core(core->core_id);

    fprintf(stderr, "[WOLF-ENGINE] Core %d started (tid=%lu)\\n",
            core->core_id, (unsigned long)pthread_self());

#if defined(WOLF_HAS_IO_URING)
    // 1. Submit the multishot accept SQE once
    wolf_uring_submit_accept(core->sentinel->uring, core->server_fd, on_accept_complete, core, NULL);
    wolf_uring_flush(core->sentinel->uring);
    
    __atomic_store_n(&core->ready, 1, __ATOMIC_RELEASE);
    
    // 2. Poll for completions indefinitely
    while (!wolf_engine_shutdown_flag) {
        wolf_uring_poll(core->sentinel->uring, 100);
    }
#else
    wolf_sentinel_add(core->sentinel, core->server_fd, NULL, NULL);
    __atomic_store_n(&core->ready, 1, __ATOMIC_RELEASE);
    char read_buf[WOLF_MAX_REQUEST_SIZE];
    while (!wolf_engine_shutdown_flag) {
        struct sockaddr_in client_addr;
        socklen_t client_len = sizeof(client_addr);
        int client_fd = accept(core->server_fd,
                               (struct sockaddr*)&client_addr, &client_len);
        if (client_fd < 0) {
            if (errno == EAGAIN || errno == EWOULDBLOCK) {
                wolf_sentinel_poll(core->sentinel, 1);
                continue;
            }
            if (errno == EINTR) continue;
            perror("accept");
            continue;
        }

        struct timeval tv = { WOLF_REQUEST_TIMEOUT_SEC, 0 };
        setsockopt(client_fd, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv));
        int opt = 1;
        setsockopt(client_fd, IPPROTO_TCP, TCP_NODELAY, &opt, sizeof(opt));

        WolfArena* arena = wolf_arena_acquire(core->arena_pool);
        WolfConnCtx* ctx = wolf_core_alloc_ctx(core, client_fd, arena);
        if (!ctx) {
            const char* busy = "HTTP/1.1 503 Service Unavailable\\r\\nContent-Length: 0\\r\\n\\r\\n";
            write(client_fd, busy, strlen(busy));
            close(client_fd);
            wolf_arena_reset(arena);
            __atomic_fetch_add(&core->requests_total, 1, __ATOMIC_RELAXED);
            continue;
        }
        ctx->core = core;

        ssize_t bytes = read(client_fd, read_buf, sizeof(read_buf) - 1);
        if (bytes <= 0) {
            if (bytes < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) {
                const char* timeout_resp = "HTTP/1.1 408 Request Timeout\\r\\nContent-Length: 0\\r\\nConnection: close\\r\\n\\r\\n";
                write(client_fd, timeout_resp, strlen(timeout_resp));
            }
            close(client_fd);
            wolf_core_free_ctx(ctx);
            continue;
        }
        read_buf[bytes] = '\\0';

        wolf_engine_parse_request(ctx, read_buf, bytes);

        if (ctx->method && strcmp(ctx->method, "GET") == 0 && ctx->path && strcmp(ctx->path, "/health") == 0) {
            char health_buf[4096];
            extern int wolf_get_active_requests(void);
            int active = wolf_get_active_requests();
            int n = snprintf(health_buf, sizeof(health_buf), "{\\"status\\":\\"ok\\",\\"active_requests\\":%d,\\"metrics\\":{", active);
            int first = 1;
            extern wolf_metric_t wolf_metrics_registry[];
            for (int i = 0; i < WOLF_MAX_METRICS; i++) {
                const char* k = atomic_load(&wolf_metrics_registry[i].key_ptr);
                if (k) {
                    if (!first) {
                        if (n < sizeof(health_buf) - 1) health_buf[n++] = ',';
                    }
                    first = 0;
                    n += snprintf(health_buf + n, sizeof(health_buf) - n, "\\"%s\\":%lld", 
                                  wolf_metrics_registry[i].name, 
                                  (long long)atomic_load(&wolf_metrics_registry[i].count));
                }
            }
            n += snprintf(health_buf + n, sizeof(health_buf) - n, "}}");
            ctx->res_body = wolf_arena_strdup(ctx->arena, health_buf);
            ctx->status_code = 200;
            ctx->res_header_keys[0] = "Content-Type";
            ctx->res_header_vals[0] = "application/json";
            ctx->res_header_keys[1] = "Access-Control-Allow-Origin";
            ctx->res_header_vals[1] = "*";
            ctx->res_header_count = 2;
            goto send_and_cleanup;
        }

        if (ctx->is_websocket) {
            wolf_engine_ws_handshake(ctx);
            extern int wolf_engine_register_ws_fd(int, const char*, const char*, const char*, const char*, const char*);
            wolf_engine_register_ws_fd(ctx->client_fd,
                                       ctx->method, ctx->path, ctx->query,
                                       ctx->ws_key,
                                       ctx->client_ip);
            ctx->client_fd = -1;
            wolf_core_free_ctx(ctx);
            continue;
        }

        int64_t ctx_id = (int64_t)(ctx - wolf_core_ctxs);
        ctx->request_id = ctx_id;
        ctx->oom_triggered = 0;
        ctx->arena_used = ctx->arena ? ctx->arena->total_allocated + ctx->arena->pos : 0;
        ctx->arena_cap = WOLF_MAX_REQUEST_MEMORY;
        extern void wolf_set_current_context(void*, void*);
        wolf_set_current_context((void*)(intptr_t)ctx_id, (void*)(intptr_t)ctx_id);

        if (ctx->method && strcmp(ctx->method, "GET") == 0 && ctx->path && strcmp(ctx->path, "/crash") == 0) {
            __atomic_store_n(&wolf_active_ctx, ctx, __ATOMIC_RELEASE);
            fprintf(stderr, "Triggering deliberate crash on core %d for test...\\n", core->core_id);
            int* ptr = NULL;
            *ptr = 42;
        }

        __atomic_fetch_add(&core->requests_active, 1, __ATOMIC_RELAXED);
        if (args->http_handler) {
            __atomic_store_n(&wolf_active_ctx, ctx, __ATOMIC_RELEASE);
            wolf_req_arena_init();
            wolf_closure_t* closure = (wolf_closure_t*)args->http_handler;
            typedef void* (*wolf_closure_fn_t)(void* env, int64_t req_id, int64_t res_id);
            wolf_closure_fn_t fn = (wolf_closure_fn_t)closure->fn;

            if (!wolf_closure_valid(closure)) {
                fprintf(stderr, "[WOLF-ENGINE] CORRUPTED closure on core %d: closure=%p\\n", core->core_id, (void*)closure);
                ctx->status_code = 500;
                __atomic_store_n(&wolf_active_ctx, NULL, __ATOMIC_RELEASE);
                goto send_and_cleanup;
            }

            ctx->oom_triggered = 0;
            if (setjmp(ctx->oom_jump) == 0) {
                fn(closure->env, ctx_id, ctx_id);
            } else {
                ctx->status_code = 500;
                ctx->res_body = "500 Internal Server Error (OOM)";
            }

            __atomic_store_n(&wolf_active_ctx, NULL, __ATOMIC_RELEASE);

            if (wolf_req_oom_check()) {
                ctx->status_code = 503;
                if (!ctx->res_body) ctx->res_body = "Service Unavailable";
                wolf_req_oom_clear();
            }
        }

send_and_cleanup:
        __atomic_fetch_sub(&core->requests_active, 1, __ATOMIC_RELAXED);
        char* out_buf = NULL;
        int out_len = 0;
        int send_ok = wolf_engine_send_response(ctx, &out_buf, &out_len);
        if (send_ok == 0 && out_buf && out_len > 0) {
            write(client_fd, out_buf, out_len);
            __atomic_fetch_add(&core->requests_total, 1, __ATOMIC_RELAXED);
            __atomic_fetch_add(&core->bytes_in,  bytes, __ATOMIC_RELAXED);
            __atomic_fetch_add(&core->bytes_out, ctx->res_body ? strlen(ctx->res_body) : 0, __ATOMIC_RELAXED);
        }

        close(client_fd);
        wolf_req_arena_flush();
        wolf_core_free_ctx(ctx);
    }
#endif

    fprintf(stderr, "[WOLF-ENGINE] Core %d shutting down (served %lld requests)\\n",
            core->core_id, (long long)core->requests_total);

    free(args);
    return NULL;
}
"""

content = re.sub(r'static void\* wolf_core_thread\(void\* arg\) \{.*return NULL;\n\}', callbacks_and_thread, content, flags=re.DOTALL)

# Because we changed wolf_engine_send_response signature, we need to fix one other call in parsing multipart maybe? 
# Wait, wolf_engine_send_response is only called here. Let's make sure.

with open("runtime/wolf_http_engine.c", "w") as f:
    f.write(content)

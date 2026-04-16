/*
 * wolf_http_engine.c — Phase 1: Thread-Per-Core HTTP Engine
 *
 * Architecture:
 *   - detect_nproc() threads, one per CPU core
 *   - SO_REUSEPORT: kernel load-balances accept() across cores
 *   - Each core has its own epoll/kqueue instance (no shared poller)
 *   - Per-core arena pool: arena_acquire/reset replaces malloc/free
 *   - Cache-local: all request data stays on the same core's L1/L2
 *
 * Target: 12,000+ RPS on a single VPS core (Phase 1)
 *         18,000+ RPS with io_uring SQPOLL (Phase 2)
 *
 * Current state: replaces wolf_http_serve() in wolf_runtime.c
 */

#define _GNU_SOURCE
#define _POSIX_C_SOURCE 200809L

#include "wolf_config_runtime.h"
#include "wolf_http_engine.h"
#include "wolf_runtime.h"
#include "wolf_uring.h"
#include <openssl/evp.h>

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <unistd.h>
#include <fcntl.h>
#include <signal.h>
#include <pthread.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <netinet/tcp.h>
#include <arpa/inet.h>
#include <sys/stat.h>
#include <time.h>

#if defined(__linux__)
#  include <sys/epoll.h>
#  include <sched.h>        /* CPU_SET, sched_setaffinity */
#  define WOLF_HAS_EPOLL 1
#elif defined(__APPLE__)
#  include <sys/event.h>
#  define WOLF_HAS_KQUEUE 1
#else
#  include <poll.h>
#  define WOLF_HAS_POLL 1
#endif

/* ================================================================
 * Detect CPU count
 * ================================================================ */

static int wolf_detect_nproc(void) {
#if defined(__linux__)
    cpu_set_t cs;
    CPU_ZERO(&cs);
    if (sched_getaffinity(0, sizeof(cs), &cs) == 0)
        return CPU_COUNT(&cs);
#endif
    long n = sysconf(_SC_NPROCESSORS_ONLN);
    if (n > 0) return (int)n;
    return 1;
}

/* ================================================================
 * Pin thread to core (Linux: sched_setaffinity, macOS: thread_policy)
 * ================================================================ */

static void wolf_pin_to_core(int core_id) {
#if defined(__linux__)
    cpu_set_t cs;
    CPU_ZERO(&cs);
    CPU_SET(core_id, &cs);
    if (sched_setaffinity(0, sizeof(cs), &cs) != 0) {
        fprintf(stderr, "[WOLF-ENGINE] Warning: could not pin thread to core %d: %s\n",
                core_id, strerror(errno));
    }
#elif defined(__APPLE__)
    /* macOS thread affinity is advisory only */
    (void)core_id;
#else
    (void)core_id;
#endif
}

/* ================================================================
 * Create SO_REUSEPORT server socket
 * Each core gets its own socket bound to the same port.
 * The kernel load-balances accept() calls across them.
 * ================================================================ */

static int wolf_create_server_socket(int port) {
    int fd = socket(AF_INET, SOCK_STREAM, 0);
    if (fd < 0) { perror("socket"); return -1; }

    int opt = 1;
    setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));

#ifdef SO_REUSEPORT
    setsockopt(fd, SOL_SOCKET, SO_REUSEPORT, &opt, sizeof(opt));
#endif

    /* TCP_NODELAY — disable Nagle for lower latency */
    setsockopt(fd, IPPROTO_TCP, TCP_NODELAY, &opt, sizeof(opt));

    struct sockaddr_in addr;
    memset(&addr, 0, sizeof(addr));
    addr.sin_family      = AF_INET;
    addr.sin_addr.s_addr = INADDR_ANY;
    addr.sin_port        = htons(port);

    if (bind(fd, (struct sockaddr*)&addr, sizeof(addr)) < 0) {
        perror("bind"); close(fd); return -1;
    }
    if (listen(fd, 4096) < 0) {
        perror("listen"); close(fd); return -1;
    }

    /* Non-blocking */
    int flags = fcntl(fd, F_GETFL, 0);
    fcntl(fd, F_SETFL, flags | O_NONBLOCK);

    return fd;
}

/* ================================================================
 * Sentinel — cross-platform I/O poller
 * ================================================================ */

WolfSentinel* wolf_sentinel_create(int core_id) {
    WolfSentinel* s = (WolfSentinel*)calloc(1, sizeof(WolfSentinel));
    s->core_id = core_id;

#if defined(WOLF_HAS_IO_URING)
    s->backend = WOLF_IO_IOURING;
    s->uring = wolf_uring_create(64, 1); // 64 entries, SQPOLL enabled
    if (!s->uring) {
        fprintf(stderr, "[WOLF-ENGINE] io_uring init failed, falling back\n");
        // Fallthrough macro logic here requires careful handling, but for now we just exit
        exit(1);
    }
    s->poll_fd = -1;

#elif defined(WOLF_HAS_EPOLL)
    s->backend = WOLF_IO_EPOLL;
    s->poll_fd = epoll_create1(EPOLL_CLOEXEC);
    if (s->poll_fd < 0) { perror("epoll_create1"); free(s); return NULL; }

#elif defined(WOLF_HAS_KQUEUE)
    s->backend = WOLF_IO_KQUEUE;
    s->poll_fd = kqueue();
    if (s->poll_fd < 0) { perror("kqueue"); free(s); return NULL; }

#else
    s->backend = WOLF_IO_POLL;
    s->poll_fd = -1;
#endif

    return s;
}

/* Context storage for callbacks — maps fd → callback+ctx */
#define WOLF_SENTINEL_MAX_FDS 4096

typedef struct {
    int                fd;
    wolf_io_callback_t cb;
    void*              ctx;
} WolfFDEntry;

static __thread WolfFDEntry wolf_fd_table[WOLF_SENTINEL_MAX_FDS];
static __thread int         wolf_fd_table_count = 0;

static WolfFDEntry* wolf_fd_find(int fd) {
    for (int i = 0; i < wolf_fd_table_count; i++)
        if (wolf_fd_table[i].fd == fd) return &wolf_fd_table[i];
    return NULL;
}

static WolfFDEntry* wolf_fd_alloc(int fd, wolf_io_callback_t cb, void* ctx) {
    if (wolf_fd_table_count >= WOLF_SENTINEL_MAX_FDS) return NULL;
    WolfFDEntry* e = &wolf_fd_table[wolf_fd_table_count++];
    e->fd = fd; e->cb = cb; e->ctx = ctx;
    return e;
}

static void wolf_fd_remove_entry(int fd) {
    for (int i = 0; i < wolf_fd_table_count; i++) {
        if (wolf_fd_table[i].fd == fd) {
            wolf_fd_table[i] = wolf_fd_table[--wolf_fd_table_count];
            return;
        }
    }
}

int wolf_sentinel_add(WolfSentinel* s, int fd, wolf_io_callback_t cb, void* ctx) {
    wolf_fd_alloc(fd, cb, ctx);

#if defined(WOLF_HAS_IO_URING)
    // io_uring is I/O completion based, not readiness based like epoll.
    // the user must submit specific RECV/SEND/ACCEPT ops through wolf_uring_submit_* instead.
    return 0;

#elif defined(WOLF_HAS_EPOLL)
    struct epoll_event ev;
    ev.events   = EPOLLIN | EPOLLET; /* Edge-triggered for performance */
    ev.data.fd  = fd;
    return epoll_ctl(s->poll_fd, EPOLL_CTL_ADD, fd, &ev);

#elif defined(WOLF_HAS_KQUEUE)
    struct kevent ev;
    EV_SET(&ev, fd, EVFILT_READ, EV_ADD | EV_ENABLE, 0, 0, (void*)(intptr_t)fd);
    return kevent(s->poll_fd, &ev, 1, NULL, 0, NULL);

#else
    (void)s;
    return 0;
#endif
}

int wolf_sentinel_remove(WolfSentinel* s, int fd) {
    wolf_fd_remove_entry(fd);

#if defined(WOLF_HAS_IO_URING)
    // io_uring removals are implicit (the request completes or gets cancelled)
    return 0;

#elif defined(WOLF_HAS_EPOLL)
    return epoll_ctl(s->poll_fd, EPOLL_CTL_DEL, fd, NULL);

#elif defined(WOLF_HAS_KQUEUE)
    struct kevent ev;
    EV_SET(&ev, fd, EVFILT_READ, EV_DELETE, 0, 0, NULL);
    return kevent(s->poll_fd, &ev, 1, NULL, 0, NULL);

#else
    (void)s; (void)fd;
    return 0;
#endif
}

int wolf_sentinel_poll(WolfSentinel* s, int timeout_ms) {
#if defined(WOLF_HAS_IO_URING)
    if (s->backend == WOLF_IO_IOURING) {
        return wolf_uring_poll((WolfURing*)s->uring, timeout_ms);
    }
    return 0;

#elif defined(WOLF_HAS_EPOLL)
    struct epoll_event events[64];
    int n = epoll_wait(s->poll_fd, events, 64, timeout_ms);
    for (int i = 0; i < n; i++) {
        WolfFDEntry* e = wolf_fd_find(events[i].data.fd);
        if (e && e->cb) e->cb(events[i].data.fd, e->ctx, (int)events[i].events);
    }
    return n;

#elif defined(WOLF_HAS_KQUEUE)
    struct kevent events[64];
    struct timespec ts = { timeout_ms / 1000, (timeout_ms % 1000) * 1000000L };
    int n = kevent(s->poll_fd, NULL, 0, events, 64, timeout_ms < 0 ? NULL : &ts);
    for (int i = 0; i < n; i++) {
        int fd = (int)(intptr_t)events[i].udata;
        WolfFDEntry* e = wolf_fd_find(fd);
        if (e && e->cb) e->cb(fd, e->ctx, EPOLLIN);
    }
    return n;

#else
    /* poll() fallback */
    struct pollfd fds[WOLF_SENTINEL_MAX_FDS];
    int count = 0;
    for (int i = 0; i < wolf_fd_table_count; i++) {
        fds[count].fd      = wolf_fd_table[i].fd;
        fds[count].events  = POLLIN;
        fds[count].revents = 0;
        count++;
    }
    int n = poll(fds, count, timeout_ms);
    if (n > 0) {
        for (int i = 0; i < count; i++) {
            if (fds[i].revents & POLLIN) {
                WolfFDEntry* e = wolf_fd_find(fds[i].fd);
                if (e && e->cb) e->cb(fds[i].fd, e->ctx, POLLIN);
            }
        }
    }
    return n;
#endif
}

void wolf_sentinel_destroy(WolfSentinel* s) {
    if (!s) return;
    if (s->poll_fd >= 0) close(s->poll_fd);
    free(s);
}

/* ================================================================
 * Per-Core Arena Pool
 * ================================================================ */

WolfArenaPool* wolf_arena_pool_create(int core_id) {
    WolfArenaPool* pool = (WolfArenaPool*)calloc(1, sizeof(WolfArenaPool));
    pool->core_id = core_id;
    pool->count   = WOLF_ARENA_POOL_SIZE;

    for (int i = 0; i < WOLF_ARENA_POOL_SIZE; i++) {
        pool->arenas[i].slab   = (char*)malloc(WOLF_ARENA_SLAB_SIZE);
        pool->arenas[i].cap    = WOLF_ARENA_SLAB_SIZE;
        pool->arenas[i].pos    = 0;
        pool->arenas[i].in_use = 0;
        if (!pool->arenas[i].slab) {
            fprintf(stderr, "[WOLF-ENGINE] OOM allocating arena pool for core %d\n", core_id);
            /* Continue with partial allocation */
        }
    }
    return pool;
}

WolfArena* wolf_arena_acquire(WolfArenaPool* pool) {
    for (int i = 0; i < pool->count; i++) {
        if (!pool->arenas[i].in_use && pool->arenas[i].slab) {
            pool->arenas[i].in_use = 1;
            pool->arenas[i].pos    = 0;
            return &pool->arenas[i];
        }
    }
    /* All arenas busy — allocate a temporary one (fallback) */
    WolfArena* tmp = (WolfArena*)malloc(sizeof(WolfArena));
    tmp->slab   = (char*)malloc(WOLF_ARENA_SLAB_SIZE);
    tmp->cap    = WOLF_ARENA_SLAB_SIZE;
    tmp->pos    = 0;
    tmp->in_use = 1;
    return tmp;
}

void* wolf_arena_alloc(WolfArena* arena, size_t size) {
    /* Align to 8 bytes */
    size = (size + 7) & ~(size_t)7;
    if (arena->pos + size > arena->cap) {
        /* Overflow — fall back to malloc (tracked separately) */
        return calloc(1, size);
    }
    void* p = arena->slab + arena->pos;
    arena->pos += size;
    memset(p, 0, size);
    return p;
}

char* wolf_arena_strdup(WolfArena* arena, const char* s) {
    if (!s) return NULL;
    size_t len = strlen(s);
    char* p = (char*)wolf_arena_alloc(arena, len + 1);
    if (p) { memcpy(p, s, len); p[len] = '\0'; }
    return p;
}

void wolf_arena_reset(WolfArena* arena) {
    arena->pos    = 0;  /* O(1) — just reset the pointer */
    arena->in_use = 0;
}

void wolf_arena_pool_destroy(WolfArenaPool* pool) {
    if (!pool) return;
    for (int i = 0; i < pool->count; i++) {
        if (pool->arenas[i].slab) free(pool->arenas[i].slab);
    }
    free(pool);
}

/* ================================================================
 * HTTP Connection State — per-core, no mutex needed
 * ================================================================ */

typedef struct {
    int     active;
    int     client_fd;
    int     core_id;

    /* Request */
    char*   method;
    char*   path;
    char*   query;
    char*   body;
    char*   header_keys[32];
    char*   header_vals[32];
    int     header_count;

    /* Response */
    int     status_code;
    char*   res_header_keys[32];
    char*   res_header_vals[32];
    int     res_header_count;
    char*   res_body;

    /* WebSocket */
    int     is_websocket;
    char*   ws_key;

    /* Arena for this request */
    WolfArena* arena;

    /* Timing */
    struct timespec started_at;
} WolfConnCtx;

#define WOLF_CORE_CTX_MAX WOLF_CORE_MAX_CONNECTIONS

/* ================================================================
 * HTTP Request Parser (arena-backed, zero-copy where possible)
 * ================================================================ */

static void wolf_engine_parse_request(WolfConnCtx* ctx, char* raw, size_t len) {
    WolfArena* a = ctx->arena;

    /* Find header/body boundary */
    char* body_start = NULL;
    for (size_t i = 0; i + 3 < len; i++) {
        if (raw[i]=='\r' && raw[i+1]=='\n' && raw[i+2]=='\r' && raw[i+3]=='\n') {
            raw[i] = '\0';
            body_start = raw + i + 4;
            break;
        }
    }
    ctx->body = wolf_arena_strdup(a, body_start ? body_start : "");

    /* Parse request line */
    char* saveptr;
    char* line = strtok_r(raw, "\r\n", &saveptr);
    if (!line) return;

    char* l_save;
    char* method    = strtok_r(line, " ", &l_save);
    char* full_path = strtok_r(NULL, " ", &l_save);

    if (method)    ctx->method = wolf_arena_strdup(a, method);
    if (full_path) {
        char* q = strchr(full_path, '?');
        if (q) {
            *q = '\0';
            ctx->path  = wolf_arena_strdup(a, full_path);
            ctx->query = wolf_arena_strdup(a, q + 1);
        } else {
            ctx->path  = wolf_arena_strdup(a, full_path);
            ctx->query = wolf_arena_strdup(a, "");
        }
    }

    /* Parse headers */
    const char* upgrade_val = NULL;
    const char* ws_key_val  = NULL;

    while ((line = strtok_r(NULL, "\r\n", &saveptr))) {
        char* colon = strchr(line, ':');
        if (colon && ctx->header_count < 32) {
            *colon = '\0';
            char* val = colon + 1;
            while (*val == ' ') val++;
            ctx->header_keys[ctx->header_count] = wolf_arena_strdup(a, line);
            ctx->header_vals[ctx->header_count] = wolf_arena_strdup(a, val);
            if (strcasecmp(line, "Upgrade") == 0) upgrade_val = ctx->header_vals[ctx->header_count];
            if (strcasecmp(line, "Sec-WebSocket-Key") == 0) ws_key_val = ctx->header_vals[ctx->header_count];
            ctx->header_count++;
        }
    }

    if (upgrade_val && strcasecmp(upgrade_val, "websocket") == 0 && ws_key_val) {
        ctx->is_websocket = 1;
        ctx->ws_key = wolf_arena_strdup(a, ws_key_val);
    }
}

/* ================================================================
 * HTTP Response Writer
 * ================================================================ */

static void wolf_engine_send_response(WolfConnCtx* ctx) {
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

    char header_buf[4096];
    int hlen = snprintf(header_buf, sizeof(header_buf),
        "HTTP/1.1 %d %s\r\n", ctx->status_code, status_text);

    write(ctx->client_fd, header_buf, hlen);

    for (int i = 0; i < ctx->res_header_count; i++) {
        int n = snprintf(header_buf, sizeof(header_buf), "%s: %s\r\n",
                         ctx->res_header_keys[i], ctx->res_header_vals[i]);
        write(ctx->client_fd, header_buf, n);
    }

    int body_len = ctx->res_body ? (int)strlen(ctx->res_body) : 0;
    int n = snprintf(header_buf, sizeof(header_buf),
                     "Content-Length: %d\r\nConnection: keep-alive\r\n\r\n", body_len);
    write(ctx->client_fd, header_buf, n);

    if (body_len > 0)
        write(ctx->client_fd, ctx->res_body, body_len);
}

/* ================================================================
 * WebSocket handshake (same as original runtime)
 * ================================================================ */

static int wolf_engine_ws_handshake(WolfConnCtx* ctx) {
    if (!ctx->ws_key) return 0;

    char magic[] = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11";
    char combined[256];
    snprintf(combined, sizeof(combined), "%s%s", ctx->ws_key, magic);

    unsigned char hash[20];
    /* Use OpenSSL EVP via wolf_runtime if available */
    extern void wolf_crypto_init(void);
    wolf_crypto_init();

    EVP_MD_CTX* mdctx = EVP_MD_CTX_new();
    EVP_DigestInit_ex(mdctx, EVP_sha1(), NULL);
    EVP_DigestUpdate(mdctx, combined, strlen(combined));
    EVP_DigestFinal_ex(mdctx, hash, NULL);
    EVP_MD_CTX_free(mdctx);

    extern const char* wolf_base64_encode(const char* s);
    /* base64 encode the 20-byte hash */
    char b64_input[21];
    memcpy(b64_input, hash, 20);
    b64_input[20] = '\0';
    const char* accept_key = wolf_base64_encode(b64_input);

    char response[512];
    int rlen = snprintf(response, sizeof(response),
        "HTTP/1.1 101 Switching Protocols\r\n"
        "Upgrade: websocket\r\n"
        "Connection: Upgrade\r\n"
        "Sec-WebSocket-Accept: %s\r\n\r\n",
        accept_key);
    write(ctx->client_fd, response, rlen);
    return 1;
}

/* ================================================================
 * Per-Core Worker Thread
 * This is the hot path. Each thread:
 *   1. accept() on its own SO_REUSEPORT socket (kernel load-balances)
 *   2. parse request into arena memory
 *   3. call Wolf HTTP handler
 *   4. write response
 *   5. reset arena (O(1))
 * No mutex, no shared queue, no cross-core communication.
 * ================================================================ */

static volatile sig_atomic_t wolf_engine_shutdown_flag = 0;

static void wolf_engine_signal_handler(int sig) {
    (void)sig;
    wolf_engine_shutdown_flag = 1;
}

typedef struct {
    WolfCore*           core;
    wolf_http_handler_t http_handler;
    wolf_ws_handler_t   ws_handler;
} WolfCoreArgs;

/* Inline context table — per-core, no locking */
static __thread WolfConnCtx wolf_core_ctxs[WOLF_CORE_CTX_MAX];

static WolfConnCtx* wolf_core_alloc_ctx(WolfCore* core, int client_fd, WolfArena* arena) {
    for (int i = 0; i < WOLF_CORE_CTX_MAX; i++) {
        if (!wolf_core_ctxs[i].active) {
            memset(&wolf_core_ctxs[i], 0, sizeof(WolfConnCtx));
            wolf_core_ctxs[i].active      = 1;
            wolf_core_ctxs[i].client_fd   = client_fd;
            wolf_core_ctxs[i].core_id     = core->core_id;
            wolf_core_ctxs[i].status_code = 200;
            wolf_core_ctxs[i].arena       = arena;
            clock_gettime(CLOCK_MONOTONIC, &wolf_core_ctxs[i].started_at);
            return &wolf_core_ctxs[i];
        }
    }
    return NULL;
}

static void wolf_core_free_ctx(WolfConnCtx* ctx) {
    if (!ctx) return;
    if (ctx->arena) wolf_arena_reset(ctx->arena);
    ctx->active = 0;
}

static void* wolf_core_thread(void* arg) {
    WolfCoreArgs* args = (WolfCoreArgs*)arg;
    WolfCore*     core = args->core;

    /* Pin to core */
    wolf_pin_to_core(core->core_id);

    fprintf(stderr, "[WOLF-ENGINE] Core %d started (tid=%lu)\n",
            core->core_id, (unsigned long)pthread_self());

    /* Add server fd to sentinel for edge-triggered accept */
    wolf_sentinel_add(core->sentinel, core->server_fd, NULL, NULL);

    char read_buf[WOLF_MAX_REQUEST_SIZE];

    while (!wolf_engine_shutdown_flag) {
        /* Try to accept a connection */
        struct sockaddr_in client_addr;
        socklen_t client_len = sizeof(client_addr);

        int client_fd = accept(core->server_fd,
                               (struct sockaddr*)&client_addr, &client_len);

        if (client_fd < 0) {
            if (errno == EAGAIN || errno == EWOULDBLOCK) {
                /* No connection ready — poll for up to 1ms then retry */
                wolf_sentinel_poll(core->sentinel, 1);
                continue;
            }
            if (errno == EINTR) continue;
            perror("accept");
            continue;
        }

        /* Set receive timeout */
        struct timeval tv = { WOLF_REQUEST_TIMEOUT_SEC, 0 };
        setsockopt(client_fd, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv));

        /* TCP_NODELAY on client socket too */
        int opt = 1;
        setsockopt(client_fd, IPPROTO_TCP, TCP_NODELAY, &opt, sizeof(opt));

        /* Acquire arena from pool — O(1), no malloc */
        WolfArena* arena = wolf_arena_acquire(core->arena_pool);

        /* Allocate connection context in arena */
        WolfConnCtx* ctx = wolf_core_alloc_ctx(core, client_fd, arena);
        if (!ctx) {
            /* All slots full — 503 */
            const char* busy =
                "HTTP/1.1 503 Service Unavailable\r\n"
                "Content-Length: 0\r\n\r\n";
            write(client_fd, busy, strlen(busy));
            close(client_fd);
            wolf_arena_reset(arena);
            __atomic_fetch_add(&core->requests_total, 1, __ATOMIC_RELAXED);
            continue;
        }

        /* Read request */
        ssize_t bytes = read(client_fd, read_buf, sizeof(read_buf) - 1);
        if (bytes <= 0) {
            if (bytes < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) {
                /* Timeout */
                const char* timeout_resp =
                    "HTTP/1.1 408 Request Timeout\r\n"
                    "Content-Length: 0\r\nConnection: close\r\n\r\n";
                write(client_fd, timeout_resp, strlen(timeout_resp));
            }
            close(client_fd);
            wolf_core_free_ctx(ctx);
            continue;
        }
        read_buf[bytes] = '\0';

        /* Parse into arena memory */
        wolf_engine_parse_request(ctx, read_buf, bytes);

        /* WebSocket upgrade */
        if (ctx->is_websocket) {
            wolf_engine_ws_handshake(ctx);
            /* For now: hand off to the legacy WS poller */
            /* Phase 2: integrate into per-core event loop */
            wolf_core_free_ctx(ctx);
            /* Don't close fd — WS poller owns it */
            continue;
        }

        /* Set thread-local request context (legacy API compatibility) */
        int64_t ctx_id = (int64_t)(ctx - wolf_core_ctxs);
        wolf_set_current_context((void*)(intptr_t)ctx_id, (void*)(intptr_t)ctx_id);

        /* Call Wolf HTTP handler */
        __atomic_fetch_add(&core->requests_active, 1, __ATOMIC_RELAXED);
        if (args->http_handler) {
            wolf_req_arena_init();
            args->http_handler(ctx_id, ctx_id);
        }
        __atomic_fetch_sub(&core->requests_active, 1, __ATOMIC_RELAXED);

        /* Send response */
        wolf_engine_send_response(ctx);

        /* Stats */
        __atomic_fetch_add(&core->requests_total, 1, __ATOMIC_RELAXED);
        __atomic_fetch_add(&core->bytes_in,  bytes,                              __ATOMIC_RELAXED);
        __atomic_fetch_add(&core->bytes_out, ctx->res_body ? strlen(ctx->res_body) : 0, __ATOMIC_RELAXED);

        /* Close and release — arena_reset is O(1) pointer reset */
        close(client_fd);
        wolf_req_arena_flush();
        wolf_core_free_ctx(ctx);   /* resets arena */
    }

    fprintf(stderr, "[WOLF-ENGINE] Core %d shutting down (served %lld requests)\n",
            core->core_id, (long long)core->requests_total);

    free(args);
    return NULL;
}

/* ================================================================
 * Engine Lifecycle
 * ================================================================ */

WolfEngine* wolf_engine_create(int port, int core_count) {
    if (core_count <= 0) core_count = wolf_detect_nproc();
    /* Cap at 64 cores */
    if (core_count > 64) core_count = 64;

    WolfEngine* engine = (WolfEngine*)calloc(1, sizeof(WolfEngine));
    engine->port       = port;
    engine->core_count = core_count;
    engine->cores      = (WolfCore**)calloc(core_count, sizeof(WolfCore*));

    for (int i = 0; i < core_count; i++) {
        WolfCore* core = (WolfCore*)calloc(1, sizeof(WolfCore));
        core->core_id   = i;
        core->server_fd = wolf_create_server_socket(port);
        if (core->server_fd < 0) {
            fprintf(stderr, "[WOLF-ENGINE] Failed to create socket for core %d\n", i);
            free(core);
            engine->cores[i] = NULL;
            continue;
        }
        core->sentinel   = wolf_sentinel_create(i);
        core->arena_pool = wolf_arena_pool_create(i);
        engine->cores[i] = core;
    }

    return engine;
}

int wolf_engine_start(WolfEngine* engine,
                      wolf_http_handler_t handler,
                      wolf_ws_handler_t   ws_handler) {
    /* Install signal handlers */
    struct sigaction sa;
    memset(&sa, 0, sizeof(sa));
    sa.sa_handler = wolf_engine_signal_handler;
    sigemptyset(&sa.sa_mask);
    sigaction(SIGTERM, &sa, NULL);
    sigaction(SIGINT,  &sa, NULL);
    signal(SIGPIPE, SIG_IGN);

    printf("🐺 Wolf HTTP Engine — %d cores, port %d\n",
           engine->core_count, engine->port);
    printf("   Architecture: Thread-Per-Core + %s\n",
#if defined(WOLF_HAS_EPOLL)
           "epoll (Phase 1)"
#elif defined(WOLF_HAS_KQUEUE)
           "kqueue (Phase 1)"
#else
           "poll (fallback)"
#endif
    );
    printf("   Arena pool: %d × %dKB per core\n",
           WOLF_ARENA_POOL_SIZE, (int)(WOLF_ARENA_SLAB_SIZE / 1024));
    printf("   Send SIGTERM or Ctrl+C to shut down gracefully.\n");
    fflush(stdout);

    /* Start per-core threads */
    for (int i = 0; i < engine->core_count; i++) {
        WolfCore* core = engine->cores[i];
        if (!core) continue;

        WolfCoreArgs* args = (WolfCoreArgs*)malloc(sizeof(WolfCoreArgs));
        args->core         = core;
        args->http_handler = handler;
        args->ws_handler   = ws_handler;

        if (pthread_create(&core->thread, NULL, wolf_core_thread, args) != 0) {
            perror("pthread_create");
            free(args);
        }
    }

    /* Main thread acts as sysmon, sending SIGURG preemption signals to cores every 10ms */
    while (!wolf_engine_shutdown_flag) {
        usleep(10000);
        for (int i = 0; i < engine->core_count; i++) {
            WolfCore* core = engine->cores[i];
            if (core && core->thread) {
                pthread_kill(core->thread, SIGURG);
            }
        }
    }

    wolf_engine_shutdown(engine);
    return 0;
}

void wolf_engine_shutdown(WolfEngine* engine) {
    fprintf(stderr, "[WOLF-ENGINE] Shutdown initiated...\n");
    wolf_engine_shutdown_flag = 1;

    /* Wait for all core threads to exit */
    for (int i = 0; i < engine->core_count; i++) {
        WolfCore* core = engine->cores[i];
        if (!core) continue;
        pthread_join(core->thread, NULL);
        if (core->server_fd >= 0) close(core->server_fd);
    }

    fprintf(stderr, "[WOLF-ENGINE] All cores stopped.\n");
    wolf_engine_stats(engine);
}

void wolf_engine_stats(WolfEngine* engine) {
    int64_t total_requests = 0;
    int64_t total_bytes_in = 0;
    int64_t total_bytes_out = 0;

    fprintf(stderr, "\n[WOLF-ENGINE] Per-core stats:\n");
    for (int i = 0; i < engine->core_count; i++) {
        WolfCore* core = engine->cores[i];
        if (!core) continue;
        fprintf(stderr, "  Core %d: %lld requests, %lld B in, %lld B out\n",
                i,
                (long long)core->requests_total,
                (long long)core->bytes_in,
                (long long)core->bytes_out);
        total_requests  += core->requests_total;
        total_bytes_in  += core->bytes_in;
        total_bytes_out += core->bytes_out;
    }
    fprintf(stderr, "  Total:  %lld requests, %lld B in, %lld B out\n\n",
            (long long)total_requests,
            (long long)total_bytes_in,
            (long long)total_bytes_out);
}

void wolf_engine_destroy(WolfEngine* engine) {
    if (!engine) return;
    for (int i = 0; i < engine->core_count; i++) {
        WolfCore* core = engine->cores[i];
        if (!core) continue;
        wolf_sentinel_destroy(core->sentinel);
        wolf_arena_pool_destroy(core->arena_pool);
        free(core);
    }
    free(engine->cores);
    free(engine);
}

/* ================================================================
 * wolf_http_serve() replacement
 * Drop-in replacement for the original single-threaded wolf_http_serve.
 * Called from compiled Wolf programs: wolf_http_serve(port, handler)
 * ================================================================ */

void wolf_http_serve(int64_t port, void* handler_ptr) {
    wolf_http_handler_t handler = (wolf_http_handler_t)handler_ptr;

    extern void wolf_crypto_init(void);
    wolf_crypto_init();

    /* Validate mail config at startup if configured */
    if (WOLF_MAIL_FROM_EMAIL[0] || WOLF_MAIL_HOST[0]) {
        wolf_mailer_validate_config();
    }

    int core_count = WOLF_WORKER_THREADS > 0 ? WOLF_WORKER_THREADS : 0; /* 0 = auto */
    WolfEngine* engine = wolf_engine_create((int)port, core_count);

    wolf_engine_start(engine, handler, NULL);
    wolf_engine_destroy(engine);

    extern void wolf_db_pool_destroy(void);
    wolf_db_pool_destroy();

    fprintf(stderr, "[Wolf] Shutdown complete.\n");
    exit(0);
}

/* ================================================================
 * Phase 2 Stubs — WTask (to be filled in by wolf_scheduler.c)
 * These exist so the emitter can reference them without compiler errors.
 * ================================================================ */

WTask* wtask_create(WolfCore* core, int64_t req_id) {
    /* Phase 2: allocate from arena, set up stack */
    WolfArena* arena = wolf_arena_acquire(core->arena_pool);
    WTask* t = (WTask*)wolf_arena_alloc(arena, sizeof(WTask));
    t->state   = WTASK_STATE_READY;
    t->core_id = core->core_id;
    t->arena   = arena;
    t->req_id  = req_id;
    t->res_id  = req_id;
    return t;
}

void wtask_yield(WTask* task) {
    /* Phase 2: save continuation, return control to executor */
    (void)task;
    /* Currently a no-op — tasks run to completion */
}

void wtask_complete(WTask* task) {
    if (!task) return;
    if (task->arena) wolf_arena_reset(task->arena);
    task->state = WTASK_STATE_COMPLETE;
}

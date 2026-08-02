/* ================================================================
 * Wolf CGI I/O Adapter (wolf deploy)
 * ================================================================
 * This file implements the CGI execution model for Wolf, completely
 * bypassing the standard HTTP server's io_uring, epoll, and thread
 * pool architecture. It translates standard OS environment variables
 * (REQUEST_METHOD, QUERY_STRING) and stdin into a WolfConnCtx,
 * invokes the compiled user HTTP handler once, writes the HTTP
 * response to stdout, and exits.
 * 
 * Since CGI fundamentally requires an OS environment, this file
 * is strictly included #ifndef WOLF_FREESTANDING in wolf_runtime.c.
 * ================================================================ */

#ifndef WOLF_FREESTANDING

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

/* Declaration of internal arena init/destroy functions */
extern void wolf_req_arena_init(void);
extern void wolf_req_arena_flush(void);

/* Single global context for CGI mode */
static WolfConnCtx g_cgi_ctx;

/* Local helper to map CGI HTTP_* env vars into the context headers */
static void wolf_cgi_load_header(WolfConnCtx* ctx, const char* env_key, const char* header_name) {
    if (ctx->header_count >= 32) return;
    const char* val = getenv(env_key);
    if (val) {
        ctx->header_keys[ctx->header_count] = (char*)header_name;
        ctx->header_vals[ctx->header_count] = (char*)val;
        
        /* Compute simple hash for fast lookup */
        uint32_t hash = 5381;
        for (int i = 0; header_name[i]; i++) {
            char c = header_name[i];
            if (c >= 'A' && c <= 'Z') c += 32; /* tolower */
            hash = ((hash << 5) + hash) + c;
        }
        int bucket = hash % WOLF_HEADER_HASH_SLOTS;
        
        /* Linear probing */
        for (int i = 0; i < WOLF_HEADER_HASH_SLOTS; i++) {
            int probe = (bucket + i) % WOLF_HEADER_HASH_SLOTS;
            if (ctx->header_htab[probe] == -1) {
                ctx->header_htab[probe] = ctx->header_count;
                break;
            }
        }
        ctx->header_count++;
    }
}

void wolf_cgi_serve(void* handler_ptr) {
    if (!handler_ptr) exit(1);
    
    /* Ensure arena is initialized for the single request */
    wolf_req_arena_init();
    
    memset(&g_cgi_ctx, 0, sizeof(WolfConnCtx));
    memset(g_cgi_ctx.header_htab, -1, sizeof(g_cgi_ctx.header_htab));
    
    g_cgi_ctx.active = 1;
    g_cgi_ctx.request_id = 0; /* Only 1 request in CGI */
    g_cgi_ctx.status_code = 200;
    
    /* Set up global watchdog pointer (used by HTTP APIs to find ctx) */
    extern __thread void* wolf_active_ctx;
    wolf_active_ctx = &g_cgi_ctx;
    
    /* 1. Map Core CGI Environment Variables */
    char* method = getenv("REQUEST_METHOD");
    char* path = getenv("PATH_INFO");
    if (!path) path = getenv("SCRIPT_NAME"); /* Fallback if no path_info */
    if (!path) path = "/";
    char* query = getenv("QUERY_STRING");
    
    g_cgi_ctx.method = method ? method : "GET";
    g_cgi_ctx.path = path;
    g_cgi_ctx.query = query ? query : "";
    
    /* 2. Map Standard HTTP Headers */
    wolf_cgi_load_header(&g_cgi_ctx, "HTTP_USER_AGENT", "User-Agent");
    wolf_cgi_load_header(&g_cgi_ctx, "HTTP_ACCEPT", "Accept");
    wolf_cgi_load_header(&g_cgi_ctx, "HTTP_HOST", "Host");
    wolf_cgi_load_header(&g_cgi_ctx, "HTTP_AUTHORIZATION", "Authorization");
    wolf_cgi_load_header(&g_cgi_ctx, "HTTP_X_FORWARDED_FOR", "X-Forwarded-For");
    wolf_cgi_load_header(&g_cgi_ctx, "CONTENT_TYPE", "Content-Type");
    wolf_cgi_load_header(&g_cgi_ctx, "CONTENT_LENGTH", "Content-Length");
    
    /* 3. Read Request Body (if any) */
    const char* content_length_str = getenv("CONTENT_LENGTH");
    if (content_length_str) {
        int length = atoi(content_length_str);
        if (length > 0 && length <= WOLF_MAX_REQUEST_SIZE) {
            char* body = (char*)malloc(length + 1);
            if (body) {
                size_t read_bytes = 0;
                while (read_bytes < length) {
                    ssize_t n = read(STDIN_FILENO, body + read_bytes, length - read_bytes);
                    if (n <= 0) break;
                    read_bytes += n;
                }
                body[read_bytes] = '\0';
                g_cgi_ctx.body = body;
            }
        }
    }
    
    /* 4. Execute the user's compiled HTTP handler */
    wolf_closure_t* closure = (wolf_closure_t*)handler_ptr;
    typedef void* (*wolf_closure_fn_t)(void* env, int64_t req_id, int64_t res_id);
    wolf_closure_fn_t fn = (wolf_closure_fn_t)closure->fn;
    
    if (wolf_closure_valid(closure)) {
        if (setjmp(g_cgi_ctx.oom_jump) == 0) {
            fn(closure->env, 0, 0);
        } else {
            g_cgi_ctx.status_code = 500;
            g_cgi_ctx.res_body = "500 Internal Server Error (OOM)";
        }
    } else {
        g_cgi_ctx.status_code = 500;
    }
    
    /* 5. Construct and write the HTTP Response to stdout */
    /* Write Status Line */
    printf("Status: %d\r\n", g_cgi_ctx.status_code);
    
    /* Write Headers */
    int has_content_type = 0;
    for (int i = 0; i < g_cgi_ctx.res_header_count; i++) {
        if (strcasecmp(g_cgi_ctx.res_header_keys[i], "Content-Type") == 0) {
            has_content_type = 1;
        }
        printf("%s: %s\r\n", g_cgi_ctx.res_header_keys[i], g_cgi_ctx.res_header_vals[i]);
    }
    
    /* Default Content-Type if none provided */
    if (!has_content_type) {
        printf("Content-Type: text/plain\r\n");
    }
    
    /* Add Content-Length if body exists */
    if (g_cgi_ctx.res_body) {
        printf("Content-Length: %zu\r\n", strlen(g_cgi_ctx.res_body));
    } else {
        printf("Content-Length: 0\r\n");
    }
    
    /* End of Headers */
    printf("\r\n");
    fflush(stdout);
    
    /* Write Body */
    if (g_cgi_ctx.res_body) {
        size_t len = strlen(g_cgi_ctx.res_body);
        size_t written = 0;
        while (written < len) {
            ssize_t w = write(STDOUT_FILENO, g_cgi_ctx.res_body + written, len - written);
            if (w <= 0) break;
            written += w;
        }
    }
    
    /* 6. Cleanup */
    if (g_cgi_ctx.body) free(g_cgi_ctx.body);
    wolf_active_ctx = NULL;
    wolf_req_arena_flush();
    
    /* CGI is a single-shot execution, gracefully exit */
    exit(0);
}
#endif /* WOLF_FREESTANDING */

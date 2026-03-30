/*
 * Wolf Runtime Library
 * Provides core functions that compiled Wolf programs link against.
 * Compiled with: clang -c wolf_runtime.c -o wolf_runtime.o
 *
 * Optional compile-time flags:
 *   -DWOLF_REDIS_ENABLED          link against hiredis for real Redis support
 *   -DWOLF_DB_POSTGRES            use libpq instead of MySQL
 *   -DWOLF_DB_MSSQL               use MSSQL mock driver
 *   -DWOLF_DEBUG                  enable verbose pool/arena logging to stderr
 *   -DWOLF_MAX_CONCURRENT_REQUESTS=N  override default of 1024
 *   -DWOLF_MAX_REQUEST_SIZE=N         override default of 65536
 */

#define _GNU_SOURCE
#define _POSIX_C_SOURCE 200809L

#include "wolf_runtime.h"
#include "wolf_config_runtime.h"
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <ctype.h>
#include <stdarg.h>
#include <strings.h>
#include <time.h>
#include <math.h>
#include <sys/stat.h>
#include <errno.h>

/* --- Server / OS-dependent headers (stripped on bare-metal targets) --- */
#ifndef WOLF_FREESTANDING
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <unistd.h>
#include <pthread.h>
#include <curl/curl.h>
#endif /* WOLF_FREESTANDING */

/* ========== Database Configuration & Headers ========== */
#if defined(WOLF_DB_POSTGRES)
#include <libpq-fe.h>
typedef PGconn  WolfDBConn;
typedef PGresult WolfDBRes;
#define WOLF_DB_PING(conn) (PQstatus(conn) == CONNECTION_OK ? 0 : 1)
#define WOLF_DB_CLOSE(conn) PQfinish(conn)
#elif defined(WOLF_DB_MSSQL)
typedef void WolfDBConn;
typedef void WolfDBRes;
#define WOLF_DB_PING(conn) 0
#define WOLF_DB_CLOSE(conn) do {} while(0)
#else
#include <mysql/mysql.h>
typedef MYSQL     WolfDBConn;
typedef MYSQL_RES WolfDBRes;
#define WOLF_DB_PING(conn) mysql_ping(conn)
#define WOLF_DB_CLOSE(conn) mysql_close(conn)
#endif

#include <time.h>
#include <math.h>
#include <sys/stat.h>
#include <errno.h>
#include <openssl/evp.h>
#include <openssl/hmac.h>
#include <openssl/rand.h>
#include <openssl/rsa.h>
#include <openssl/pem.h>
#include <openssl/err.h>
#include <sodium.h>
#include <regex.h>

/* ========== Configurable Runtime Limits ========== */
#ifndef WOLF_MAX_CONCURRENT_REQUESTS
#define WOLF_MAX_CONCURRENT_REQUESTS 1024
#endif
#define MAX_CONCURRENT_REQUESTS 512

#ifndef WOLF_MAX_REQUEST_SIZE
#define WOLF_MAX_REQUEST_SIZE 65536
#endif

#ifndef WOLF_DEFINE_MAX
#define WOLF_DEFINE_MAX 256
#endif

/*
 * Graceful Shutdown
 *
 * wolf_shutdown_requested — set to 1 by SIGTERM/SIGINT handler.
 * wolf_active_requests    — in-flight request counter.
 *
 * wolf_http_serve() installs signal handlers and checks the flag
 * after every accept(). When set, it waits until
 * wolf_active_requests == 0, calls wolf_db_pool_destroy(), then
 * exits with code 0.
 */
static volatile sig_atomic_t wolf_shutdown_requested = 0;
static volatile sig_atomic_t wolf_active_requests    = 0;
#ifndef WOLF_FREESTANDING
static pthread_mutex_t wolf_drain_mutex = PTHREAD_MUTEX_INITIALIZER;
static pthread_cond_t  wolf_drain_cond  = PTHREAD_COND_INITIALIZER;
#endif /* WOLF_FREESTANDING */


/* CLI Arguments */
static int    wolf_argc_val = 0;
static char** wolf_argv_val = NULL;

void wolf_init_args(int argc, char** argv) {
    wolf_argc_val = argc;
    wolf_argv_val = argv;
    #ifdef WOLF_DEBUG
    fprintf(stderr, "[WOLF-CLI] Initialized with argc=%d argv=%p\n", argc, (void*)argv);
    if (argc > 0) fprintf(stderr, "[WOLF-CLI] argv[0]=%s\n", argv[0]);
    #endif
}

int64_t wolf_argc(void) {
    return (int64_t)wolf_argc_val;
}

const char* wolf_argv(int64_t index) {
    if (index < 0 || index >= wolf_argc_val) {
        #ifdef WOLF_DEBUG
        fprintf(stderr, "[WOLF-CLI] argv index %ld out of bounds (0-%d)\n", (long)index, wolf_argc_val-1);
        #endif
        return "";
    }
    const char* val = wolf_argv_val[index];
    #ifdef WOLF_DEBUG
    fprintf(stderr, "[WOLF-CLI] argv[%ld] = %s\n", (long)index, val ? val : "NULL");
    #endif
    return val ? val : "";
}

static void wolf_signal_handler(int sig) {
    (void)sig;
    wolf_shutdown_requested = 1;
    /* Write a visible message. write() is async-signal-safe. */
    const char msg[] = "\n[Wolf] Shutdown signal received — draining requests...\n";
    write(STDERR_FILENO, msg, sizeof(msg) - 1);
}

/* Forward declarations */
static __thread int64_t wolf_current_req_id = -1;
static __thread int64_t wolf_current_res_id = -1;  /* -1 = no active HTTP context → use stdout */
void wolf_http_res_write(int64_t res_id, const char* body);
static char* wolf_base64_encode_bin(const char* data, size_t len);

static void free_http_context(int id);

/* --- HTTP / WebSocket Context Types --- */
#define WOLF_MAX_UPLOADS 8
typedef struct {
    const char* field_name;
    const char* filename;
    const char* content_type;
    const char* data;
    size_t      size;
} wolf_upload_t;

typedef struct {
    int    active;
    int    client_fd;
    char*  method;
    char*  path;
    char*  query;
    char*  body;
    char*  header_keys[32];
    char*  header_vals[32];
    int    header_count;
    int    status_code;
    char*  res_header_keys[32];
    char*  res_header_vals[32];
    int    res_header_count;
    char*  res_body;
    wolf_upload_t uploads[WOLF_MAX_UPLOADS];
    int           upload_count;
    int           is_websocket;
    char*         ws_key;

    /* WebSocket state for non-blocking poller */
    int           ws_state;       /* 0: header, 1: payload */
    unsigned char ws_header[10];
    int           ws_header_pos;
    int           ws_header_len;
    uint64_t      ws_payload_len;
    char*         ws_payload_buf;
    uint64_t      ws_payload_pos;
    unsigned char ws_mask[4];
    int           ws_masked;
    int           ws_opcode;
} wolf_http_context_t;

typedef void (*wolf_http_handler_t)(int64_t req_id, int64_t res_id);
typedef void (*wolf_ws_handler_t)(int64_t req_id, const char* message);

static wolf_http_context_t http_contexts[MAX_CONCURRENT_REQUESTS];
static pthread_mutex_t http_mutex = PTHREAD_MUTEX_INITIALIZER;
static wolf_http_handler_t global_wolf_handler = NULL;
static wolf_ws_handler_t   global_ws_handler   = NULL;

/* ================================================================ *
 * Unified Worker Pool & Task Queue                                 *
 * ================================================================ */

typedef enum {
    WOLF_TASK_EMPTY,
    WOLF_TASK_HTTP,
    WOLF_TASK_WS_EVENT
} wolf_task_type_t;

typedef struct {
    wolf_task_type_t type;
    int64_t          id;      /* req_id or context_id */
    char*            payload; /* for WS_EVENT */
} wolf_task_t;

#ifndef WOLF_WORKER_THREADS
#define WOLF_WORKER_THREADS 16
#endif

#define WOLF_TASK_QUEUE_SIZE 4096

static wolf_task_t     wolf_task_queue[WOLF_TASK_QUEUE_SIZE];
static int             wolf_task_head = 0;
static int             wolf_task_tail = 0;
static int             wolf_task_count = 0;
static pthread_mutex_t wolf_task_mutex = PTHREAD_MUTEX_INITIALIZER;
static pthread_cond_t  wolf_task_cond  = PTHREAD_COND_INITIALIZER;
static pthread_t       wolf_worker_pool[WOLF_WORKER_THREADS];

static void wolf_task_push(wolf_task_t task) {
    pthread_mutex_lock(&wolf_task_mutex);
    if (wolf_task_count < WOLF_TASK_QUEUE_SIZE) {
        wolf_task_queue[wolf_task_tail] = task;
        wolf_task_tail = (wolf_task_tail + 1) % WOLF_TASK_QUEUE_SIZE;
        wolf_task_count++;
        pthread_cond_signal(&wolf_task_cond);
    }
    pthread_mutex_unlock(&wolf_task_mutex);
}

static wolf_task_t wolf_task_pop(void) {
    pthread_mutex_lock(&wolf_task_mutex);
    while (wolf_task_count == 0 && !wolf_shutdown_requested) {
        pthread_cond_wait(&wolf_task_cond, &wolf_task_mutex);
    }
    wolf_task_t task = {WOLF_TASK_EMPTY, 0, NULL};
    if (wolf_task_count > 0) {
        task = wolf_task_queue[wolf_task_head];
        wolf_task_head = (wolf_task_head + 1) % WOLF_TASK_QUEUE_SIZE;
        wolf_task_count--;
    }
    pthread_mutex_unlock(&wolf_task_mutex);
    return task;
}

static void* http_worker(void* arg); /* Forward ref */

static void* wolf_worker_thread_func(void* arg) {
    (void)arg;
#if !defined(WOLF_DB_POSTGRES) && !defined(WOLF_DB_MSSQL)
    mysql_thread_init();
#endif
    while (!wolf_shutdown_requested) {
        wolf_task_t task = wolf_task_pop();
        if (task.type == WOLF_TASK_HTTP) {
            http_worker((void*)(intptr_t)task.id);
        } else if (task.type == WOLF_TASK_WS_EVENT) {
            if (global_ws_handler) {
                wolf_req_arena_init();
                wolf_set_current_context((void*)(intptr_t)task.id, (void*)(intptr_t)task.id);
                global_ws_handler(task.id, task.payload);
                wolf_req_arena_flush();
            }
            if (task.payload) free(task.payload);
        }
    }
#if !defined(WOLF_DB_POSTGRES) && !defined(WOLF_DB_MSSQL)
    mysql_thread_end();
#endif
    return NULL;
}

static void wolf_worker_pool_init(void) {
    for (int i = 0; i < WOLF_WORKER_THREADS; i++) {
        pthread_create(&wolf_worker_pool[i], NULL, wolf_worker_thread_func, NULL);
        pthread_detach(wolf_worker_pool[i]);
    }
}

/* --- Multi-platform Poller (epoll / kqueue / poll) --- */
#ifndef WOLF_FREESTANDING
#if defined(__linux__)
#include <sys/epoll.h>
#define WOLF_USE_EPOLL
#elif defined(__APPLE__)
#include <sys/event.h>
#define WOLF_USE_KQUEUE
#else
#include <poll.h>
#define WOLF_USE_POLL
#endif
#include <fcntl.h>

static int wolf_ws_poller_fd = -1;
static pthread_t wolf_ws_poller_thread_id;

static void wolf_ws_poller_init(void) {
#ifdef WOLF_USE_EPOLL
    wolf_ws_poller_fd = epoll_create1(0);
#elif defined(WOLF_USE_KQUEUE)
    wolf_ws_poller_fd = kqueue();
#endif
#ifdef WOLF_DEBUG
    fprintf(stderr, "[WOLF-WS] Poller initialized\n");
#endif
}

static void wolf_ws_poller_add(int fd, int id) {
    if (wolf_ws_poller_fd < 0) return;
    /* Set non-blocking */
    int flags = fcntl(fd, F_GETFL, 0);
    fcntl(fd, F_SETFL, flags | O_NONBLOCK);

#ifdef WOLF_USE_EPOLL
    struct epoll_event ev;
    ev.events = EPOLLIN;
    ev.data.u64 = (uint64_t)id;
    epoll_ctl(wolf_ws_poller_fd, EPOLL_CTL_ADD, fd, &ev);
#elif defined(WOLF_USE_KQUEUE)
    struct kevent ev;
    EV_SET(&ev, fd, EVFILT_READ, EV_ADD | EV_ENABLE, 0, 0, (void*)(intptr_t)id);
    kevent(wolf_ws_poller_fd, &ev, 1, NULL, 0, NULL);
#elif defined(WOLF_USE_POLL)
    /* poll() doesn't use a persistent fd, we'll manage it in the thread loop */
    (void)id;
#endif
#ifdef WOLF_DEBUG
    fprintf(stderr, "[WOLF-WS] Added fd=%d to poller\n", fd);
#endif
}

static void wolf_ws_handle_read_event(int id); /* Forward ref */

static void* wolf_ws_poller_thread_func(void* arg) {
    (void)arg;
    #ifdef WOLF_USE_EPOLL
    struct epoll_event events[64];
    while (!wolf_shutdown_requested) {
        int n = epoll_wait(wolf_ws_poller_fd, events, 64, 1000);
        for (int i = 0; i < n; i++) {
            wolf_ws_handle_read_event((int)events[i].data.u64);
        }
    }
    #elif defined(WOLF_USE_KQUEUE)
    struct kevent events[64];
    struct timespec timeout = {1, 0};
    while (!wolf_shutdown_requested) {
        int n = kevent(wolf_ws_poller_fd, NULL, 0, events, 64, &timeout);
        for (int i = 0; i < n; i++) {
            wolf_ws_handle_read_event((int)(intptr_t)events[i].udata);
        }
    }
    #elif defined(WOLF_USE_POLL)
    struct pollfd fds[MAX_CONCURRENT_REQUESTS];
    int ids[MAX_CONCURRENT_REQUESTS];
    while (!wolf_shutdown_requested) {
        int count = 0;
        pthread_mutex_lock(&http_mutex);
        for (int i = 0; i < MAX_CONCURRENT_REQUESTS; i++) {
            if (http_contexts[i].active && http_contexts[i].is_websocket && http_contexts[i].client_fd > 0) {
                fds[count].fd = http_contexts[i].client_fd;
                fds[count].events = POLLIN;
                ids[count] = i;
                count++;
            }
        }
        pthread_mutex_unlock(&http_mutex);
        
        if (count == 0) { usleep(100000); continue; }

        int n = poll(fds, count, 1000);
        if (n > 0) {
            for (int i = 0; i < count; i++) {
                if (fds[i].revents & POLLIN) {
                    wolf_ws_handle_read_event(ids[i]);
                }
            }
        }
    }
    #endif
    return NULL;
}

static void wolf_ws_handle_read_event(int id) {
    if (id < 0 || id >= MAX_CONCURRENT_REQUESTS) return;
    wolf_http_context_t* ctx = &http_contexts[id];
    int fd = ctx->client_fd;

    if (ctx->ws_state == 0) { /* Reading Header */
        if (ctx->ws_header_len == 0) ctx->ws_header_len = 2; /* Min header */
        while (ctx->ws_header_pos < ctx->ws_header_len) {
            ssize_t r = read(fd, ctx->ws_header + ctx->ws_header_pos, ctx->ws_header_len - ctx->ws_header_pos);
            if (r <= 0) {
                if (r < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) return;
                goto close_ws; /* Error or EOF */
            }
            ctx->ws_header_pos += r;
        }

        /* Parse first 2 bytes */
        if (ctx->ws_header_pos >= 2 && ctx->ws_header_len == 2) {
            ctx->ws_opcode = ctx->ws_header[0] & 0x0F;
            ctx->ws_masked = (ctx->ws_header[1] & 0x80) != 0;
            ctx->ws_payload_len = ctx->ws_header[1] & 0x7F;

            if (ctx->ws_payload_len == 126) ctx->ws_header_len = 4;
            else if (ctx->ws_payload_len == 127) ctx->ws_header_len = 10;
            
            if (ctx->ws_masked) ctx->ws_header_len += 4;
            
            /* If header is now longer, keep reading */
            if (ctx->ws_header_pos < ctx->ws_header_len) return;
        }

        /* Header fully read */
        if (ctx->ws_header_pos == ctx->ws_header_len) {
            int pos = 2;
            if (ctx->ws_payload_len == 126) {
                ctx->ws_payload_len = (ctx->ws_header[2] << 8) | ctx->ws_header[3];
                pos = 4;
            } else if (ctx->ws_payload_len == 127) {
                ctx->ws_payload_len = 0;
                for (int i = 0; i < 8; i++) ctx->ws_payload_len = (ctx->ws_payload_len << 8) | ctx->ws_header[2+i];
                pos = 10;
            }
            if (ctx->ws_masked) {
                memcpy(ctx->ws_mask, ctx->ws_header + pos, 4);
            }
            
            if (ctx->ws_payload_len > 10 * 1024 * 1024) goto close_ws; /* 10MB safety */
            
            ctx->ws_payload_buf = malloc(ctx->ws_payload_len + 1);
            if (!ctx->ws_payload_buf) goto close_ws;
            ctx->ws_payload_pos = 0;
            ctx->ws_state = 1; /* Switch to payload */
        }
    }

    if (ctx->ws_state == 1) { /* Reading Payload */
        while (ctx->ws_payload_pos < ctx->ws_payload_len) {
            ssize_t r = read(fd, ctx->ws_payload_buf + ctx->ws_payload_pos, ctx->ws_payload_len - ctx->ws_payload_pos);
            if (r <= 0) {
                if (r < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) return;
                goto close_ws;
            }
            ctx->ws_payload_pos += r;
        }

        /* Payload fully read */
        if (ctx->ws_masked) {
            for (size_t i = 0; i < ctx->ws_payload_len; i++) ctx->ws_payload_buf[i] ^= ctx->ws_mask[i % 4];
        }
        ctx->ws_payload_buf[ctx->ws_payload_len] = '\0';

        if (ctx->ws_opcode == 0x08) goto close_ws; /* Close frame */

        if (ctx->ws_opcode == 0x01) { /* Text frame */
            wolf_task_t task = {WOLF_TASK_WS_EVENT, (int64_t)id, ctx->ws_payload_buf};
            wolf_task_push(task);
            ctx->ws_payload_buf = NULL; /* Ownership transferred to task */
        } else {
            free(ctx->ws_payload_buf);
            ctx->ws_payload_buf = NULL;
        }

        /* Reset for next frame */
        ctx->ws_state = 0;
        ctx->ws_header_pos = 0;
        ctx->ws_header_len = 2;
    }
    return;

close_ws:
    if (ctx->ws_payload_buf) free(ctx->ws_payload_buf);
    ctx->ws_payload_buf = NULL;
    close(fd);
    free_http_context(id);
}

static void wolf_ws_poller_start(void) {
    pthread_create(&wolf_ws_poller_thread_id, NULL, wolf_ws_poller_thread_func, NULL);
    pthread_detach(wolf_ws_poller_thread_id);
}
#endif /* WOLF_FREESTANDING */


/* ================================================================ *
 * Per-Request Memory Arena                                         *
 *                                                                  *
 * All allocations that should live exactly as long as one HTTP     *
 * request register themselves here via wolf_req_alloc() /          *
 * wolf_req_strdup(). At end of request, wolf_req_arena_flush()     *
 * frees everything in O(n) — one call, no leaks.                  *
 *                                                                  *
 * Thread-safe: each worker thread has its own arena (__thread).   *
 *                                                                  *
 * IMPORTANT: wolf_req_strdup / wolf_req_alloc must ONLY be used   *
 * for data that lives for the duration of a single request.       *
 * Application-lifetime data (wolf_defines, pool credentials)      *
 * must use plain malloc / strdup instead.                         *
 * ================================================================ */

#define WOLF_ARENA_CHUNK 256

typedef struct {
    void   **ptrs;
    int      count;
    int      cap;
    int      active;
} WolfReqArena;

static __thread WolfReqArena wolf_req_arena = {NULL, 0, 0, 0};

void wolf_req_arena_init(void) {
    wolf_req_arena.count  = 0;
    wolf_req_arena.active = 1;
}

void* wolf_req_alloc_register(void* ptr) {
    if (!ptr || !wolf_req_arena.active) return ptr;
    if (wolf_req_arena.count >= wolf_req_arena.cap) {
        int      new_cap   = wolf_req_arena.cap + WOLF_ARENA_CHUNK;
        void**   new_ptrs  = realloc(wolf_req_arena.ptrs, new_cap * sizeof(void*));
        if (!new_ptrs) {
            /* OOM in arena management — we must free the ptr and return NULL
               to prevent a leak since we can't track it anymore. */
            if (ptr) free(ptr);
            return NULL;
        }
        wolf_req_arena.ptrs = new_ptrs;
        wolf_req_arena.cap  = new_cap;
    }
    wolf_req_arena.ptrs[wolf_req_arena.count++] = ptr;
    return ptr;
}

/* Helper for resizing safely within the request arena.
 * Since we do not store metadata size block easily, we'll manually implement reallocation
 * explicitly where needed. For things like simple string reallocs in json decoder,
 * we can just pass the old_size.
 */
void* wolf_req_realloc(void* old_ptr, size_t old_size, size_t new_size) {
    if (!old_ptr) return wolf_req_alloc(new_size);
    if (new_size == 0) return NULL;
    void* new_ptr = wolf_req_alloc(new_size);
    if (new_ptr) {
        memcpy(new_ptr, old_ptr, old_size < new_size ? old_size : new_size);
    }
    return new_ptr;
}

static inline char* json_decode_realloc(char* old, size_t old_sz, size_t new_sz) {
    return (char*)wolf_req_realloc(old, old_sz, new_sz);
}

void* wolf_req_alloc(size_t sz) {
    void* p = malloc(sz);
    if (p) memset(p, 0, sz);
    return wolf_req_alloc_register(p);
}

char* wolf_req_strdup(const char* s) {
    if (!s) return NULL;
    char* p = strdup(s);
    return (char*)wolf_req_alloc_register(p);
}

void wolf_req_arena_flush(void) {
    wolf_req_arena.active = 0;
    for (int i = 0; i < wolf_req_arena.count; i++) {
        if (wolf_req_arena.ptrs[i]) {
            free(wolf_req_arena.ptrs[i]);
            wolf_req_arena.ptrs[i] = NULL;
        }
    }
    wolf_req_arena.count = 0;
}

static int wolf_curl_inited = 0;
static void wolf_ensure_curl(void) {
    if (!wolf_curl_inited) {
        curl_global_init(CURL_GLOBAL_DEFAULT);
        wolf_curl_inited = 1;
    }
}


/* ================================================================ *
 * HTTP Client (libcurl)                                            *
 * ================================================================ */

typedef struct {
    char*  data;
    size_t size;
} WolfHTTPClientResult;

static size_t wolf_http_client_write_cb(void* contents, size_t size, size_t nmemb, void* userp) {
    size_t realsize = size * nmemb;
    WolfHTTPClientResult* res = (WolfHTTPClientResult*)userp;

    char* ptr = realloc(res->data, res->size + realsize + 1);
    if (!ptr) return 0;

    res->data = ptr;
    memcpy(&(res->data[res->size]), contents, realsize);
    res->size += realsize;
    res->data[res->size] = 0;

    return realsize;
}


/* ========== Debug Logging ========== */
#ifdef WOLF_DEBUG
#define WOLF_LOG(...) fprintf(stderr, __VA_ARGS__)
#else
#define WOLF_LOG(...) ((void)0)
#endif

/* ========== Print ========== */

void wolf_print_str(const char* s) {
    if (!s) return;
    if (wolf_current_res_id >= 0) {
        wolf_http_res_write(wolf_current_res_id, s);
    } else {
        printf("%s", s);
    }
}

void wolf_print_int(int64_t n) {
    printf("%lld", (long long)n);
}

void wolf_print_float(double f) {
    char buf[64];
    snprintf(buf, sizeof(buf), "%g", f);
    printf("%s", buf);
}

void wolf_print_bool(int b) {
    printf("%s", b ? "true" : "false");
}

void wolf_print_nil(void) {
    printf("nil");
}

void wolf_println(void) {
    if (wolf_current_res_id < 0) {
        printf("\n");
    }
}

/* --- Output & Display (Wolf Way) --- */

void wolf_say(const char* s) {
    if (!s) return;
    if (wolf_current_res_id >= 0) {
        wolf_http_res_write(wolf_current_res_id, s);
        wolf_http_res_write(wolf_current_res_id, "\n");
    } else {
        printf("%s\n", s);
    }
}

void wolf_show(void* variable) {
    if (wolf_current_res_id >= 0) {
        if (variable) {
            char buf[32];
            snprintf(buf, sizeof(buf), "%p", variable);
            wolf_http_res_write(wolf_current_res_id, buf);
        } else {
            wolf_http_res_write(wolf_current_res_id, "nil");
        }
    } else {
        if (variable) printf("%p\n", variable);
        else printf("nil\n");
    }
}

void wolf_inspect(void* variable) {
    if (wolf_current_res_id >= 0) {
        char buf[64];
        if (variable) snprintf(buf, sizeof(buf), "[ptr] %p", variable);
        else snprintf(buf, sizeof(buf), "[nil] null");
        wolf_http_res_write(wolf_current_res_id, buf);
    } else {
        if (variable) printf("[ptr] %p\n", variable);
        else printf("[nil] null\n");
    }
}

/* ========== String Operations ========== */

const char* wolf_string_concat(const char* a, const char* b) {
    if (!a) a = "";
    if (!b) b = "";
    size_t la = strlen(a);
    size_t lb = strlen(b);
    char* result = (char*)wolf_req_alloc(la + lb + 1);
    memcpy(result, a, la);
    memcpy(result + la, b, lb);
    result[la + lb] = '\0';
    return result;
}

int64_t wolf_strings_length(const char* s) {
    if (!s) return 0;
    return (int64_t)strlen(s);
}

const char* wolf_string_upper(const char* s) {
    if (!s) return "";
    size_t len = strlen(s);
    char* result = (char*)wolf_req_alloc(len + 1);
    for (size_t i = 0; i < len; i++)
        result[i] = (char)toupper((unsigned char)s[i]);
    result[len] = '\0';
    return result;
}

const char* wolf_string_lower(const char* s) {
    if (!s) return "";
    size_t len = strlen(s);
    char* result = (char*)wolf_req_alloc(len + 1);
    for (size_t i = 0; i < len; i++)
        result[i] = (char)tolower((unsigned char)s[i]);
    result[len] = '\0';
    return result;
}

const char* wolf_string_trim(const char* s) {
    if (!s) return "";
    while (*s && isspace((unsigned char)*s)) s++;
    size_t len = strlen(s);
    while (len > 0 && isspace((unsigned char)s[len - 1])) len--;
    char* result = (char*)wolf_req_alloc(len + 1);
    memcpy(result, s, len);
    result[len] = '\0';
    return result;
}

/* ========== Math Operations ========== */

double wolf_math_abs(double v)    { return v < 0 ? -v : v; }
double wolf_math_ceil(double v)   { int64_t i = (int64_t)v; return (v > i) ? (double)(i+1) : (double)i; }
double wolf_math_floor(double v)  { int64_t i = (int64_t)v; return (v < i) ? (double)(i-1) : (double)i; }
double wolf_math_max(double a, double b) { return a > b ? a : b; }
double wolf_math_min(double a, double b) { return a < b ? a : b; }
int64_t wolf_math_random(int64_t min, int64_t max) {
    srand((unsigned int)(time(NULL)));
    return min + rand() % ((max + 1) - min);
}

double wolf_math_sin(double v)   { return sin(v); }
double wolf_math_cos(double v)   { return cos(v); }
double wolf_math_tan(double v)   { return tan(v); }
double wolf_math_asin(double v)  { return asin(v); }
double wolf_math_acos(double v)  { return acos(v); }
double wolf_math_atan(double v)  { return atan(v); }
double wolf_math_atan2(double y, double x) { return atan2(y, x); }

double wolf_math_sqrt(double v)               { return sqrt(v); }
double wolf_math_pow(double base, double exp_val) { return pow(base, exp_val); }
double wolf_math_log(double v)                { return log(v); }
double wolf_math_log10(double v)              { return log10(v); }
double wolf_math_exp(double v)                { return exp(v); }

double wolf_math_round(double v, int64_t precision) {
    double p = pow(10.0, (double)precision);
    return round(v * p) / p;
}
double wolf_math_fmod(double a, double b)     { return fmod(a, b); }
double wolf_math_pi(void)                     { return 3.14159265358979323846; }

const char* wolf_number_format(double number, int64_t decimals,
                               const char* dec_point, const char* thousands_sep) {
    if (!dec_point) dec_point = ".";
    if (!thousands_sep) thousands_sep = ",";

    char fmt[32];
    snprintf(fmt, sizeof(fmt), "%%.%lldf", (long long)decimals);
    char raw[128];
    snprintf(raw, sizeof(raw), fmt, number < 0 ? -number : number);

    char* dot = strchr(raw, '.');
    int int_len = dot ? (int)(dot - raw) : (int)strlen(raw);

    int commas = (int_len - 1) / 3;
    int result_len = (number < 0 ? 1 : 0) + int_len + commas
                   + (dot ? (int)strlen(dec_point) + (int)strlen(dot + 1) : 0) + 1;
    char* result = (char*)wolf_req_alloc(result_len + 16);
    char* p = result;

    if (number < 0) *p++ = '-';

    int sep_len = (int)strlen(thousands_sep);
    for (int i = 0; i < int_len; i++) {
        *p++ = raw[i];
        int remaining = int_len - i - 1;
        if (remaining > 0 && remaining % 3 == 0) {
            for (int s = 0; s < sep_len; s++) *p++ = thousands_sep[s];
        }
    }
    if (dot && decimals > 0) {
        int dp_len = (int)strlen(dec_point);
        for (int s = 0; s < dp_len; s++) *p++ = dec_point[s];
        char* frac = dot + 1;
        for (int i = 0; i < (int)decimals && frac[i]; i++) *p++ = frac[i];
    }
    *p = '\0';
    return result;
}

/* ========== Time & System ========== */

int64_t wolf_time_now(void) { return (int64_t)time(NULL); }

const char* wolf_env_get(const char* key, const char* def_val) {
    const char* val = getenv(key);
    return val ? val : (def_val ? def_val : "");
}

int wolf_env_has(const char* key) {
    return getenv(key) != NULL ? 1 : 0;
}

const char* wolf_time_date(const char* format, int64_t timestamp) {
    time_t rawtime = (time_t)timestamp;
    struct tm *info = localtime(&rawtime);
    char* buf = (char*)wolf_req_alloc(256);
    strftime(buf, 256, format, info);
    return buf;
}

void wolf_system_sleep(int64_t seconds) { sleep((unsigned int)seconds); }
void wolf_system_exit(int64_t code)     { exit((int)code); }
void wolf_system_die(const char* message) {
    if (message) printf("%s\n", message);
    exit(1);
}

/* ========== Sessions ========== */

void wolf_session_begin(void) {}
void wolf_session_set(const char* key, const char* value) { (void)key; (void)value; }
const char* wolf_session_get(const char* key) { (void)key; return ""; }
void wolf_session_end(void) {}

/* ========== Define System (PHP-style constants) ==========
 *
 * FIX: Use plain strdup() here — NOT wolf_req_strdup().
 * wolf_defines must persist for the entire process lifetime.
 * wolf_req_strdup() registers pointers with the per-request arena
 * which is flushed at the end of every HTTP request, which would
 * turn all define keys/values into dangling pointers after request 1.
 */

#ifndef WOLF_DEFINE_MAX
#define WOLF_DEFINE_MAX 256
#endif

#ifndef WOLF_FREESTANDING
static pthread_mutex_t wolf_defines_mutex = PTHREAD_MUTEX_INITIALIZER;
#endif /* WOLF_FREESTANDING */
static struct {
    char* keys[WOLF_DEFINE_MAX];
    char* values[WOLF_DEFINE_MAX];
    int   count;
} wolf_defines = { .count = 0 };

/* Convenience macros — no-ops when compiling without pthreads */
#ifdef WOLF_FREESTANDING
#  define WOLF_DEFINES_LOCK()
#  define WOLF_DEFINES_UNLOCK()
#else
#  define WOLF_DEFINES_LOCK()   pthread_mutex_lock(&wolf_defines_mutex)
#  define WOLF_DEFINES_UNLOCK() pthread_mutex_unlock(&wolf_defines_mutex)
#endif

void wolf_define(const char* key, const char* value) {
    if (!key) return;
    WOLF_DEFINES_LOCK();
    if (wolf_defines.count >= WOLF_DEFINE_MAX) {
        WOLF_DEFINES_UNLOCK();
        return;
    }
    for (int i = 0; i < wolf_defines.count; i++) {
        if (strcmp(wolf_defines.keys[i], key) == 0) {
            WOLF_DEFINES_UNLOCK();
            return; /* immutable */
        }
    }
    wolf_defines.keys[wolf_defines.count]   = strdup(key);
    wolf_defines.values[wolf_defines.count] = value ? strdup(value) : strdup("");
    wolf_defines.count++;
    WOLF_DEFINES_UNLOCK();
}

int wolf_defined(const char* key) {
    if (!key) return 0;
    WOLF_DEFINES_LOCK();
    for (int i = 0; i < wolf_defines.count; i++) {
        if (strcmp(wolf_defines.keys[i], key) == 0) {
            WOLF_DEFINES_UNLOCK();
            return 1;
        }
    }
    WOLF_DEFINES_UNLOCK();
    return 0;
}

const char* wolf_define_get(const char* key) {
    if (!key) return "";
    WOLF_DEFINES_LOCK();
    for (int i = 0; i < wolf_defines.count; i++) {
        if (strcmp(wolf_defines.keys[i], key) == 0) {
            const char* val = wolf_defines.values[i];
            WOLF_DEFINES_UNLOCK();
            return val;
        }
    }
    WOLF_DEFINES_UNLOCK();
    return "";
}

/* ========== Database — Connection Pool ========== */

typedef struct {
    WolfDBConn *conn;
    int         in_use;
} WolfPoolSlot;

static WolfPoolSlot    wolf_pool[WOLF_DB_POOL_SIZE];
static int             wolf_pool_inited = 0;
#ifndef WOLF_FREESTANDING
static pthread_mutex_t wolf_pool_mutex  = PTHREAD_MUTEX_INITIALIZER;
static pthread_cond_t  wolf_pool_cond   = PTHREAD_COND_INITIALIZER;
#endif /* WOLF_FREESTANDING */

/* FIX: pool credentials are application-lifetime — use plain strdup(),
 * NOT wolf_req_strdup() which is per-request and gets freed after
 * each HTTP request flush.                                             */
static char *wolf_pool_host   = NULL;
static char *wolf_pool_user   = NULL;
static char *wolf_pool_pass   = NULL;
static char *wolf_pool_dbname = NULL;

typedef struct {
    WolfDBConn *conn;
    char       *sql;
    WolfDBRes  *last_result;
} WolfDBStmt;

static WolfDBConn* wolf_pool_open_one(void) {
    const char *host   = wolf_pool_host   ? wolf_pool_host   : WOLF_DB_HOST;
    const char *user   = wolf_pool_user   ? wolf_pool_user   : WOLF_DB_USER;
    const char *pass   = wolf_pool_pass   ? wolf_pool_pass   : WOLF_DB_PASS;
    const char *dbname = wolf_pool_dbname ? wolf_pool_dbname : WOLF_DB_NAME;

#if defined(WOLF_DB_POSTGRES)
    char conninfo[512];
    snprintf(conninfo, sizeof(conninfo),
             "host=%s port=%d dbname=%s user=%s password=%s",
             host && *host ? host : "localhost", WOLF_DB_PORT,
             dbname ? dbname : "", user ? user : "", pass ? pass : "");
    WolfDBConn *conn = PQconnectdb(conninfo);
    if (PQstatus(conn) != CONNECTION_OK) {
        fprintf(stderr, "[WOLF-POOL] PG connect failed: %s\n", PQerrorMessage(conn));
        PQfinish(conn);
        return NULL;
    }
    return conn;

#elif defined(WOLF_DB_MSSQL)
    return (WolfDBConn*)1;

#else
    MYSQL *conn = mysql_init(NULL);
    if (!conn) return NULL;

    if (host && (strcmp(host, "localhost") == 0 || strcmp(host, "") == 0)) {
        const char *sockets[] = {
            "/opt/lampp/var/mysql/mysql.sock",
            "/var/run/mysqld/mysqld.sock",
            "/tmp/mysql.sock",
            "/var/lib/mysql/mysql.sock",
            NULL
        };
        for (int i = 0; sockets[i]; i++) {
            if (access(sockets[i], F_OK) == 0) {
                if (mysql_real_connect(conn, NULL, user, pass, dbname,
                                       0, sockets[i], 0)) {
                    mysql_set_character_set(conn, "utf8mb4");
                    return conn;
                }
                mysql_close(conn);
                conn = mysql_init(NULL);
                if (!conn) return NULL;
            }
        }
        if (mysql_real_connect(conn, "127.0.0.1", user, pass, dbname,
                               WOLF_DB_PORT, NULL, 0)) {
            mysql_set_character_set(conn, "utf8mb4");
            return conn;
        }
    } else {
        if (mysql_real_connect(conn, host, user, pass, dbname,
                               WOLF_DB_PORT, NULL, 0)) {
            mysql_set_character_set(conn, "utf8mb4");
            return conn;
        }
    }
    fprintf(stderr, "[WOLF-POOL] MySQL connect failed: %s\n", mysql_error(conn));
    mysql_close(conn);
    return NULL;
#endif
}

static void wolf_pool_init_locked(void) {
    if (wolf_pool_inited) return;
    fprintf(stderr, "[WOLF-POOL] Initializing pool (size=%d) host=%s db=%s\n",
            WOLF_DB_POOL_SIZE,
            wolf_pool_host ? wolf_pool_host : WOLF_DB_HOST,
            wolf_pool_dbname ? wolf_pool_dbname : WOLF_DB_NAME);
    for (int i = 0; i < WOLF_DB_POOL_SIZE; i++) {
        wolf_pool[i].in_use = 0;
        wolf_pool[i].conn   = wolf_pool_open_one();
        if (wolf_pool[i].conn)
            fprintf(stderr, "[WOLF-POOL] slot %d OK\n", i);
        else
            fprintf(stderr, "[WOLF-POOL] slot %d FAILED (will retry on acquire)\n", i);
    }
    wolf_pool_inited = 1;
}

/* DB pool lock macros — no-ops when compiling without pthreads */
#ifdef WOLF_FREESTANDING
#  define WOLF_POOL_LOCK()
#  define WOLF_POOL_UNLOCK()
#  define WOLF_POOL_SIGNAL()
#  define WOLF_POOL_TIMEDWAIT(d)  ETIMEDOUT   /* always "timeout" — no blocking in freestanding */
#else
#  define WOLF_POOL_LOCK()        pthread_mutex_lock(&wolf_pool_mutex)
#  define WOLF_POOL_UNLOCK()      pthread_mutex_unlock(&wolf_pool_mutex)
#  define WOLF_POOL_SIGNAL()      pthread_cond_signal(&wolf_pool_cond)
#  define WOLF_POOL_TIMEDWAIT(d)  pthread_cond_timedwait(&wolf_pool_cond, &wolf_pool_mutex, (d))
#endif

static WolfDBConn* wolf_pool_acquire(void) {
    struct timespec deadline;
    clock_gettime(CLOCK_REALTIME, &deadline);
    deadline.tv_sec += WOLF_DB_POOL_TIMEOUT;

    WOLF_POOL_LOCK();
    wolf_pool_init_locked();

    while (1) {
        for (int i = 0; i < WOLF_DB_POOL_SIZE; i++) {
            if (!wolf_pool[i].in_use) {
                if (wolf_pool[i].conn && WOLF_DB_PING(wolf_pool[i].conn) != 0) {
                    WOLF_LOG("[WOLF-POOL] slot %d stale, reconnecting\n", i);
                    WOLF_DB_CLOSE(wolf_pool[i].conn);
                    wolf_pool[i].conn = wolf_pool_open_one();
                }
                if (!wolf_pool[i].conn)
                    wolf_pool[i].conn = wolf_pool_open_one();
                if (wolf_pool[i].conn) {
                    wolf_pool[i].in_use = 1;
                    WOLF_LOG("[WOLF-POOL] acquire slot=%d\n", i);
                    WOLF_POOL_UNLOCK();
                    return wolf_pool[i].conn;
                }
            }
        }
        int rc = WOLF_POOL_TIMEDWAIT(&deadline);
        if (rc == ETIMEDOUT) {
            fprintf(stderr, "[WOLF-POOL] timeout waiting for free slot (%ds)\n",
                    WOLF_DB_POOL_TIMEOUT);
            WOLF_POOL_UNLOCK();
            return NULL;
        }
    }
}

static void wolf_pool_release(WolfDBConn *conn) {
    if (!conn) return;
    WOLF_POOL_LOCK();
    for (int i = 0; i < WOLF_DB_POOL_SIZE; i++) {
        if (wolf_pool[i].conn == conn) {
            wolf_pool[i].in_use = 0;
            WOLF_LOG("[WOLF-POOL] release slot=%d\n", i);
            WOLF_POOL_SIGNAL();
            break;
        }
    }
    WOLF_POOL_UNLOCK();
}

/* FIX: pool credentials stored with plain strdup() so they survive
 * across all HTTP requests for the entire process lifetime.          */
void* wolf_db_connect(const char* host, const char* user,
                      const char* pass, const char* dbname) {
    WOLF_POOL_LOCK();
    if (!wolf_pool_host && host && *host) {
        wolf_pool_host   = strdup(host);
        wolf_pool_user   = (user   && *user)   ? strdup(user)   : NULL;
        wolf_pool_pass   = (pass   && *pass)   ? strdup(pass)   : NULL;
        wolf_pool_dbname = (dbname && *dbname) ? strdup(dbname) : NULL;
    }
    WOLF_POOL_UNLOCK();

    WolfDBConn *conn = wolf_pool_acquire();
    if (!conn) fprintf(stderr, "[WOLF-POOL] failed to acquire connection\n");
    return (void*)conn;
}

static char* wolf_internal_str_replace(const char* orig, const char* rep,
                                        const char* with, int limit) {
    char *result, *ins, *tmp;
    int len_rep, len_with, len_front, count;
    if (!orig || !rep) return NULL;
    len_rep = strlen(rep);
    if (len_rep == 0) return NULL;
    if (!with) with = "";
    len_with = strlen(with);
    ins = (char *)orig;
    for (count = 0; (tmp = strstr(ins, rep)); ++count) {
        ins = tmp + len_rep;
        if (limit > 0 && count + 1 >= limit) { count++; break; }
    }
    tmp = result = wolf_req_alloc(strlen(orig) + (len_with - len_rep) * count + 1);
    if (!result) return NULL;
    int replaced = 0;
    while (count--) {
        ins = strstr(orig, rep);
        len_front = ins - orig;
        tmp = strncpy(tmp, orig, len_front) + len_front;
        tmp = strcpy(tmp, with) + len_with;
        orig += len_front + len_rep;
        replaced++;
        if (limit > 0 && replaced >= limit) break;
    }
    strcpy(tmp, orig);
    return result;
}

void* wolf_db_prepare(void* conn, const char* sql) {
    if (!conn) return NULL;
    WolfDBStmt* stmt = (WolfDBStmt*)wolf_req_alloc(sizeof(WolfDBStmt));
    stmt->conn = (WolfDBConn*)conn;
    stmt->sql  = wolf_req_strdup(sql ? sql : "");
    stmt->last_result = NULL;
    return stmt;
}

void wolf_db_bind(void* stmt_ptr, const char* param, const char* value) {
    if (!stmt_ptr || !param) return;
    WolfDBStmt* stmt = (WolfDBStmt*)stmt_ptr;
    if (!value) value = "";

#if defined(WOLF_DB_POSTGRES)
    char *escaped = wolf_req_alloc(strlen(value) * 2 + 1);
    int err;
    PQescapeStringConn(stmt->conn, escaped, value, strlen(value), &err);
#elif defined(WOLF_DB_MSSQL)
    char *escaped = wolf_req_alloc(strlen(value) * 2 + 1);
    strcpy(escaped, value);
#else
    size_t val_len = strlen(value);
    char *escaped = wolf_req_alloc(val_len * 2 + 1);
    mysql_real_escape_string(stmt->conn, escaped, value, val_len);
#endif

    char *quoted = wolf_req_alloc(strlen(escaped) + 3);
    sprintf(quoted, "'%s'", escaped);
    char *new_sql = wolf_internal_str_replace(stmt->sql, param, quoted, 1);
    if (new_sql) stmt->sql = new_sql;
}

int64_t wolf_db_execute(void* stmt_ptr) {
    if (!stmt_ptr) return 0;
    WolfDBStmt* stmt = (WolfDBStmt*)stmt_ptr;

#if defined(WOLF_DB_POSTGRES)
    if (stmt->last_result) { PQclear(stmt->last_result); stmt->last_result = NULL; }
    WolfDBRes *res = PQexec(stmt->conn, stmt->sql);
    if (PQresultStatus(res) != PGRES_COMMAND_OK &&
        PQresultStatus(res) != PGRES_TUPLES_OK) {
        printf("[WOLF-DB] PG Query failed: %s | SQL: %s\n", PQerrorMessage(stmt->conn), stmt->sql);
        PQclear(res); return 0;
    }
    stmt->last_result = res; return 1;
#elif defined(WOLF_DB_MSSQL)
    (void)stmt;
    return 1;
#else
    if (stmt->last_result) { mysql_free_result(stmt->last_result); stmt->last_result = NULL; }
    if (mysql_query(stmt->conn, stmt->sql)) {
        printf("[WOLF-DB] MySQL Query failed: %s | SQL: %s\n", mysql_error(stmt->conn), stmt->sql);
        return 0;
    }
    stmt->last_result = mysql_store_result(stmt->conn);
    return 1;
#endif
}

static wolf_value_t* wolf_val_make(int type); /* forward declaration */

void* wolf_db_fetch_all(void* stmt_ptr) {
    void* arr = wolf_array_create();
    if (!stmt_ptr) return arr;
    WolfDBStmt* stmt = (WolfDBStmt*)stmt_ptr;
    if (!stmt->last_result) return arr;

#if defined(WOLF_DB_POSTGRES)
    WolfDBRes *res = stmt->last_result;
    int num_fields = PQnfields(res);
    int num_rows   = PQntuples(res);
    for (int r = 0; r < num_rows; r++) {
        void *row = wolf_map_create();
        for (int i = 0; i < num_fields; i++) {
            if (PQgetisnull(res, r, i))
                wolf_map_set(row, PQfname(res, i), NULL);
            else {
                wolf_value_t *sv = wolf_val_make(WOLF_TYPE_STRING);
                sv->val.s = wolf_req_strdup(PQgetvalue(res, r, i));
                wolf_map_set(row, PQfname(res, i), sv);
            }
        }
        wolf_array_push(arr, row);
    }
    return arr;
#elif defined(WOLF_DB_MSSQL)
    return arr;
#else
    int num_fields = mysql_num_fields(stmt->last_result);
    MYSQL_FIELD *fields = mysql_fetch_fields(stmt->last_result);
    MYSQL_ROW row_data;
    while ((row_data = mysql_fetch_row(stmt->last_result))) {
        unsigned long *lengths = mysql_fetch_lengths(stmt->last_result);
        void *row = wolf_map_create();
        for (int i = 0; i < num_fields; i++) {
            if (row_data[i]) {
                wolf_value_t *sv = wolf_val_make(WOLF_TYPE_STRING);
                char *val = wolf_req_alloc(lengths[i] + 1);
                memcpy(val, row_data[i], lengths[i]);
                val[lengths[i]] = '\0';
                sv->val.s = val;
                wolf_map_set(row, fields[i].name, sv);
            } else {
                wolf_map_set(row, fields[i].name, NULL);
            }
        }
        wolf_array_push(arr, row);
    }
    return arr;
#endif
}

void* wolf_db_fetch_one(void* stmt_ptr) {
    if (!stmt_ptr) return wolf_map_create();
    WolfDBStmt* stmt = (WolfDBStmt*)stmt_ptr;
    if (!stmt->last_result) return wolf_map_create();

#if defined(WOLF_DB_POSTGRES)
    WolfDBRes *res = stmt->last_result;
    if (PQntuples(res) > 0) {
        int num_fields = PQnfields(res);
        void *row = wolf_map_create();
        for (int i = 0; i < num_fields; i++) {
            if (PQgetisnull(res, 0, i))
                wolf_map_set(row, PQfname(res, i), NULL);
            else {
                wolf_value_t *sv = wolf_val_make(WOLF_TYPE_STRING);
                sv->val.s = wolf_req_strdup(PQgetvalue(res, 0, i));
                wolf_map_set(row, PQfname(res, i), sv);
            }
        }
        return row;
    }
    return wolf_map_create();
#elif defined(WOLF_DB_MSSQL)
    return wolf_map_create();
#else
    int num_fields = mysql_num_fields(stmt->last_result);
    MYSQL_FIELD *fields = mysql_fetch_fields(stmt->last_result);
    MYSQL_ROW row_data = mysql_fetch_row(stmt->last_result);
    if (row_data) {
        unsigned long *lengths = mysql_fetch_lengths(stmt->last_result);
        void *row = wolf_map_create();
        for (int i = 0; i < num_fields; i++) {
            if (row_data[i]) {
                wolf_value_t *sv = wolf_val_make(WOLF_TYPE_STRING);
                char *val = wolf_req_alloc(lengths[i] + 1);
                memcpy(val, row_data[i], lengths[i]);
                val[lengths[i]] = '\0';
                sv->val.s = val;
                wolf_map_set(row, fields[i].name, sv);
            } else {
                wolf_map_set(row, fields[i].name, NULL);
            }
        }
        return row;
    }
    return wolf_map_create();
#endif
}

int64_t wolf_db_row_count(void* stmt_ptr) {
    if (!stmt_ptr) return 0;
    WolfDBStmt* stmt = (WolfDBStmt*)stmt_ptr;
#if defined(WOLF_DB_POSTGRES)
    return stmt->last_result ? (int64_t)PQntuples(stmt->last_result) : 0;
#elif defined(WOLF_DB_MSSQL)
    (void)stmt;
    return 0;
#else
    if (stmt->last_result) return (int64_t)mysql_num_rows(stmt->last_result);
    return (int64_t)mysql_affected_rows(stmt->conn);
#endif
}

int64_t wolf_db_last_insert_id(void* conn_ptr) {
    if (!conn_ptr) return 0;
#if defined(WOLF_DB_POSTGRES)
    return 0;
#elif defined(WOLF_DB_MSSQL)
    return 0;
#else
    return (int64_t)mysql_insert_id((WolfDBConn*)conn_ptr);
#endif
}

void wolf_db_close(void* conn_ptr) {
    if (!conn_ptr) return;
    wolf_pool_release((WolfDBConn*)conn_ptr);
}

/*
 * wolf_db_pool_destroy — close every connection in the pool and
 * wake any threads blocked in wolf_pool_acquire() so they can
 * observe the shutdown flag and return gracefully.
 *
 * Safe to call from main thread only (not from a signal handler).
 * After this call the pool is unusable — intended for process exit.
 */
void wolf_db_pool_destroy(void) {
    WOLF_POOL_LOCK();

    fprintf(stderr, "[WOLF-POOL] Destroying pool (%d slots)...\n",
            WOLF_DB_POOL_SIZE);

    for (int i = 0; i < WOLF_DB_POOL_SIZE; i++) {
        if (wolf_pool[i].conn) {
            WOLF_DB_CLOSE(wolf_pool[i].conn);
            wolf_pool[i].conn   = NULL;
            wolf_pool[i].in_use = 0;
            WOLF_LOG("[WOLF-POOL] slot %d closed\n", i);
        }
    }

    wolf_pool_inited = 0;

#ifndef WOLF_FREESTANDING
    /* Wake every thread waiting in wolf_pool_acquire() */
    pthread_cond_broadcast(&wolf_pool_cond);
#endif

    WOLF_POOL_UNLOCK();

    /* Free persistent credential strings */
    if (wolf_pool_host)   { free(wolf_pool_host);   wolf_pool_host   = NULL; }
    if (wolf_pool_user)   { free(wolf_pool_user);   wolf_pool_user   = NULL; }
    if (wolf_pool_pass)   { free(wolf_pool_pass);   wolf_pool_pass   = NULL; }
    if (wolf_pool_dbname) { free(wolf_pool_dbname); wolf_pool_dbname = NULL; }

    fprintf(stderr, "[WOLF-POOL] Pool destroyed.\n");
}

void wolf_db_begin_transaction(void* conn_ptr) {
    if (!conn_ptr) return;
#if defined(WOLF_DB_POSTGRES)
    PQexec((WolfDBConn*)conn_ptr, "BEGIN");
#elif defined(WOLF_DB_MSSQL)
#else
    mysql_query((WolfDBConn*)conn_ptr, "START TRANSACTION");
#endif
}

void wolf_db_commit(void* conn_ptr) {
    if (!conn_ptr) return;
#if defined(WOLF_DB_POSTGRES)
    PQexec((WolfDBConn*)conn_ptr, "COMMIT");
#elif defined(WOLF_DB_MSSQL)
#else
    mysql_query((WolfDBConn*)conn_ptr, "COMMIT");
#endif
}

void wolf_db_rollback(void* conn_ptr) {
    if (!conn_ptr) return;
#if defined(WOLF_DB_POSTGRES)
    PQexec((WolfDBConn*)conn_ptr, "ROLLBACK");
#elif defined(WOLF_DB_MSSQL)
#else
    mysql_query((WolfDBConn*)conn_ptr, "ROLLBACK");
#endif
}

/* ================================================================ *
 * Redis                                                            *
 *                                                                  *
 * With    -DWOLF_REDIS_ENABLED: real hiredis implementation.      *
 * Without -DWOLF_REDIS_ENABLED: safe no-op stubs so the runtime   *
 * compiles and links on machines without hiredis installed.        *
 * ================================================================ */

#ifdef WOLF_REDIS_ENABLED
#include <hiredis/hiredis.h>

static __thread redisContext* wolf_redis_ctx = NULL;

void* wolf_redis_connect(const char* host, int64_t port, const char* pass) {
    if (wolf_redis_ctx) return wolf_redis_ctx;
    struct timeval tv = {1, 500000};
    wolf_redis_ctx = redisConnectWithTimeout(host, (int)port, tv);
    if (wolf_redis_ctx == NULL || wolf_redis_ctx->err) {
        if (wolf_redis_ctx) {
            fprintf(stderr, "[WOLF-REDIS] Connection error: %s\n", wolf_redis_ctx->errstr);
            redisFree(wolf_redis_ctx);
            wolf_redis_ctx = NULL;
        } else {
            fprintf(stderr, "[WOLF-REDIS] Connection error: can't allocate redis context\n");
        }
        return NULL;
    }
    if (pass && strlen(pass) > 0) {
        redisReply *reply = redisCommand(wolf_redis_ctx, "AUTH %s", pass);
        if (reply) freeReplyObject(reply);
    }
    return wolf_redis_ctx;
}

void wolf_redis_set(const char* key, const char* value, int64_t ttl) {
    redisContext* c = wolf_redis_ctx;
    fprintf(stderr, "[DEBUG REDIS SET] ctx=%p key=%s val=%s ttl=%ld\n", c, key ? key : "NULL", value ? value : "NULL", ttl);
    if (!c || !key) return;
    redisReply *reply;
    if (ttl > 0)
        reply = redisCommand(c, "SET %s %s EX %lld", key, value ? value : "", (long long)ttl);
    else
        reply = redisCommand(c, "SET %s %s", key, value ? value : "");
    if (reply) freeReplyObject(reply);
}

const char* wolf_redis_get(const char* key) {
    redisContext* c = wolf_redis_ctx;
    if (!c || !key) return "";
    redisReply *reply = redisCommand(c, "GET %s", key);
    if (reply && reply->type == REDIS_REPLY_STRING) {
        char *str = wolf_req_strdup(reply->str);
        freeReplyObject(reply);
        return str;
    }
    if (reply) freeReplyObject(reply);
    return "";
}

int64_t wolf_redis_del(const char* key) {
    redisContext* c = wolf_redis_ctx;
    if (!c || !key) return 0;
    redisReply *reply = redisCommand(c, "DEL %s", key);
    int64_t result = 0;
    if (reply && reply->type == REDIS_REPLY_INTEGER) result = reply->integer;
    if (reply) freeReplyObject(reply);
    return result;
}

int wolf_redis_exists(const char* key) {
    redisContext* c = wolf_redis_ctx;
    if (!c || !key) return 0;
    redisReply *reply = redisCommand(c, "EXISTS %s", key);
    int result = 0;
    if (reply && reply->type == REDIS_REPLY_INTEGER) result = reply->integer > 0 ? 1 : 0;
    if (reply) freeReplyObject(reply);
    return result;
}

void wolf_redis_hset(const char* key, const char* field, const char* value) {
    redisContext* c = wolf_redis_ctx;
    if (!c || !key || !field) return;
    redisReply *reply = redisCommand(c, "HSET %s %s %s", key, field, value ? value : "");
    if (reply) freeReplyObject(reply);
}

const char* wolf_redis_hget(const char* key, const char* field) {
    redisContext* c = wolf_redis_ctx;
    if (!c || !key || !field) return "";
    redisReply *reply = redisCommand(c, "HGET %s %s", key, field);
    if (reply && reply->type == REDIS_REPLY_STRING) {
        char *str = wolf_req_strdup(reply->str);
        freeReplyObject(reply);
        return str;
    }
    if (reply) freeReplyObject(reply);
    return "";
}

void wolf_redis_close() {
    redisContext* c = wolf_redis_ctx;
    if (c) {
        redisFree(c);
        if (c == wolf_redis_ctx) wolf_redis_ctx = NULL;
    }
}

#else  /* !WOLF_REDIS_ENABLED — safe no-op stubs */

void*       wolf_redis_connect(const char* h, int64_t p, const char* pw)
                { (void)h; (void)p; (void)pw; return NULL; }
void        wolf_redis_set(const char* k, const char* v, int64_t t)
                { (void)k; (void)v; (void)t; }
const char* wolf_redis_get(const char* k)
                { (void)k; return ""; }
int64_t     wolf_redis_del(const char* k)
                { (void)k; return 0; }
int         wolf_redis_exists(const char* k)
                { (void)k; return 0; }
void        wolf_redis_hset(const char* k, const char* f, const char* v)
                { (void)k; (void)f; (void)v; }
const char* wolf_redis_hget(const char* k, const char* f)
                { (void)k; (void)f; return ""; }
void        wolf_redis_close()
                { }

#endif /* WOLF_REDIS_ENABLED */

/* ========== Stdlib Strings & JSON ========== */

int wolf_strings_contains(const char* s, const char* substr) {
    if (!s || !substr) return 0;
    return strstr(s, substr) != NULL;
}
const char* wolf_strings_upper(const char* s)   { return wolf_strtoupper(s); }
const char* wolf_strings_title(const char* s)   { return wolf_ucwords(s); }
const char* wolf_strings_trimleft(const char* s, const char* cutset)  { (void)cutset; return wolf_ltrim(s); }
const char* wolf_strings_trimright(const char* s, const char* cutset) { (void)cutset; return wolf_rtrim(s); }

const char* wolf_strings_split(const char* s, const char* sep) { (void)sep; return s; }

const char* wolf_strings_join(const char* arr, const char* sep) {
    if (!arr) return "";
    size_t len = strlen(arr);
    char* result = (char*)wolf_req_alloc(len + 1);
    for (size_t i = 0; i < len; i++) {
        result[i] = (arr[i] == ',') ? (sep ? sep[0] : '-') : arr[i];
    }
    result[len] = '\0';
    return result;
}

/* ========== Data Structures (Arrays & Maps) ========== */

typedef struct {
    void**  items;
    int64_t length;
    int64_t capacity;
} wolf_array_t;

typedef struct {
    char**  keys;
    void**  values;
    int64_t size;
    int64_t capacity;
} wolf_map_t;

int wolf_is_tagged_value(void* ptr) {
    if (!ptr) return 0;
    wolf_value_t* v = (wolf_value_t*)ptr;
    return (v->magic == WOLF_VALUE_MAGIC);
}

static wolf_value_t* wolf_val_make(int type) {
    wolf_value_t* v = (wolf_value_t*)wolf_req_alloc(sizeof(wolf_value_t));
    v->magic = WOLF_VALUE_MAGIC;
    v->type = type;
    return v;
}

static char* wolf_json_encode_value(void* val); /* forward */

static const char* wolf_value_unwrap_string(void* val) {
    if (!val) return "";
    if (!wolf_is_tagged_value(val)) return (const char*)val;
    wolf_value_t* tagged = (wolf_value_t*)val;
    switch (tagged->type) {
        case WOLF_TYPE_INT: {
            char* buf = (char*)wolf_req_alloc(32);
            snprintf(buf, 32, "%lld", (long long)tagged->val.i);
            return buf;
        }
        case WOLF_TYPE_FLOAT: {
            char* buf = (char*)wolf_req_alloc(64);
            snprintf(buf, 64, "%g", tagged->val.f);
            return buf;
        }
        case WOLF_TYPE_BOOL:  return tagged->val.b ? "true" : "false";
        case WOLF_TYPE_NULL:  return "";
        case WOLF_TYPE_STRING: return tagged->val.s ? tagged->val.s : "";
        case WOLF_TYPE_MAP:
        case WOLF_TYPE_ARRAY: return wolf_json_encode_value(val);
        default: return (const char*)val;
    }
}

static double wolf_value_unwrap_double(void* val) {
    if (!val) return 0.0;
    if (!wolf_is_tagged_value(val)) return atof((const char*)val);
    wolf_value_t* tagged = (wolf_value_t*)val;
    switch (tagged->type) {
        case WOLF_TYPE_INT:    return (double)tagged->val.i;
        case WOLF_TYPE_FLOAT:  return tagged->val.f;
        case WOLF_TYPE_BOOL:   return tagged->val.b ? 1.0 : 0.0;
        case WOLF_TYPE_STRING: return tagged->val.s ? atof(tagged->val.s) : 0.0;
        default:               return 0.0;
    }
}

wolf_value_t* wolf_val_int(int64_t i) {
    wolf_value_t* v = wolf_val_make(WOLF_TYPE_INT);
    v->val.i = i; return v;
}
wolf_value_t* wolf_val_float(double f) {
    wolf_value_t* v = wolf_val_make(WOLF_TYPE_FLOAT);
    v->val.f = f; return v;
}
wolf_value_t* wolf_val_bool(int b) {
    wolf_value_t* v = wolf_val_make(WOLF_TYPE_BOOL);
    v->val.b = b; return v;
}
wolf_value_t* wolf_val_array(void* arr) {
    wolf_value_t* v = wolf_val_make(WOLF_TYPE_ARRAY);
    v->val.ptr = arr; return v;
}

/* Dynamic string buffer for safe JSON building */
typedef struct { char* data; size_t len; size_t cap; } wolf_strbuf_t;

static wolf_strbuf_t* wolf_strbuf_new(void) {
    wolf_strbuf_t* b = (wolf_strbuf_t*)wolf_req_alloc(sizeof(wolf_strbuf_t));
    if (!b) return NULL;
    b->cap = 256; b->len = 0;
    b->data = (char*)wolf_req_alloc(b->cap);
    if (!b->data) { return NULL; }
    b->data[0] = '\0';
    return b;
}

static int wolf_strbuf_append(wolf_strbuf_t* b, const char* s) {
    if (!b || !s) return 0;
    size_t slen = strlen(s);
    while (b->len + slen + 1 > b->cap) {
        size_t old_cap = b->cap;
        b->cap *= 2;
        char* nd = (char*)wolf_req_realloc(b->data, old_cap, b->cap);
        if (!nd) return 0;
        b->data = nd;
    }
    memcpy(b->data + b->len, s, slen);
    b->len += slen;
    b->data[b->len] = '\0';
    return 1;
}

static char* wolf_strbuf_take(wolf_strbuf_t* b) {
    if (!b) return wolf_req_strdup("");
    return b->data;
}

static char* wolf_json_encode_map(wolf_map_t* m) {
    if (!m) return wolf_req_strdup("{}");
    int64_t n = m->size;
    int64_t* order = (int64_t*)wolf_req_alloc(n * sizeof(int64_t));
    if (!order) return wolf_req_strdup("{}");
    for (int64_t i = 0; i < n; i++) order[i] = i;
    for (int64_t i = 0; i < n - 1; i++)
        for (int64_t j = i + 1; j < n; j++)
            if (strcmp(m->keys[order[i]], m->keys[order[j]]) > 0) {
                int64_t tmp = order[i]; order[i] = order[j]; order[j] = tmp;
            }
    wolf_strbuf_t* buf = wolf_strbuf_new();
    if (!buf) return wolf_req_strdup("{}");
    wolf_strbuf_append(buf, "{");
    for (int64_t i = 0; i < n; i++) {
        if (i > 0) wolf_strbuf_append(buf, ",");
        wolf_strbuf_append(buf, "\"");
        wolf_strbuf_append(buf, m->keys[order[i]]);
        wolf_strbuf_append(buf, "\":");
        char* val = wolf_json_encode_value(m->values[order[i]]);
        wolf_strbuf_append(buf, val);
    }
    wolf_strbuf_append(buf, "}");
    return wolf_strbuf_take(buf);
}

static char* wolf_json_encode_array(wolf_array_t* a) {
    if (!a) return wolf_req_strdup("[]");
    wolf_strbuf_t* buf = wolf_strbuf_new();
    if (!buf) return wolf_req_strdup("[]");
    wolf_strbuf_append(buf, "[");
    for (int64_t i = 0; i < a->length; i++) {
        if (i > 0) wolf_strbuf_append(buf, ",");
        wolf_strbuf_append(buf, wolf_json_encode_value(a->items[i]));
    }
    wolf_strbuf_append(buf, "]");
    return wolf_strbuf_take(buf);
}

static char* wolf_json_encode_value(void* val) {
    if (!val) return wolf_req_strdup("null");

    wolf_value_t* tagged = (wolf_value_t*)val;
    if (tagged->type == WOLF_TYPE_INT) {
        char* buf = (char*)wolf_req_alloc(32);
        snprintf(buf, 32, "%lld", (long long)tagged->val.i);
        return buf;
    }
    if (tagged->type == WOLF_TYPE_FLOAT) {
        char* buf = (char*)wolf_req_alloc(64);
        snprintf(buf, 64, "%g", tagged->val.f);
        return buf;
    }
    if (tagged->type == WOLF_TYPE_BOOL)
        return wolf_req_strdup(tagged->val.b ? "true" : "false");
    if (tagged->type == WOLF_TYPE_NULL)
        return wolf_req_strdup("null");
    if (tagged->type == WOLF_TYPE_STRING) {
        const char* s = tagged->val.s;
        size_t len = strlen(s);
        char* out = (char*)wolf_req_alloc(len * 2 + 3);
        char* w = out;
        *w++ = '"';
        for (size_t i = 0; i < len; i++) {
            if      (s[i] == '"')  { *w++ = '\\'; *w++ = '"'; }
            else if (s[i] == '\\') { *w++ = '\\'; *w++ = '\\'; }
            else if (s[i] == '\n') { *w++ = '\\'; *w++ = 'n'; }
            else if (s[i] == '\r') { *w++ = '\\'; *w++ = 'r'; }
            else if (s[i] == '\t') { *w++ = '\\'; *w++ = 't'; }
            else *w++ = s[i];
        }
        *w++ = '"'; *w = '\0';
        return out;
    }
    if (tagged->type == WOLF_TYPE_MAP)
        return wolf_json_encode_map((wolf_map_t*)tagged->val.ptr);
    if (tagged->type == WOLF_TYPE_ARRAY)
        return wolf_json_encode_array((wolf_array_t*)tagged->val.ptr);

    /* Fallback: raw pointer — try array, then map, then plain string */
    wolf_array_t* a = (wolf_array_t*)val;
    if (a && (uintptr_t)a % 8 == 0 &&
        a->capacity > 0 && a->capacity <= 1000000 &&
        a->length >= 0 && a->length <= a->capacity && 
        a->items != NULL && ((uintptr_t)a->items % 8 == 0))
        return wolf_json_encode_array(a);

    wolf_map_t* m = (wolf_map_t*)val;
    if (m && (uintptr_t)m % 8 == 0 &&
        m->capacity > 0 && m->capacity <= 1000000 &&
        m->size >= 0 && m->size <= m->capacity && 
        m->keys != NULL && ((uintptr_t)m->keys % 8 == 0) &&
        m->values != NULL && ((uintptr_t)m->values % 8 == 0))
        return wolf_json_encode_map(m);

    const char* s = (const char*)val;
    size_t len = strlen(s);
    char* out = (char*)wolf_req_alloc(len * 2 + 3);
    char* w = out;
    *w++ = '"';
    for (size_t i = 0; i < len; i++) {
        if      (s[i] == '"')  { *w++ = '\\'; *w++ = '"'; }
        else if (s[i] == '\\') { *w++ = '\\'; *w++ = '\\'; }
        else if (s[i] == '\n') { *w++ = '\\'; *w++ = 'n'; }
        else if (s[i] == '\r') { *w++ = '\\'; *w++ = 'r'; }
        else if (s[i] == '\t') { *w++ = '\\'; *w++ = 't'; }
        else *w++ = s[i];
    }
    *w++ = '"'; *w = '\0';
    return out;
}

const char* wolf_json_encode(void* obj) {
    if (!obj) return wolf_req_strdup("null");
    return wolf_json_encode_value(obj);
}

void* wolf_array_create(void) {
    wolf_array_t* arr = (wolf_array_t*)wolf_req_alloc(sizeof(wolf_array_t));
    arr->capacity = 8;
    arr->length   = 0;
    arr->items    = (void**)wolf_req_alloc(sizeof(void*) * arr->capacity);
    return arr;
}

void wolf_array_push(void* a, void* item) {
    if (!a) return;
    wolf_array_t* arr = (wolf_array_t*)a;
    if (arr->length >= arr->capacity) {
        arr->capacity *= 2;
        arr->items = (void**)realloc(arr->items, sizeof(void*) * arr->capacity);
    }
    arr->items[arr->length++] = item;
}

void* wolf_array_get(void* a, int64_t index) {
    if (!a) return NULL;
    wolf_array_t* arr = (wolf_array_t*)a;
    if (index < 0 || index >= arr->length) return NULL;
    void* val = arr->items[index];
    if (!val) return NULL;
    wolf_value_t* tagged = (wolf_value_t*)val;
    if (tagged->type < WOLF_TYPE_STRING || tagged->type > WOLF_TYPE_ARRAY) return val;
    return (void*)wolf_value_unwrap_string(val);
}

int64_t wolf_array_length(void* a) {
    if (!a) return 0;
    return ((wolf_array_t*)a)->length;
}

/* ========== Phase 2 Stdlib — Array Functions ========== */

int64_t wolf_count(void* a)       { return wolf_array_length(a); }

int wolf_in_array(const char* val, void* a) {
    if (!a || !val) return 0;
    wolf_array_t* arr = (wolf_array_t*)a;
    for (int64_t i = 0; i < arr->length; i++)
        if (arr->items[i] && strcmp((const char*)arr->items[i], val) == 0) return 1;
    return 0;
}

int64_t wolf_array_search(const char* val, void* a) {
    if (!a || !val) return -1;
    wolf_array_t* arr = (wolf_array_t*)a;
    for (int64_t i = 0; i < arr->length; i++)
        if (arr->items[i] && strcmp((const char*)arr->items[i], val) == 0) return i;
    return -1;
}

void* wolf_array_pop(void* a) {
    if (!a) return NULL;
    wolf_array_t* arr = (wolf_array_t*)a;
    if (arr->length == 0) return NULL;
    return arr->items[--arr->length];
}

void* wolf_array_shift(void* a) {
    if (!a) return NULL;
    wolf_array_t* arr = (wolf_array_t*)a;
    if (arr->length == 0) return NULL;
    void* first = arr->items[0];
    for (int64_t i = 0; i < arr->length - 1; i++) arr->items[i] = arr->items[i+1];
    arr->length--;
    return first;
}

void wolf_array_unshift(void* a, void* item) {
    if (!a) return;
    wolf_array_t* arr = (wolf_array_t*)a;
    if (arr->length >= arr->capacity) {
        arr->capacity *= 2;
        arr->items = (void**)realloc(arr->items, sizeof(void*) * arr->capacity);
    }
    for (int64_t i = arr->length; i > 0; i--) arr->items[i] = arr->items[i-1];
    arr->items[0] = item;
    arr->length++;
}

void* wolf_array_reverse(void* a) {
    if (!a) return wolf_array_create();
    wolf_array_t* arr = (wolf_array_t*)a;
    void* result = wolf_array_create();
    for (int64_t i = arr->length - 1; i >= 0; i--) wolf_array_push(result, arr->items[i]);
    return result;
}

void* wolf_array_unique(void* a) {
    if (!a) return wolf_array_create();
    wolf_array_t* arr = (wolf_array_t*)a;
    void* result = wolf_array_create();
    for (int64_t i = 0; i < arr->length; i++) {
        int found = 0;
        wolf_array_t* res = (wolf_array_t*)result;
        for (int64_t j = 0; j < res->length; j++) {
            if (arr->items[i] && res->items[j] &&
                strcmp((const char*)arr->items[i], (const char*)res->items[j]) == 0) {
                found = 1; break;
            }
        }
        if (!found) wolf_array_push(result, arr->items[i]);
    }
    return result;
}

void* wolf_array_merge(void* a, void* b) {
    void* result = wolf_array_create();
    if (a) { wolf_array_t* aa = (wolf_array_t*)a; for (int64_t i=0;i<aa->length;i++) wolf_array_push(result,aa->items[i]); }
    if (b) { wolf_array_t* bb = (wolf_array_t*)b; for (int64_t i=0;i<bb->length;i++) wolf_array_push(result,bb->items[i]); }
    return result;
}

void* wolf_array_slice(void* a, int64_t offset, int64_t len) {
    void* result = wolf_array_create();
    if (!a) return result;
    wolf_array_t* arr = (wolf_array_t*)a;
    if (offset < 0) offset = arr->length + offset;
    if (offset < 0) offset = 0;
    if (len < 0) len = arr->length - offset + len;
    if (len <= 0) return result;
    for (int64_t i = offset; i < offset + len && i < arr->length; i++)
        wolf_array_push(result, arr->items[i]);
    return result;
}

void wolf_sort(void* a) {
    if (!a) return;
    wolf_array_t* arr = (wolf_array_t*)a;
    for (int64_t i = 0; i < arr->length - 1; i++)
        for (int64_t j = 0; j < arr->length - i - 1; j++) {
            const char* s1 = arr->items[j]   ? (const char*)arr->items[j]   : "";
            const char* s2 = arr->items[j+1] ? (const char*)arr->items[j+1] : "";
            if (strcmp(s1, s2) > 0) { void* t=arr->items[j]; arr->items[j]=arr->items[j+1]; arr->items[j+1]=t; }
        }
}

void wolf_rsort(void* a) {
    if (!a) return;
    wolf_array_t* arr = (wolf_array_t*)a;
    for (int64_t i = 0; i < arr->length - 1; i++)
        for (int64_t j = 0; j < arr->length - i - 1; j++) {
            const char* s1 = arr->items[j]   ? (const char*)arr->items[j]   : "";
            const char* s2 = arr->items[j+1] ? (const char*)arr->items[j+1] : "";
            if (strcmp(s1, s2) < 0) { void* t=arr->items[j]; arr->items[j]=arr->items[j+1]; arr->items[j+1]=t; }
        }
}

double wolf_array_sum(void* a) {
    if (!a) return 0.0;
    wolf_array_t* arr = (wolf_array_t*)a;
    double sum = 0.0;
    for (int64_t i = 0; i < arr->length; i++)
        sum += wolf_value_unwrap_double(arr->items[i]);
    return sum;
}

void* wolf_array_keys(void* m) {
    if (!m) return wolf_array_create();
    wolf_map_t* map = (wolf_map_t*)m;
    void* result = wolf_array_create();
    for (int64_t i = 0; i < map->size; i++) wolf_array_push(result, (void*)map->keys[i]);
    return result;
}

void* wolf_array_values(void* m) {
    if (!m) return wolf_array_create();
    wolf_map_t* map = (wolf_map_t*)m;
    void* result = wolf_array_create();
    for (int64_t i = 0; i < map->size; i++) wolf_array_push(result, map->values[i]);
    return result;
}

void* wolf_array_diff(void* a, void* b) {
    void* result = wolf_array_create();
    if (!a) return result;
    wolf_array_t* arr_a = (wolf_array_t*)a;
    for (int64_t i = 0; i < arr_a->length; i++)
        if (!wolf_in_array((const char*)arr_a->items[i], b))
            wolf_array_push(result, arr_a->items[i]);
    return result;
}

void* wolf_array_intersect(void* a, void* b) {
    void* result = wolf_array_create();
    if (!a || !b) return result;
    wolf_array_t* arr_a = (wolf_array_t*)a;
    for (int64_t i = 0; i < arr_a->length; i++)
        if (wolf_in_array((const char*)arr_a->items[i], b))
            wolf_array_push(result, arr_a->items[i]);
    return result;
}

void* wolf_array_flip(void* a) {
    if (!a) return wolf_map_create();
    wolf_array_t* arr = (wolf_array_t*)a;
    void* result = wolf_map_create();
    for (int64_t i = 0; i < arr->length; i++) {
        char idx[32];
        snprintf(idx, sizeof(idx), "%lld", (long long)i);
        wolf_map_set(result, (const char*)arr->items[i], wolf_req_strdup(idx));
    }
    return result;
}

void* wolf_range(int64_t start, int64_t end) {
    void* result = wolf_array_create();
    int64_t step = start <= end ? 1 : -1;
    for (int64_t i = start; step > 0 ? i <= end : i >= end; i += step) {
        char* s = (char*)wolf_req_alloc(32);
        snprintf(s, 32, "%lld", (long long)i);
        wolf_array_push(result, s);
    }
    return result;
}

/* ========== Phase 2 Stdlib — Additional Array Functions ========== */
 
/* array_fill — fill array with $num copies of $value starting at $start index */
void* wolf_array_fill(int64_t start, int64_t num, const char* value) {
    void* result = wolf_array_create();
    if (num <= 0) return result;
    if (!value) value = "";
    for (int64_t i = 0; i < num; i++) {
        wolf_value_t* sv = wolf_val_make(WOLF_TYPE_STRING);
        sv->val.s = wolf_req_strdup(value);
        wolf_array_push(result, sv);
    }
    return result;
}
 
/* array_combine — create map from keys array + values array */
void* wolf_array_combine(void* keys, void* values) {
    void* result = wolf_map_create();
    if (!keys || !values) return result;
    wolf_array_t* karr = (wolf_array_t*)keys;
    wolf_array_t* varr = (wolf_array_t*)values;
    int64_t len = karr->length < varr->length ? karr->length : varr->length;
    for (int64_t i = 0; i < len; i++) {
        const char* k = karr->items[i] ? (const char*)karr->items[i] : "";
        wolf_map_set(result, k, varr->items[i]);
    }
    return result;
}
 
/* array_chunk — split array into chunks of $size, returns array of arrays */
void* wolf_array_chunk(void* a, int64_t size) {
    void* result = wolf_array_create();
    if (!a || size <= 0) return result;
    wolf_array_t* arr = (wolf_array_t*)a;
    void* chunk = NULL;
    for (int64_t i = 0; i < arr->length; i++) {
        if (i % size == 0) {
            chunk = wolf_array_create();
            wolf_array_push(result, chunk);
        }
        wolf_array_push(chunk, arr->items[i]);
    }
    return result;
}
 
/* array_column — pluck a single column from a 2D array of maps */
void* wolf_array_column(void* a, const char* col) {
    void* result = wolf_array_create();
    if (!a || !col) return result;
    wolf_array_t* arr = (wolf_array_t*)a;
    for (int64_t i = 0; i < arr->length; i++) {
        void* row = arr->items[i];
        if (!row) { wolf_array_push(result, NULL); continue; }
        /* Unwrap tagged map value if necessary */
        if (wolf_is_tagged_value(row)) {
            wolf_value_t* tagged = (wolf_value_t*)row;
            if (tagged->type == WOLF_TYPE_MAP) row = tagged->val.ptr;
            else { wolf_array_push(result, NULL); continue; }
        }
        void* val = wolf_map_get(row, col);
        if (!val) { wolf_array_push(result, NULL); continue; }
        /* wolf_map_get already unwraps tagged values to char* for string types */
        wolf_value_t* sv = wolf_val_make(WOLF_TYPE_STRING);
        sv->val.s = wolf_req_strdup((const char*)val);
        wolf_array_push(result, sv);
    }
    return result;
}
 
/* array_product — multiply all values in array */
double wolf_array_product(void* a) {
    if (!a) return 0.0;
    wolf_array_t* arr = (wolf_array_t*)a;
    if (arr->length == 0) return 0.0;
    double product = 1.0;
    for (int64_t i = 0; i < arr->length; i++)
        if (arr->items[i]) product *= atof((const char*)arr->items[i]);
    return product;
}
 
/* array_diff_key — items in a whose keys don't exist in b */
void* wolf_array_diff_key(void* a, void* b) {
    void* result = wolf_map_create();
    if (!a) return result;
    wolf_map_t* ma = (wolf_map_t*)a;
    wolf_map_t* mb = (wolf_map_t*)b;
    for (int64_t i = 0; i < ma->size; i++) {
        int found = 0;
        if (mb) {
            for (int64_t j = 0; j < mb->size; j++) {
                if (strcmp(ma->keys[i], mb->keys[j]) == 0) { found = 1; break; }
            }
        }
        if (!found) wolf_map_set(result, ma->keys[i], ma->values[i]);
    }
    return result;
}
 
/* array_splice — remove $length elements from $offset, return removed elements */
void* wolf_array_splice(void* a, int64_t offset, int64_t length) {
    void* removed = wolf_array_create();
    if (!a) return removed;
    wolf_array_t* arr = (wolf_array_t*)a;
    if (offset < 0) offset = arr->length + offset;
    if (offset < 0) offset = 0;
    if (offset >= arr->length) return removed;
    if (length < 0 || offset + length > arr->length)
        length = arr->length - offset;
    /* Collect removed elements */
    for (int64_t i = offset; i < offset + length; i++)
        wolf_array_push(removed, arr->items[i]);
    /* Shift remaining elements left */
    for (int64_t i = offset; i < arr->length - length; i++)
        arr->items[i] = arr->items[i + length];
    arr->length -= length;
    return removed;
}
 
/* array_pad — pad array to $size with $value */
void* wolf_array_pad(void* a, int64_t size, const char* value) {
    wolf_array_t* arr = a ? (wolf_array_t*)a : NULL;
    int64_t cur_len = arr ? arr->length : 0;
    int64_t abs_size = size < 0 ? -size : size;
    void* result = wolf_array_create();
    if (!value) value = "";
 
    if (size < 0) {
        /* Pad at beginning */
        for (int64_t i = 0; i < abs_size - cur_len; i++)
            wolf_array_push(result, wolf_req_strdup(value));
        if (arr) for (int64_t i = 0; i < arr->length; i++)
            wolf_array_push(result, arr->items[i]);
    } else {
        /* Pad at end */
        if (arr) for (int64_t i = 0; i < arr->length; i++)
            wolf_array_push(result, arr->items[i]);
        for (int64_t i = cur_len; i < abs_size; i++)
            wolf_array_push(result, wolf_req_strdup(value));
    }
    return result;
}
 
/* array_count_values — count occurrences of each value */
void* wolf_array_count_values(void* a) {
    void* result = wolf_map_create();
    if (!a) return result;
    wolf_array_t* arr = (wolf_array_t*)a;
    for (int64_t i = 0; i < arr->length; i++) {
        if (!arr->items[i]) continue;
        const char* key = (const char*)arr->items[i];
        void* existing = wolf_map_get(result, key);
        int64_t count = existing ? atoll((const char*)existing) + 1 : 1;
        wolf_map_set_int(result, key, count);
    }
    return result;
}
 
/* usort — sort with custom comparator (Wolf function pointer) */
/* Note: Wolf comparator support requires closure/function-pointer
 * wiring in the emitter. For now expose a string-length comparator
 * as a demonstration; full usort lands with closure support. */
 
/* array_rand — return one or more random keys from array */
int64_t wolf_array_rand_one(void* a) {
    if (!a) return 0;
    wolf_array_t* arr = (wolf_array_t*)a;
    if (arr->length == 0) return 0;
    wolf_crypto_init();
    uint32_t rnd;
    randombytes_buf(&rnd, sizeof(rnd));
    return (int64_t)(rnd % (uint32_t)arr->length);
}
 
/* array_sum already exists — add array_mean, array_median, array_mode
 * here since they logically belong with array operations */
 
double wolf_array_mean(void* a) {
    if (!a) return 0.0;
    wolf_array_t* arr = (wolf_array_t*)a;
    if (arr->length == 0) return 0.0;
    return wolf_array_sum(a) / (double)arr->length;
}
 
double wolf_array_median(void* a) {
    if (!a) return 0.0;
    wolf_array_t* arr = (wolf_array_t*)a;
    if (arr->length == 0) return 0.0;
    /* Copy values into a double array and sort */
    double* vals = (double*)wolf_req_alloc(arr->length * sizeof(double));
    for (int64_t i = 0; i < arr->length; i++)
        vals[i] = wolf_value_unwrap_double(arr->items[i]);
    /* Bubble sort — acceptable for small arrays */
    for (int64_t i = 0; i < arr->length - 1; i++)
        for (int64_t j = 0; j < arr->length - i - 1; j++)
            if (vals[j] > vals[j+1]) { double t = vals[j]; vals[j] = vals[j+1]; vals[j+1] = t; }
    int64_t mid = arr->length / 2;
    if (arr->length % 2 == 0) return (vals[mid-1] + vals[mid]) / 2.0;
    return vals[mid];
}
 
const char* wolf_array_mode(void* a) {
    if (!a) return wolf_req_strdup("");
    wolf_array_t* arr = (wolf_array_t*)a;
    if (arr->length == 0) return wolf_req_strdup("");
    /* Count occurrences of each value */
    const char* mode_val = arr->items[0] ? (const char*)arr->items[0] : "";
    int64_t mode_count = 0;
    for (int64_t i = 0; i < arr->length; i++) {
        if (!arr->items[i]) continue;
        const char* v = (const char*)arr->items[i];
        int64_t count = 0;
        for (int64_t j = 0; j < arr->length; j++)
            if (arr->items[j] && strcmp(v, (const char*)arr->items[j]) == 0) count++;
        if (count > mode_count) { mode_count = count; mode_val = v; }
    }
    return wolf_req_strdup(mode_val);
}
 
double wolf_array_variance(void* a) {
    if (!a) return 0.0;
    wolf_array_t* arr = (wolf_array_t*)a;
    if (arr->length == 0) return 0.0;
    double mean = wolf_array_mean(a);
    double sum = 0.0;
    for (int64_t i = 0; i < arr->length; i++) {
        double v = wolf_value_unwrap_double(arr->items[i]);
        double diff = v - mean;
        sum += diff * diff;
    }
    return sum / (double)arr->length;
}
 
double wolf_array_std_dev(void* a) {
    return sqrt(wolf_array_variance(a));
}
 
double wolf_array_percentile(void* a, double p) {
    if (!a) return 0.0;
    wolf_array_t* arr = (wolf_array_t*)a;
    if (arr->length == 0) return 0.0;
    if (p <= 0.0) p = 0.0;
    if (p >= 100.0) p = 100.0;
    double* vals = (double*)wolf_req_alloc(arr->length * sizeof(double));
    for (int64_t i = 0; i < arr->length; i++)
        vals[i] = arr->items[i] ? atof((const char*)arr->items[i]) : 0.0;
    /* Sort */
    for (int64_t i = 0; i < arr->length - 1; i++)
        for (int64_t j = 0; j < arr->length - i - 1; j++)
            if (vals[j] > vals[j+1]) { double t = vals[j]; vals[j] = vals[j+1]; vals[j+1] = t; }
    double idx = (p / 100.0) * (double)(arr->length - 1);
    int64_t lo = (int64_t)idx;
    int64_t hi = lo + 1;
    if (hi >= arr->length) return vals[arr->length - 1];
    double frac = idx - (double)lo;
    return vals[lo] + frac * (vals[hi] - vals[lo]);
}
 
/* ========== Map ========== */

void* wolf_map_create(void) {
    wolf_map_t* m = (wolf_map_t*)wolf_req_alloc(sizeof(wolf_map_t));
    m->capacity = 8;
    m->size     = 0;
    m->keys     = (char**)wolf_req_alloc(sizeof(char*) * m->capacity);
    m->values   = (void**)wolf_req_alloc(sizeof(void*) * m->capacity);
    return m;
}

void wolf_map_set_int(void* map_ptr, const char* key, int64_t value) {
    if (!map_ptr || !key) return;
    wolf_value_t* v = wolf_val_make(WOLF_TYPE_INT); v->val.i = value;
    wolf_map_t* m = (wolf_map_t*)map_ptr;
    for (int64_t i = 0; i < m->size; i++) if (strcmp(m->keys[i], key)==0) { m->values[i]=v; return; }
    if (m->size >= m->capacity) {
        size_t old_k = sizeof(char*) * m->capacity;
        size_t old_v = sizeof(void*) * m->capacity;
        m->capacity*=2;
        m->keys=(char**)wolf_req_realloc(m->keys, old_k, sizeof(char*)*m->capacity);
        m->values=(void**)wolf_req_realloc(m->values, old_v, sizeof(void*)*m->capacity);
    }
    m->keys[m->size] = wolf_req_strdup(key); m->values[m->size]=v; m->size++;
}

void wolf_map_set_float(void* map_ptr, const char* key, double value) {
    if (!map_ptr || !key) return;
    wolf_value_t* v = wolf_val_make(WOLF_TYPE_FLOAT); v->val.f = value;
    wolf_map_t* m = (wolf_map_t*)map_ptr;
    for (int64_t i = 0; i < m->size; i++) if (strcmp(m->keys[i], key)==0) { m->values[i]=v; return; }
    if (m->size >= m->capacity) {
        size_t old_k = sizeof(char*) * m->capacity;
        size_t old_v = sizeof(void*) * m->capacity;
        m->capacity*=2;
        m->keys=(char**)wolf_req_realloc(m->keys, old_k, sizeof(char*)*m->capacity);
        m->values=(void**)wolf_req_realloc(m->values, old_v, sizeof(void*)*m->capacity);
    }
    m->keys[m->size] = wolf_req_strdup(key); m->values[m->size]=v; m->size++;
}

void wolf_map_set_bool(void* map_ptr, const char* key, int value) {
    if (!map_ptr || !key) return;
    wolf_value_t* v = wolf_val_make(WOLF_TYPE_BOOL); v->val.b = value;
    wolf_map_t* m = (wolf_map_t*)map_ptr;
    for (int64_t i = 0; i < m->size; i++) if (strcmp(m->keys[i], key)==0) { m->values[i]=v; return; }
    if (m->size >= m->capacity) {
        size_t old_k = sizeof(char*) * m->capacity;
        size_t old_v = sizeof(void*) * m->capacity;
        m->capacity*=2;
        m->keys=(char**)wolf_req_realloc(m->keys, old_k, sizeof(char*)*m->capacity);
        m->values=(void**)wolf_req_realloc(m->values, old_v, sizeof(void*)*m->capacity);
    }
    m->keys[m->size] = wolf_req_strdup(key); m->values[m->size]=v; m->size++;
}

void wolf_map_set(void* map_ptr, const char* key, void* value) {
    if (!map_ptr || !key) return;
    wolf_map_t* m = (wolf_map_t*)map_ptr;
    for (int64_t i = 0; i < m->size; i++) if (strcmp(m->keys[i], key)==0) { m->values[i]=value; return; }
    if (m->size >= m->capacity) {
        size_t old_k = sizeof(char*) * m->capacity;
        size_t old_v = sizeof(void*) * m->capacity;
        m->capacity*=2;
        m->keys=(char**)wolf_req_realloc(m->keys, old_k, sizeof(char*)*m->capacity);
        m->values=(void**)wolf_req_realloc(m->values, old_v, sizeof(void*)*m->capacity);
    }
    m->keys[m->size] = wolf_req_strdup(key); m->values[m->size]=value; m->size++;
}

void* wolf_map_get(void* map_ptr, const char* key) {
    if (!map_ptr || !key) return NULL;
    wolf_map_t* m = (wolf_map_t*)map_ptr;
    for (int64_t i = 0; i < m->size; i++) {
        if (strcmp(m->keys[i], key) == 0) {
            void* val = m->values[i];
            if (!val || !wolf_is_tagged_value(val)) return val;
            wolf_value_t* tagged = (wolf_value_t*)val;
            switch (tagged->type) {
                case WOLF_TYPE_INT: { char* buf=(char*)wolf_req_alloc(32); snprintf(buf,32,"%lld",(long long)tagged->val.i); return buf; }
                case WOLF_TYPE_FLOAT: { char* buf=(char*)wolf_req_alloc(64); snprintf(buf,64,"%g",tagged->val.f); return buf; }
                case WOLF_TYPE_BOOL:   return wolf_req_strdup(tagged->val.b ? "true" : "false");
                case WOLF_TYPE_NULL:   return NULL;
                case WOLF_TYPE_STRING: return tagged->val.s ? (void*)tagged->val.s : (void*)"";
                case WOLF_TYPE_MAP:    return val;
                case WOLF_TYPE_ARRAY:  return val;
                default: return val;
            }
        }
    }
    return NULL;
}

int64_t wolf_math_randomint(int64_t max) {
    if (max <= 0) return 0;
    return rand() % max;
}

bool wolf_strings_isempty(const char* s) {
    return (!s || s[0] == '\0');
}

const char* wolf_strings_substring(const char* s, int64_t start, int64_t end) {
    if (!s) return "";
    int64_t len = end - start;
    if (len <= 0) return "";
    return wolf_substr(s, start, len);
}

const char* wolf_http_query(const char* key) {
    if (wolf_current_req_id < 0) return "";
    return wolf_http_req_query(wolf_current_req_id, key);
}

void* wolf_class_create(const char* name) { (void)name; return wolf_map_create(); }

/* ========== Conversions ========== */

const char* wolf_int_to_string(int64_t n) {
    char* buf = (char*)wolf_req_alloc(32);
    snprintf(buf, 32, "%lld", (long long)n);
    return buf;
}
const char* wolf_float_to_string(double f) {
    char* buf = (char*)wolf_req_alloc(64);
    snprintf(buf, 64, "%g", f);
    return buf;
}
const char* wolf_bool_to_string(int b) { return b ? "true" : "false"; }

/* ========== Memory ========== */

void* wolf_alloc(int64_t size) { return wolf_req_alloc((size_t)size); }
void  wolf_free(void* ptr)     { free(ptr); }

/* ========== HTTP Server (requires OS networking — disabled on bare-metal) ========== */
#ifndef WOLF_FREESTANDING

/* wolf_current_req_id and wolf_current_res_id are declared at the top of this file (line ~148) */


void wolf_set_current_context(void* req_id, void* res_id) {
    wolf_current_req_id = (intptr_t)req_id;
    wolf_current_res_id = (intptr_t)res_id;
}

const char* wolf_get_request_body(void) {
    if (wolf_current_req_id < 0) return "";
    return wolf_http_req_body(wolf_current_req_id);
}
const char* wolf_get_request_header(const char* key) {
    if (wolf_current_req_id < 0 || !key) return "";
    return wolf_http_req_header(wolf_current_req_id, key);
}
const char* wolf_get_request_method(void) {
    if (wolf_current_req_id < 0) return "";
    return wolf_http_req_method(wolf_current_req_id);
}
const char* wolf_get_request_path(void) {
    if (wolf_current_req_id < 0) return "";
    return wolf_http_req_path(wolf_current_req_id);
}

const char* wolf_input(const char* key) {
    const char* body = wolf_get_request_body();
    if (!body || strlen(body) == 0) return "";
    if (!key || strlen(key) == 0) return body;
    return body;
}

void wolf_http_response_code(int64_t code) {
    if (wolf_current_res_id < 0) return;
    wolf_http_res_status(wolf_current_res_id, code);
}

void wolf_http_write_response(const char* body) {
    if (wolf_current_res_id < 0) return;
    wolf_http_res_header(wolf_current_res_id, "Content-Type", "application/json");
    wolf_http_res_write(wolf_current_res_id, body);
}
void wolf_http_header(const char* key, const char* value) {
    if (wolf_current_res_id < 0) return;
    wolf_http_res_header(wolf_current_res_id, key, value);
}

void wolf_http_status(int64_t code) {
    if (wolf_current_res_id < 0) return;
    wolf_http_res_status(wolf_current_res_id, code);
}
static int alloc_http_context(int client_fd) {
    pthread_mutex_lock(&http_mutex);
    for (int i = 0; i < MAX_CONCURRENT_REQUESTS; i++) {
        if (!http_contexts[i].active) {
            memset(&http_contexts[i], 0, sizeof(wolf_http_context_t));
            http_contexts[i].active      = 1;
            http_contexts[i].client_fd   = client_fd;
            http_contexts[i].status_code = 200;
            pthread_mutex_unlock(&http_mutex);
            return i;
        }
    }
    pthread_mutex_unlock(&http_mutex);
    return -1;
}

static void free_http_context(int id) {
    if (id < 0 || id >= MAX_CONCURRENT_REQUESTS) return;
    pthread_mutex_lock(&http_mutex);
    wolf_http_context_t* ctx = &http_contexts[id];
    
    /* 
     * NOTE: We NO LONGER call free() on ctx->method, ctx->path, etc. here.
     * These are allocated via wolf_req_strdup() which places them in the
     * per-request arena. wolf_req_arena_flush() handles the actual freeing.
     * Manual free() here causes a double-free crash.
     */
    ctx->method = NULL;
    ctx->path   = NULL;
    ctx->query  = NULL;
    ctx->body   = NULL;
    ctx->header_count = 0;
    ctx->res_header_count = 0;
    ctx->res_body = NULL;
    ctx->active = 0;
    
    pthread_mutex_unlock(&http_mutex);
}

/* ---------------------------------------------------------------
 * wolfBMH — Boyer-Moore-Horspool search for a fixed-length pattern
 * inside a binary buffer.  Returns pointer to first match or NULL.
 * Average-case complexity: O(n / m) — dramatically better than the
 * naive O(n × m) memcmp scan for large uploads.
 * --------------------------------------------------------------- */
static const char* wolfBMH(const char* haystack, size_t hay_len,
                            const char* needle,   size_t ndl_len) {
    if (ndl_len == 0 || ndl_len > hay_len) return NULL;
    /* Build bad-character shift table */
    size_t skip[256];
    for (size_t i = 0; i < 256; i++) skip[i] = ndl_len;
    for (size_t i = 0; i < ndl_len - 1; i++)
        skip[(unsigned char)needle[i]] = ndl_len - 1 - i;
    /* Search */
    size_t i = ndl_len - 1;
    while (i < hay_len) {
        size_t j = ndl_len - 1;
        size_t k = i;
        while (j < ndl_len && haystack[k] == needle[j]) {
            if (j == 0) return haystack + k;
            j--; k--;
        }
        i += skip[(unsigned char)haystack[i]];
    }
    return NULL;
}

/* ---------------------------------------------------------------
 * wolf_parse_multipart — parse multipart/form-data body.
 * Boundary is extracted from the Content-Type header value, e.g.:
 *   "multipart/form-data; boundary=----WebKitFormBoundaryXXX"
 * All memory allocated via wolf_req_alloc (per-request arena).
 * Boundary search uses Boyer-Moore-Horspool (O(n/m) average).
 * --------------------------------------------------------------- */
static void wolf_parse_multipart(wolf_http_context_t* ctx,
                                  const char* ct_header,
                                  const char* body, size_t body_len) {
    /* Extract boundary string */
    const char* bp = strstr(ct_header, "boundary=");
    if (!bp) return;
    bp += 9;
    /* Strip optional quotes */
    char boundary[256];
    size_t bi = 0;
    while (*bp && *bp != ';' && *bp != '\r' && *bp != '\n' && bi < 254) {
        if (*bp != '"') { boundary[bi++] = *bp; }
        bp++;
    }
    boundary[bi] = '\0';
    if (bi == 0) return;

    /* Full delimiter: "--" + boundary */
    char delim[260];
    snprintf(delim, sizeof(delim), "--%s", boundary);
    size_t delim_len = strlen(delim);

    const char* p   = body;
    const char* end = body + body_len;

    while (p < end && ctx->upload_count < WOLF_MAX_UPLOADS) {
        /* Boyer-Moore-Horspool: find next boundary delimiter */
        const char* part_start = wolfBMH(p, (size_t)(end - p), delim, delim_len);
        if (!part_start) break;
        p = part_start + delim_len;

        /* End-of-multipart marker: "--" after boundary */
        if (p + 2 <= end && p[0] == '-' && p[1] == '-') break;
        if (p + 2 <= end && p[0] == '\r' && p[1] == '\n') p += 2;

        /* Parse part headers until blank line */
        const char* field_name   = NULL;
        const char* filename     = NULL;
        const char* part_ct      = "application/octet-stream";
        const char* part_hdr_end = NULL;

        const char* hp = p;
        while (hp < end) {
            /* Find end of this header line */
            const char* eol = NULL;
            for (const char* s = hp; s + 1 < end; s++) {
                if (s[0] == '\r' && s[1] == '\n') { eol = s; break; }
            }
            if (eol == hp) { /* Blank line = part header end */
                part_hdr_end = eol + 2; 
                break;
            }

            /* Copy header line for parsing */
            size_t hlen = (size_t)(eol - hp);
            char* hline = (char*)wolf_req_alloc(hlen + 1);
            memcpy(hline, hp, hlen);
            hline[hlen] = '\0';

            /* Content-Disposition */
            if (strncasecmp(hline, "Content-Disposition:", 20) == 0) {
                char* np = strstr(hline, "name=");
                if (np) {
                    np += 5;
                    int quoted = (*np == '"');
                    if (quoted) np++;
                    char* ne = np;
                    while (*ne && (quoted ? *ne != '"' : (*ne != ';' && *ne != '\r'))) ne++;
                    size_t nl = (size_t)(ne - np);
                    char* nbuf = (char*)wolf_req_alloc(nl + 1);
                    memcpy(nbuf, np, nl); nbuf[nl] = '\0';
                    field_name = nbuf;
                }
                char* fp = strstr(hline, "filename=");
                if (fp) {
                    fp += 9;
                    int quoted = (*fp == '"');
                    if (quoted) fp++;
                    char* fe = fp;
                    while (*fe && (quoted ? *fe != '"' : (*fe != ';' && *fe != '\r'))) fe++;
                    size_t fl = (size_t)(fe - fp);
                    char* fbuf = (char*)wolf_req_alloc(fl + 1);
                    memcpy(fbuf, fp, fl); fbuf[fl] = '\0';
                    filename = fbuf;
                }
            }

            /* Content-Type of part */
            if (strncasecmp(hline, "Content-Type:", 13) == 0) {
                char* ctv = hline + 13;
                while (*ctv == ' ') ctv++;
                part_ct = wolf_req_strdup(ctv);
            }

            hp = eol + 2;
        }
        if (!part_hdr_end || !field_name || !filename) {
            /* Not a file part — skip */
            p = part_hdr_end ? part_hdr_end : hp;
            continue;
        }

        /* Part body: from part_hdr_end to next delimiter */
        const char* data_start = part_hdr_end;
        const char* data_end   = end;
        for (const char* s = data_start; s + delim_len + 2 <= end; s++) {
            if (s[0]=='\r' && s[1]=='\n' && memcmp(s+2, delim, delim_len)==0) {
                data_end = s; break;
            }
        }

        size_t data_size = (size_t)(data_end - data_start);
        char*  data_buf  = (char*)wolf_req_alloc(data_size + 1);
        memcpy(data_buf, data_start, data_size);
        data_buf[data_size] = '\0';

        wolf_upload_t* up = &ctx->uploads[ctx->upload_count++];
        up->field_name   = field_name;
        up->filename     = filename;
        up->content_type = part_ct;
        up->data         = data_buf;
        up->size         = data_size;

        p = data_end;
    }
}

static void parse_http_request(int id, char* raw_req, size_t len) {
    wolf_http_context_t* ctx = &http_contexts[id];
    ctx->upload_count = 0;

    /* Split headers from body at \r\n\r\n */
    char* body_start = NULL;
    for (size_t i = 0; i + 3 < len; i++) {
        if (raw_req[i]=='\r' && raw_req[i+1]=='\n' &&
            raw_req[i+2]=='\r' && raw_req[i+3]=='\n') {
            raw_req[i] = '\0';
            body_start = raw_req + i + 4;
            break;
        }
    }
    size_t body_len = body_start ? (size_t)(len - (size_t)(body_start - raw_req)) : 0;
    ctx->body = wolf_req_strdup(body_start ? body_start : "");

    char* saveptr;
    char* line = strtok_r(raw_req, "\r\n", &saveptr);
    if (!line) return;
    char* l_save;
    char* method    = strtok_r(line, " ", &l_save);
    char* full_path = strtok_r(NULL, " ", &l_save);
    if (method) ctx->method = wolf_req_strdup(method);
    if (full_path) {
        char* q = strchr(full_path, '?');
        if (q) { *q = '\0'; ctx->path = wolf_req_strdup(full_path); ctx->query = wolf_req_strdup(q+1); }
        else   { ctx->path = wolf_req_strdup(full_path); ctx->query = wolf_req_strdup(""); }
    }
    const char* content_type_val = NULL;
    const char* upgrade_val = NULL;
    const char* ws_key_val = NULL;
    while ((line = strtok_r(NULL, "\r\n", &saveptr))) {
        char* colon = strchr(line, ':');
        if (colon && ctx->header_count < 32) {
            *colon = '\0'; char* val = colon + 1; while (*val==' ') val++;
            ctx->header_keys[ctx->header_count] = wolf_req_strdup(line);
            ctx->header_vals[ctx->header_count] = wolf_req_strdup(val);
            if (strcasecmp(line, "Content-Type") == 0) content_type_val = ctx->header_vals[ctx->header_count];
            if (strcasecmp(line, "Upgrade") == 0) upgrade_val = ctx->header_vals[ctx->header_count];
            if (strcasecmp(line, "Sec-WebSocket-Key") == 0) ws_key_val = ctx->header_vals[ctx->header_count];
            ctx->header_count++;
        }
    }
    if (upgrade_val && strcasecmp(upgrade_val, "websocket") == 0 && ws_key_val) {
        ctx->is_websocket = 1;
        ctx->ws_key = wolf_req_strdup(ws_key_val);
    }

    /* If multipart body, parse uploads */
    if (body_start && body_len > 0 && content_type_val &&
        strstr(content_type_val, "multipart/form-data")) {
        wolf_parse_multipart(ctx, content_type_val, body_start, body_len);
    } else {
    }
}

#ifndef WOLF_REQUEST_TIMEOUT_SEC
#define WOLF_REQUEST_TIMEOUT_SEC 30
#endif

void* wolf_ws_on_message(void* handler) {
    global_ws_handler = (wolf_ws_handler_t)handler;
    return NULL;
}

void* wolf_ws_send(int64_t req_id, const char* message) {
    if (req_id < 0 || req_id >= MAX_CONCURRENT_REQUESTS) return NULL;
    wolf_http_context_t* ctx = &http_contexts[req_id];
    if (!ctx->active || ctx->client_fd <= 0) return NULL;

    size_t len = message ? strlen(message) : 0;
    unsigned char header[10];
    int header_len = 0;

    header[0] = 0x81; // FIN + Text
    if (len <= 125) {
        header[1] = (unsigned char)len;
        header_len = 2;
    } else if (len <= 65535) {
        header[1] = 126;
        header[2] = (len >> 8) & 0xFF;
        header[3] = len & 0xFF;
        header_len = 4;
    } else {
        header[1] = 127;
        for (int i = 0; i < 8; i++) {
            header[9-i] = (len >> (i*8)) & 0xFF;
        }
        header_len = 10;
    }

    write(ctx->client_fd, header, header_len);
    if (len > 0) write(ctx->client_fd, message, len);
}


static void* http_worker(void* arg) {
#if !defined(WOLF_DB_POSTGRES) && !defined(WOLF_DB_MSSQL)
    mysql_thread_init();
#endif

    int id = (int)(intptr_t)arg;
    wolf_http_context_t* ctx = &http_contexts[id];

    /* ---- Request timeout via SO_RCVTIMEO ---- */
    struct timeval tv;
    tv.tv_sec  = WOLF_REQUEST_TIMEOUT_SEC;
    tv.tv_usec = 0;
    setsockopt(ctx->client_fd, SOL_SOCKET, SO_RCVTIMEO,
               (const char*)&tv, sizeof(tv));

    char buffer[WOLF_MAX_REQUEST_SIZE];
    memset(buffer, 0, sizeof(buffer));
    ssize_t bytes_read = read(ctx->client_fd, buffer, sizeof(buffer) - 1);

    if (bytes_read > 0) {
        parse_http_request(id, buffer, bytes_read);

        if (ctx->is_websocket) {
            /* Handshake */
            char magic[] = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11";
            char combined[256];
            snprintf(combined, sizeof(combined), "%s%s", ctx->ws_key, magic);
            unsigned char hash[20];
            EVP_MD_CTX *mdctx = EVP_MD_CTX_new();
            EVP_DigestInit_ex(mdctx, EVP_sha1(), NULL);
            EVP_DigestUpdate(mdctx, combined, strlen(combined));
            EVP_DigestFinal_ex(mdctx, hash, NULL);
            EVP_MD_CTX_free(mdctx);
            const char* accept_key = wolf_base64_encode_bin((const char*)hash, 20);
            char res[512];
            snprintf(res, sizeof(res),
                     "HTTP/1.1 101 Switching Protocols\r\n"
                     "Upgrade: websocket\r\n"
                     "Connection: Upgrade\r\n"
                     "Sec-WebSocket-Accept: %s\r\n\r\n",
                     accept_key);
            write(ctx->client_fd, res, strlen(res));
            
            /* Handover to non-blocking poller */
            wolf_ws_poller_add(ctx->client_fd, id);

            /* Decrement in-flight counter but DO NOT close fd or free context yet */
            __atomic_fetch_sub(&wolf_active_requests, 1, __ATOMIC_SEQ_CST);
            if (wolf_shutdown_requested && wolf_active_requests == 0) {
                pthread_mutex_lock(&wolf_drain_mutex);
                pthread_cond_signal(&wolf_drain_cond);
                pthread_mutex_unlock(&wolf_drain_mutex);
            }
            return NULL;
        }


        if (global_wolf_handler) {
            wolf_req_arena_init();
            wolf_set_current_context((void*)(intptr_t)id, (void*)(intptr_t)id);
            global_wolf_handler((int64_t)id, (int64_t)id);
        }

        const char* status_text = "OK";
        switch(ctx->status_code) {
            case 201: status_text = "Created"; break;
            case 204: status_text = "No Content"; break;
            case 301: status_text = "Moved Permanently"; break;
            case 302: status_text = "Found"; break; // Redirect
            case 400: status_text = "Bad Request"; break;
            case 401: status_text = "Unauthorized"; break;
            case 403: status_text = "Forbidden"; break;
            case 404: status_text = "Not Found"; break;
            case 500: status_text = "Internal Server Error"; break;
            case 503: status_text = "Service Unavailable"; break;
        }

        char res_header[2048];
        snprintf(res_header, sizeof(res_header),
                 "HTTP/1.1 %d %s\r\n", ctx->status_code, status_text);
        write(ctx->client_fd, res_header, strlen(res_header));

        for (int i = 0; i < ctx->res_header_count; i++) {
            snprintf(res_header, sizeof(res_header), "%s: %s\r\n",
                     ctx->res_header_keys[i], ctx->res_header_vals[i]);
            write(ctx->client_fd, res_header, strlen(res_header));
        }

        int body_len = ctx->res_body ? (int)strlen(ctx->res_body) : 0;
        snprintf(res_header, sizeof(res_header),
                 "Content-Length: %d\r\n\r\n", body_len);
        write(ctx->client_fd, res_header, strlen(res_header));

        if (body_len > 0)
            write(ctx->client_fd, ctx->res_body, body_len);

    } else if (bytes_read == 0) {
        /* Client closed connection before sending anything — ignore */
    } else {
        /* bytes_read < 0 */
        if (errno == EAGAIN || errno == EWOULDBLOCK) {
            /* Request timed out — return 408 */
            const char* timeout_resp =
                "HTTP/1.1 408 Request Timeout\r\n"
                "Content-Length: 0\r\n"
                "Connection: close\r\n\r\n";
            write(ctx->client_fd, timeout_resp, strlen(timeout_resp));
            WOLF_LOG("[WOLF-HTTP] request timeout on fd=%d\n", ctx->client_fd);
        }
        /* Other errors (ECONNRESET etc) — fall through and close */
    }

    close(ctx->client_fd);
    free_http_context(id);
    wolf_req_arena_flush();

    /* Decrement in-flight counter AFTER arena flush */
    __atomic_fetch_sub(&wolf_active_requests, 1, __ATOMIC_SEQ_CST);
    if (wolf_shutdown_requested && wolf_active_requests == 0) {
        pthread_mutex_lock(&wolf_drain_mutex);
        pthread_cond_signal(&wolf_drain_cond);
        pthread_mutex_unlock(&wolf_drain_mutex);
    }

#if !defined(WOLF_DB_POSTGRES) && !defined(WOLF_DB_MSSQL)
    mysql_thread_end();
#endif
    return NULL;
}

void wolf_http_serve(int64_t port, void* handler_ptr) {
    global_wolf_handler = (wolf_http_handler_t)handler_ptr;

#if !defined(WOLF_DB_POSTGRES) && !defined(WOLF_DB_MSSQL)
    mysql_library_init(0, NULL, NULL);
#endif
    
    wolf_crypto_init();
    
    /* ---- Install signal handlers ---- */
    struct sigaction sa;
    memset(&sa, 0, sizeof(sa));
    sa.sa_handler = wolf_signal_handler;
    sigemptyset(&sa.sa_mask);
    sa.sa_flags = 0;          /* No SA_RESTART — lets accept() return EINTR */
    sigaction(SIGTERM, &sa, NULL);
    sigaction(SIGINT,  &sa, NULL);

    /* Ignore SIGPIPE so writing to a closed socket doesn't kill the process */
    signal(SIGPIPE, SIG_IGN);

    /* ---- Create server socket ---- */
    int server_fd;
    struct sockaddr_in address;
    int opt = 1;

    if ((server_fd = socket(AF_INET, SOCK_STREAM, 0)) == 0) {
        perror("wolf_http: socket failed");
        exit(EXIT_FAILURE);
    }
    if (setsockopt(server_fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt))) {
        perror("wolf_http: setsockopt");
        exit(EXIT_FAILURE);
    }

    address.sin_family      = AF_INET;
    address.sin_addr.s_addr = INADDR_ANY;
    address.sin_port        = htons(port);

    if (bind(server_fd, (struct sockaddr*)&address, sizeof(address)) < 0) {
        perror("wolf_http: bind failed");
        exit(EXIT_FAILURE);
    }
    if (listen(server_fd, 1024) < 0) {
        perror("wolf_http: listen failed");
        exit(EXIT_FAILURE);
    }

    printf("🐺 Wolf HTTP Server running on port %d...\n", (int)port);
    fflush(stdout);
    printf("   Send SIGTERM or Ctrl+C to shut down gracefully.\n");

    /* Initialize Worker Pool and WebSocket Poller */
    wolf_worker_pool_init();
    wolf_ws_poller_init();
    wolf_ws_poller_start();

    /* ---- Accept loop ---- */
    while (!wolf_shutdown_requested) {
        struct sockaddr_in client_addr;
        socklen_t client_len = sizeof(client_addr);
        int client_fd = accept(server_fd,
                               (struct sockaddr*)&client_addr,
                               &client_len);

        if (client_fd < 0) {
            if (errno == EINTR) {
                /* Signal interrupted accept() — check shutdown flag */
                break;
            }
            /* Transient error (ECONNABORTED, etc.) — keep going */
            continue;
        }

        /* Reject new connections when shutting down */
        if (wolf_shutdown_requested) {
            const char* busy =
                "HTTP/1.1 503 Service Unavailable\r\n"
                "Content-Length: 0\r\n"
                "Connection: close\r\n\r\n";
            write(client_fd, busy, strlen(busy));
            close(client_fd);
            break;
        }

        int id = alloc_http_context(client_fd);
        if (id >= 0) {
            /* Increment in-flight counter BEFORE pushing task */
            __atomic_fetch_add(&wolf_active_requests, 1, __ATOMIC_SEQ_CST);

            wolf_task_t task = {WOLF_TASK_HTTP, (int64_t)id, NULL};
            wolf_task_push(task);
        } else {
            /* All slots full */
            const char* busy =
                "HTTP/1.1 503 Service Unavailable\r\n"
                "Content-Length: 0\r\n\r\n";
            write(client_fd, busy, strlen(busy));
            close(client_fd);
        }
    }

    /* ---- Shutdown sequence ---- */
    fprintf(stderr, "[Wolf] Stopping accept loop. Draining %d in-flight request(s)...\n",
            (int)wolf_active_requests);

    close(server_fd);

    /*
     * Wait for in-flight requests to finish using condition variable.
     * Each http_worker decrements wolf_active_requests on exit.
     * Timeout: WOLF_REQUEST_TIMEOUT_SEC + 2s grace.
     */
    pthread_mutex_lock(&wolf_drain_mutex);
    struct timespec ts;
    clock_gettime(CLOCK_REALTIME, &ts);
    ts.tv_sec += WOLF_REQUEST_TIMEOUT_SEC + 2;

    while (wolf_active_requests > 0) {
        int rc = pthread_cond_timedwait(&wolf_drain_cond, &wolf_drain_mutex, &ts);
        if (rc == ETIMEDOUT) break;
    }
    pthread_mutex_unlock(&wolf_drain_mutex);

    if (wolf_active_requests > 0) {
        fprintf(stderr, "[Wolf] Drain timeout — %d request(s) still in flight. Forcing exit.\n",
                (int)wolf_active_requests);
    } else {
        fprintf(stderr, "[Wolf] All requests drained.\n");
    }

    /* Destroy DB connection pool */
    wolf_db_pool_destroy();

#if !defined(WOLF_DB_POSTGRES) && !defined(WOLF_DB_MSSQL)
    mysql_library_end();
#endif

    fprintf(stderr, "[Wolf] Shutdown complete.\n");
    exit(0);
}

#endif /* WOLF_FREESTANDING — end of HTTP server block */

/* --- Request API (requires HTTP server context) --- */
#ifndef WOLF_FREESTANDING

const char* wolf_http_req_method(int64_t req_id) {
    if (req_id < 0 || req_id >= MAX_CONCURRENT_REQUESTS) return "";
    return http_contexts[req_id].method ? http_contexts[req_id].method : "";
}
const char* wolf_http_req_path(int64_t req_id) {
    if (req_id < 0 || req_id >= MAX_CONCURRENT_REQUESTS) return "";
    return http_contexts[req_id].path ? http_contexts[req_id].path : "";
}
const char* wolf_http_req_query(int64_t req_id, const char* key) {
    if (req_id < 0 || req_id >= MAX_CONCURRENT_REQUESTS || !key) return "";
    char* query = http_contexts[req_id].query;
    if (!query) return "";
    char* q_copy = wolf_req_strdup(query);
    char* saveptr;
    char* pair = strtok_r(q_copy, "&", &saveptr);
    while (pair) {
        char* eq = strchr(pair, '=');
        if (eq) { *eq = '\0'; if (strcmp(pair, key)==0) return wolf_req_strdup(eq+1); }
        pair = strtok_r(NULL, "&", &saveptr);
    }
    return "";
}
const char* wolf_http_req_header(int64_t req_id, const char* key) {
    if (req_id < 0 || req_id >= MAX_CONCURRENT_REQUESTS || !key) return "";
    wolf_http_context_t* ctx = &http_contexts[req_id];
    for (int i = 0; i < ctx->header_count; i++)
        if (strcasecmp(ctx->header_keys[i], key) == 0) return ctx->header_vals[i];
    return "";
}
const char* wolf_http_req_body(int64_t req_id) {
    if (req_id < 0 || req_id >= MAX_CONCURRENT_REQUESTS) return "";
    return http_contexts[req_id].body ? http_contexts[req_id].body : "";
}

static const char WOLF_B64_CHARS[] = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
static char* wolf_base64_encode_bin(const char* data, size_t len) {
    if (!data || len == 0) return wolf_req_strdup("");
    size_t out_len = 4 * ((len + 2) / 3);
    char* out = (char*)wolf_req_alloc(out_len + 1);
    char* p = out;
    for (size_t i = 0; i < len; i += 3) {
        uint32_t n = ((uint32_t)(unsigned char)data[i]) << 16;
        if (i + 1 < len) n |= ((uint32_t)(unsigned char)data[i+1]) << 8;
        if (i + 2 < len) n |= (uint32_t)(unsigned char)data[i+2];
        *p++ = WOLF_B64_CHARS[(n >> 18) & 0x3F];
        *p++ = WOLF_B64_CHARS[(n >> 12) & 0x3F];
        *p++ = (i + 1 < len) ? WOLF_B64_CHARS[(n >> 6) & 0x3F] : '=';
        *p++ = (i + 2 < len) ? WOLF_B64_CHARS[n & 0x3F] : '=';
    }
    *p = '\0';
    return out;
}

/* Returns JSON object with upload metadata, or "" if not found.
 * Format: {"name":"photo.jpg","type":"image/jpeg","size":12345,"data":"<base64>"}
 * The caller can then json_decode() it and use Wolf_File::Save() to persist. */
const char* wolf_http_req_file(int64_t req_id, const char* field_name) {
    if (req_id < 0 || req_id >= MAX_CONCURRENT_REQUESTS || !field_name) {
        return "";
    }
    wolf_http_context_t* ctx = &http_contexts[req_id];
    for (int i = 0; i < ctx->upload_count; i++) {
        wolf_upload_t* up = &ctx->uploads[i];
        if (strcmp(up->field_name, field_name) == 0) {
            /* Base64-encode the raw bytes using binary size */
            const char* b64 = wolf_base64_encode_bin(up->data, up->size);
            /* Build compact JSON using arena-based string buffer */
            wolf_strbuf_t* buf = wolf_strbuf_new();
            char sizebuf[32];
            snprintf(sizebuf, sizeof(sizebuf), "%zu", up->size);
            wolf_strbuf_append(buf, "{");
            wolf_strbuf_append(buf, "\"name\":\""); wolf_strbuf_append(buf, up->filename);     wolf_strbuf_append(buf, "\",");
            wolf_strbuf_append(buf, "\"type\":\""); wolf_strbuf_append(buf, up->content_type); wolf_strbuf_append(buf, "\",");
            wolf_strbuf_append(buf, "\"size\":");  wolf_strbuf_append(buf, sizebuf);           wolf_strbuf_append(buf, ",");
            wolf_strbuf_append(buf, "\"data\":\""); wolf_strbuf_append(buf, b64 ? b64 : "");  wolf_strbuf_append(buf, "\"");
            wolf_strbuf_append(buf, "}");
            const char* result = wolf_req_strdup(buf->data);
            return result;
        }
    }
    return "";
}

int64_t wolf_http_req_file_count(int64_t req_id) {
    if (req_id < 0 || req_id >= MAX_CONCURRENT_REQUESTS) return 0;
    return (int64_t)http_contexts[req_id].upload_count;
}

/* --- Response API --- */

void wolf_http_res_header(int64_t res_id, const char* key, const char* value) {
    if (res_id < 0 || res_id >= MAX_CONCURRENT_REQUESTS || !key || !value) return;
    wolf_http_context_t* ctx = &http_contexts[res_id];
    if (ctx->res_header_count < 32) {
        ctx->res_header_keys[ctx->res_header_count] = wolf_req_strdup(key);
        ctx->res_header_vals[ctx->res_header_count] = wolf_req_strdup(value);
        ctx->res_header_count++;
    }
}

void wolf_http_res_status(int64_t res_id, int64_t status_code) {
    if (res_id < 0 || res_id >= MAX_CONCURRENT_REQUESTS) return;
    http_contexts[res_id].status_code = (int)status_code;
}

void wolf_http_res_write(int64_t res_id, const char* body) {
    if (res_id < 0 || res_id >= MAX_CONCURRENT_REQUESTS || !body) return;
    wolf_http_context_t* ctx = &http_contexts[res_id];
    if (ctx->res_body) {
        size_t old_len = strlen(ctx->res_body);
        size_t new_len = strlen(body);
        char* new_body = wolf_req_alloc(old_len + new_len + 1);
        strcpy(new_body, ctx->res_body);
        strcat(new_body, body);
        ctx->res_body = new_body;
    } else {
        ctx->res_body = wolf_req_strdup(body);
    }
}

/* --- HTTP Client (libcurl integration) --- */

static char* wolf_json_escape(const char* s) {
    if (!s) return "";
    size_t len = strlen(s);
    char* res = wolf_req_alloc(len * 4 + 1);
    char* p = res;
    while (*s) {
        if (*s == '"') { *p++ = '\\'; *p++ = '"'; }
        else if (*s == '\\') { *p++ = '\\'; *p++ = '\\'; }
        else if (*s == '\n') { *p++ = '\\'; *p++ = 'n'; }
        else if (*s == '\r') { *p++ = '\\'; *p++ = 'r'; }
        else if (*s == '\t') { *p++ = '\\'; *p++ = 't'; }
        else { *p++ = *s; }
        s++;
    }
    *p = '\0';
    return res;
}

const char* wolf_http_get(const char* url) {
    wolf_ensure_curl();
    CURL *curl;
    CURLcode res;
    WolfHTTPClientResult chunk = { .data = malloc(1), .size = 0 };
    chunk.data[0] = '\0';

    curl = curl_easy_init();
    if (!curl) return "";

    curl_easy_setopt(curl, CURLOPT_URL, url);
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, wolf_http_client_write_cb);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, (void *)&chunk);
    curl_easy_setopt(curl, CURLOPT_USERAGENT, "wolf/1.0");
    curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 0L); // Default to skip for simple dev

    res = curl_easy_perform(curl);

    long status_code = 0;
    if (res == CURLE_OK) {
        curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &status_code);
    }

    /* Build JSON response in request arena */
    const char* escaped_body = wolf_json_escape(chunk.data);
    char* json = wolf_req_alloc(strlen(escaped_body) + 128);
    sprintf(json, "{\"status\":%ld,\"body\":\"%s\",\"error\":%d}", 
            status_code, escaped_body, (int)res);

    free(chunk.data);
    curl_easy_cleanup(curl);
    return json;
}

const char* wolf_http_post(const char* url, const char* body) {
    wolf_ensure_curl();
    CURL *curl;
    CURLcode res;
    WolfHTTPClientResult chunk = { .data = malloc(1), .size = 0 };
    chunk.data[0] = '\0';

    curl = curl_easy_init();
    if (!curl) return "";

    curl_easy_setopt(curl, CURLOPT_URL, url);
    curl_easy_setopt(curl, CURLOPT_POSTFIELDS, body);
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, wolf_http_client_write_cb);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, (void *)&chunk);
    curl_easy_setopt(curl, CURLOPT_USERAGENT, "wolf/1.0");
    curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 0L);

    res = curl_easy_perform(curl);

    long status_code = 0;
    if (res == CURLE_OK) {
        curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &status_code);
    }

    const char* escaped_body = wolf_json_escape(chunk.data);
    char* json = wolf_req_alloc(strlen(escaped_body) + 128);
    sprintf(json, "{\"status\":%ld,\"body\":\"%s\",\"error\":%d}", 
            status_code, escaped_body, (int)res);

    free(chunk.data);
    curl_easy_cleanup(curl);
    return json;
}

const char* wolf_http_put(const char* url, const char* body) {
    wolf_ensure_curl(); CURL *curl; CURLcode res;
    WolfHTTPClientResult chunk = { .data = malloc(1), .size = 0 };
    chunk.data[0] = '\0';
    curl = curl_easy_init(); if (!curl) return "";
    curl_easy_setopt(curl, CURLOPT_URL, url);
    curl_easy_setopt(curl, CURLOPT_CUSTOMREQUEST, "PUT");
    curl_easy_setopt(curl, CURLOPT_POSTFIELDS, body);
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, wolf_http_client_write_cb);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, (void *)&chunk);
    curl_easy_setopt(curl, CURLOPT_USERAGENT, "wolf/1.0");
    curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 0L);
    res = curl_easy_perform(curl);
    long code = 0; if (res == CURLE_OK) curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &code);
    const char* escaped = wolf_json_escape(chunk.data);
    char* json = wolf_req_alloc(strlen(escaped) + 128);
    sprintf(json, "{\"status\":%ld,\"body\":\"%s\",\"error\":%d}", code, escaped, (int)res);
    free(chunk.data); curl_easy_cleanup(curl);
    return json;
}

const char* wolf_http_delete(const char* url) {
    wolf_ensure_curl(); CURL *curl; CURLcode res;
    WolfHTTPClientResult chunk = { .data = malloc(1), .size = 0 };
    chunk.data[0] = '\0';
    curl = curl_easy_init(); if (!curl) return "";
    curl_easy_setopt(curl, CURLOPT_URL, url);
    curl_easy_setopt(curl, CURLOPT_CUSTOMREQUEST, "DELETE");
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, wolf_http_client_write_cb);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, (void *)&chunk);
    curl_easy_setopt(curl, CURLOPT_USERAGENT, "wolf/1.0");
    curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 0L);
    res = curl_easy_perform(curl);
    long code = 0; if (res == CURLE_OK) curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &code);
    const char* escaped = wolf_json_escape(chunk.data);
    char* json = wolf_req_alloc(strlen(escaped) + 128);
    sprintf(json, "{\"status\":%ld,\"body\":\"%s\",\"error\":%d}", code, escaped, (int)res);
    free(chunk.data); curl_easy_cleanup(curl);
    return json;
}

const char* wolf_http_patch(const char* url, const char* body) {
    wolf_ensure_curl(); CURL *curl; CURLcode res;
    WolfHTTPClientResult chunk = { .data = malloc(1), .size = 0 };
    chunk.data[0] = '\0';
    curl = curl_easy_init(); if (!curl) return "";
    curl_easy_setopt(curl, CURLOPT_URL, url);
    curl_easy_setopt(curl, CURLOPT_CUSTOMREQUEST, "PATCH");
    curl_easy_setopt(curl, CURLOPT_POSTFIELDS, body);
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, wolf_http_client_write_cb);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, (void *)&chunk);
    curl_easy_setopt(curl, CURLOPT_USERAGENT, "wolf/1.0");
    curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 0L);
    res = curl_easy_perform(curl);
    long code = 0; if (res == CURLE_OK) curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &code);
    const char* escaped = wolf_json_escape(chunk.data);
    char* json = wolf_req_alloc(strlen(escaped) + 128);
    sprintf(json, "{\"status\":%ld,\"body\":\"%s\",\"error\":%d}", code, escaped, (int)res);
    free(chunk.data); curl_easy_cleanup(curl);
    return json;
}

#endif /* WOLF_FREESTANDING — end of HTTP request/response API */

/* ========== Phase 1 Stdlib — Strings ========== */

const char* wolf_strtoupper(const char* s) { return wolf_string_upper(s); }
const char* wolf_strtolower(const char* s) { return wolf_string_lower(s); }

const char* wolf_ucfirst(const char* s) {
    if (!s || !*s) return s ? wolf_req_strdup(s) : wolf_req_strdup("");
    char* r = wolf_req_strdup(s);
    r[0] = (char)toupper((unsigned char)r[0]);
    return r;
}

const char* wolf_ucwords(const char* s) {
    if (!s) return wolf_req_strdup("");
    char* r = wolf_req_strdup(s);
    int cap = 1;
    for (int i = 0; r[i]; i++) {
        if (r[i]==' '||r[i]=='\t'||r[i]=='\n') cap=1;
        else if (cap) { r[i]=(char)toupper((unsigned char)r[i]); cap=0; }
    }
    return r;
}

const char* wolf_lcfirst(const char* s) {
    if (!s || !*s) return s ? wolf_req_strdup(s) : wolf_req_strdup("");
    char* r = wolf_req_strdup(s);
    r[0] = (char)tolower((unsigned char)r[0]);
    return r;
}

const char* wolf_trim(const char* s)  { return wolf_string_trim(s); }

const char* wolf_ltrim(const char* s) {
    if (!s) return wolf_req_strdup("");
    while (*s && isspace((unsigned char)*s)) s++;
    return wolf_req_strdup(s);
}

const char* wolf_rtrim(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len = strlen(s);
    while (len > 0 && isspace((unsigned char)s[len-1])) len--;
    char* r = (char*)wolf_req_alloc(len + 1);
    memcpy(r, s, len); r[len] = '\0';
    return r;
}

int wolf_str_contains(const char* s, const char* sub)    { return (!s||!sub) ? 0 : strstr(s,sub)!=NULL; }
int wolf_str_starts_with(const char* s, const char* pfx) { return (!s||!pfx) ? 0 : strncmp(s,pfx,strlen(pfx))==0; }
int wolf_str_ends_with(const char* s, const char* suf) {
    if (!s||!suf) return 0;
    size_t sl=strlen(s),ul=strlen(suf);
    if (ul>sl) return 0;
    return strcmp(s+sl-ul, suf)==0;
}

const char* wolf_str_replace(const char* find, const char* rep, const char* s) {
    if (!s||!find||!rep||!*find) return s ? wolf_req_strdup(s) : wolf_req_strdup("");
    size_t fl=strlen(find), rl=strlen(rep);
    int count=0;
    const char* p=s;
    while ((p=strstr(p,find))) { count++; p+=fl; }
    size_t new_len=strlen(s)+count*(rl-fl);
    char* result=(char*)wolf_req_alloc(new_len+1);
    char* w=result; p=s;
    while (*p) {
        if (strncmp(p,find,fl)==0) { memcpy(w,rep,rl); w+=rl; p+=fl; }
        else { *w++=*p++; }
    }
    *w='\0'; return result;
}

const char* wolf_str_repeat(const char* s, int64_t times) {
    if (!s||times<=0) return wolf_req_strdup("");
    size_t sl=strlen(s);
    char* r=(char*)wolf_req_alloc(sl*times+1); r[0]='\0';
    for (int64_t i=0;i<times;i++) strcat(r,s);
    return r;
}

const char* wolf_str_pad(const char* s, int64_t len, const char* pad) {
    if (!s) return wolf_req_strdup("");
    if (!pad||!*pad) pad=" ";
    size_t sl=strlen(s);
    if ((int64_t)sl>=len) return wolf_req_strdup(s);
    char* r=(char*)wolf_req_alloc(len+1);
    strcpy(r,s);
    size_t pl=strlen(pad);
    for (size_t pos=sl; (int64_t)pos<len; pos++) r[pos]=pad[(pos-sl)%pl];
    r[len]='\0'; return r;
}

void* wolf_explode(const char* sep, const char* s) {
    void* result=wolf_array_create();
    if (!s||!sep||!*sep) return result;
    size_t sep_len=strlen(sep);
    const char* p=s;
    while (1) {
        const char* found=strstr(p,sep);
        if (!found) { if (*p) wolf_array_push(result,wolf_req_strdup(p)); break; }
        size_t chunk_len=found-p;
        if (chunk_len>0) {
            char* chunk=(char*)wolf_req_alloc(chunk_len+1);
            memcpy(chunk,p,chunk_len); chunk[chunk_len]='\0';
            wolf_array_push(result,chunk);
        }
        p=found+sep_len;
    }
    return result;
}

const char* wolf_implode(const char* sep, void* arr) {
    if (!arr) return wolf_req_strdup("");
    if (!sep) sep="";
    wolf_array_t* a=(wolf_array_t*)arr;
    if (a->length==0) return wolf_req_strdup("");
    size_t sep_len=strlen(sep), total=0;
    for (int64_t i=0;i<a->length;i++) {
        if (a->items[i]) total+=strlen((const char*)a->items[i]);
        if (i<a->length-1) total+=sep_len;
    }
    char* result=(char*)wolf_req_alloc(total+1);
    char* w=result;
    for (int64_t i=0;i<a->length;i++) {
        if (a->items[i]) { size_t l=strlen((const char*)a->items[i]); memcpy(w,a->items[i],l); w+=l; }
        if (i<a->length-1) { memcpy(w,sep,sep_len); w+=sep_len; }
    }
    *w='\0'; return result;
}

const char* wolf_substr(const char* s, int64_t start, int64_t len) {
    if (!s) return wolf_req_strdup("");
    size_t sl=strlen(s);
    if (start<0) start=(int64_t)sl+start;
    if (start<0) start=0;
    if ((size_t)start>=sl) return wolf_req_strdup("");
    if (len<0) len=(int64_t)sl-start+len;
    if (len<=0) return wolf_req_strdup("");
    if ((size_t)(start+len)>sl) len=(int64_t)sl-start;
    char* r=(char*)wolf_req_alloc(len+1);
    memcpy(r,s+start,len); r[len]='\0';
    return r;
}

int64_t wolf_strpos(const char* s, const char* sub) {
    if (!s||!sub) return -1;
    const char* p=strstr(s,sub);
    return p ? (int64_t)(p-s) : -1;
}

int64_t wolf_strrpos(const char* s, const char* sub) {
    if (!s||!sub) return -1;
    int64_t last=-1; size_t sl=strlen(sub);
    const char* p=s;
    while ((p=strstr(p,sub))) { last=(int64_t)(p-s); p+=sl; }
    return last;
}

int64_t wolf_str_word_count(const char* s) {
    if (!s) return 0;
    int64_t count=0; int in_word=0;
    while (*s) { if (isspace((unsigned char)*s)) in_word=0; else if (!in_word) { in_word=1; count++; } s++; }
    return count;
}

int64_t wolf_strcmp(const char* a, const char* b) {
    if (!a) a=""; if (!b) b="";
    return (int64_t)strcmp(a,b);
}

const char* wolf_nl2br(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len=strlen(s), nl=0;
    for (size_t i=0;i<len;i++) if (s[i]=='\n') nl++;
    char* r=(char*)wolf_req_alloc(len+nl*5+1); char* w=r;
    for (size_t i=0;i<len;i++) {
        if (s[i]=='\n') { memcpy(w,"<br>\n",5); w+=5; } else *w++=s[i];
    }
    *w='\0'; return r;
}

const char* wolf_strip_tags(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len=strlen(s);
    char* r=(char*)wolf_req_alloc(len+1); char* w=r;
    int in_tag=0;
    for (size_t i=0;i<len;i++) {
        if (s[i]=='<') in_tag=1;
        else if (s[i]=='>') in_tag=0;
        else if (!in_tag) *w++=s[i];
    }
    *w='\0'; return r;
}

const char* wolf_htmlspecialchars(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len=strlen(s);
    char* r=(char*)wolf_req_alloc(len*6+1); char* w=r;
    for (size_t i=0;i<len;i++) {
        switch(s[i]) {
            case '&': memcpy(w,"&amp;",5); w+=5; break;
            case '<': memcpy(w,"&lt;",4);  w+=4; break;
            case '>': memcpy(w,"&gt;",4);  w+=4; break;
            case '"': memcpy(w,"&quot;",6);w+=6; break;
            case '\'':memcpy(w,"&#039;",6);w+=6; break;
            default:  *w++=s[i];
        }
    }
    *w='\0'; return r;
}

const char* wolf_addslashes(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len=strlen(s);
    char* r=(char*)wolf_req_alloc(len*2+1); char* w=r;
    for (size_t i=0;i<len;i++) {
        if (s[i]=='\''||s[i]=='"'||s[i]=='\\') *w++='\\';
        *w++=s[i];
    }
    *w='\0'; return r;
}

const char* wolf_stripslashes(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len=strlen(s);
    char* r=(char*)wolf_req_alloc(len+1); char* w=r;
    for (size_t i=0;i<len;i++) {
        if (s[i]=='\\'&&i+1<len) { i++; *w++=s[i]; } else *w++=s[i];
    }
    *w='\0'; return r;
}

const char* wolf_sprintf(const char* fmt, ...) {
    if (!fmt) return wolf_req_strdup("");
    va_list args;
    va_start(args, fmt);
    int len = vsnprintf(NULL, 0, fmt, args);
    va_end(args);
    if (len < 0) return wolf_req_strdup("");
    char* r = (char*)wolf_req_alloc(len + 1);
    va_start(args, fmt);
    vsnprintf(r, len + 1, fmt, args);
    va_end(args);
    return r;
}

/* ================================================================ *
 * STDLIB-01 — Additional String Functions                          *
 * ================================================================ */

/* wolf_str_ireplace — case-insensitive string replace */
const char* wolf_str_ireplace(const char* find, const char* rep, const char* s) {
    if (!s || !find || !rep || !*find) return s ? wolf_req_strdup(s) : wolf_req_strdup("");
    size_t fl = strlen(find), rl = strlen(rep), sl = strlen(s);
    /* Build a lowercase copy of both s and find for comparison */
    char* s_low   = (char*)wolf_req_alloc(sl + 1);
    char* find_low = (char*)wolf_req_alloc(fl + 1);
    for (size_t i = 0; i < sl; i++) s_low[i]   = (char)tolower((unsigned char)s[i]);
    for (size_t i = 0; i < fl; i++) find_low[i] = (char)tolower((unsigned char)find[i]);
    s_low[sl] = '\0'; find_low[fl] = '\0';

    /* Count occurrences */
    int count = 0;
    const char* p = s_low;
    while ((p = strstr(p, find_low))) { count++; p += fl; }
    if (count == 0) return wolf_req_strdup(s);

    /* Build result using positions from lowercase search but chars from original */
    size_t new_len = sl + count * (rl > fl ? rl - fl : 0) + 1;
    char* result = (char*)wolf_req_alloc(new_len);
    char* w = result;
    const char* orig = s;
    const char* low  = s_low;
    while (*low) {
        const char* found = strstr(low, find_low);
        if (!found) {
            size_t rest = strlen(orig);
            memcpy(w, orig, rest);
            w += rest;
            break;
        }
        size_t prefix = found - low;
        memcpy(w, orig, prefix);
        w += prefix;
        memcpy(w, rep, rl);
        w += rl;
        orig += prefix + fl;
        low   = found  + fl;
    }
    *w = '\0';
    return result;
}

/* wolf_htmlspecialchars_decode — decode HTML entities back to chars */
const char* wolf_htmlspecialchars_decode(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len = strlen(s);
    char* r = (char*)wolf_req_alloc(len + 1);
    char* w = r;
    for (size_t i = 0; i < len; ) {
        if (s[i] == '&') {
            if (strncmp(s+i, "&amp;",  5) == 0) { *w++ = '&';  i += 5; }
            else if (strncmp(s+i, "&lt;",   4) == 0) { *w++ = '<';  i += 4; }
            else if (strncmp(s+i, "&gt;",   4) == 0) { *w++ = '>';  i += 4; }
            else if (strncmp(s+i, "&quot;", 6) == 0) { *w++ = '"';  i += 6; }
            else if (strncmp(s+i, "&#039;", 6) == 0) { *w++ = '\''; i += 6; }
            else { *w++ = s[i++]; }
        } else {
            *w++ = s[i++];
        }
    }
    *w = '\0';
    return r;
}

/* wolf_similar_text — returns similarity percentage (0.0–100.0) */
double wolf_similar_text(const char* a, const char* b) {
    if (!a || !b) return 0.0;
    size_t la = strlen(a), lb = strlen(b);
    if (la == 0 && lb == 0) return 100.0;
    if (la == 0 || lb == 0) return 0.0;

    /* Find longest common substring */
    size_t longest = 0, pos_a = 0, pos_b = 0;
    for (size_t i = 0; i < la; i++) {
        for (size_t j = 0; j < lb; j++) {
            size_t l = 0;
            while (i + l < la && j + l < lb && a[i+l] == b[j+l]) l++;
            if (l > longest) { longest = l; pos_a = i; pos_b = j; }
        }
    }
    if (longest == 0) return 0.0;

    double common = (double)longest;

    /* Recurse on the parts before and after the common substring */
    if (pos_a > 0 && pos_b > 0) {
        char* sub_a = (char*)wolf_req_alloc(pos_a + 1);
        char* sub_b = (char*)wolf_req_alloc(pos_b + 1);
        memcpy(sub_a, a, pos_a); sub_a[pos_a] = '\0';
        memcpy(sub_b, b, pos_b); sub_b[pos_b] = '\0';
        common += wolf_similar_text(sub_a, sub_b) * (double)(pos_a + pos_b) / 100.0;
    }
    size_t after_a = la - pos_a - longest;
    size_t after_b = lb - pos_b - longest;
    if (after_a > 0 && after_b > 0) {
        char* sub_a = (char*)wolf_req_alloc(after_a + 1);
        char* sub_b = (char*)wolf_req_alloc(after_b + 1);
        memcpy(sub_a, a + pos_a + longest, after_a); sub_a[after_a] = '\0';
        memcpy(sub_b, b + pos_b + longest, after_b); sub_b[after_b] = '\0';
        common += wolf_similar_text(sub_a, sub_b) * (double)(after_a + after_b) / 100.0;
    }

    return (common * 2.0 / (double)(la + lb)) * 100.0;
}

/* wolf_wordwrap — wrap string at word boundary */
const char* wolf_wordwrap(const char* s, int64_t width, const char* brk, int cut_long) {
    if (!s) return wolf_req_strdup("");
    if (!brk || !*brk) brk = "\n";
    if (width <= 0) width = 75;

    size_t sl = strlen(s), bl = strlen(brk);
    /* Worst case: every character gets a break inserted */
    char* result = (char*)wolf_req_alloc(sl * (bl + 1) + 1);
    char* w = result;

    size_t line_len = 0;
    const char* p = s;

    while (*p) {
        /* Find next word boundary */
        const char* word_end = p;
        while (*word_end && *word_end != ' ' && *word_end != '\n') word_end++;
        size_t word_len = word_end - p;

        if (*p == '\n') {
            /* Hard newline — reset line length */
            *w++ = '\n';
            line_len = 0;
            p++;
            continue;
        }

        if (word_len == 0) {
            /* Space */
            if (line_len > 0) {
                *w++ = ' ';
                line_len++;
            }
            p++;
            continue;
        }

        if (line_len > 0 && line_len + 1 + word_len > (size_t)width) {
            /* Word doesn't fit — insert break */
            memcpy(w, brk, bl); w += bl;
            line_len = 0;
        } else if (line_len > 0) {
            *w++ = ' ';
            line_len++;
        }

        if (cut_long && word_len > (size_t)width) {
            /* Cut long word */
            while (word_len > 0) {
                size_t space_left = (size_t)width - line_len;
                if (space_left == 0) {
                    memcpy(w, brk, bl); w += bl;
                    line_len = 0;
                    space_left = (size_t)width;
                }
                size_t chunk = word_len < space_left ? word_len : space_left;
                memcpy(w, p, chunk); w += chunk;
                p        += chunk;
                word_len -= chunk;
                line_len += chunk;
            }
        } else {
            memcpy(w, p, word_len); w += word_len;
            line_len += word_len;
            p        += word_len;
        }
    }
    *w = '\0';
    return result;
}

/* wolf_quoted_printable_encode — RFC 2045 quoted-printable encoding */
const char* wolf_quoted_printable_encode(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len = strlen(s);
    /* Worst case: every byte becomes =XX (3 chars) plus soft line breaks */
    char* r = (char*)wolf_req_alloc(len * 4 + 1);
    char* w = r;
    int line_len = 0;

    for (size_t i = 0; i < len; i++) {
        unsigned char c = (unsigned char)s[i];

        /* Soft line break at 75 chars */
        if (line_len >= 75) {
            *w++ = '='; *w++ = '\n';
            line_len = 0;
        }

        if (c == '\r' && i + 1 < len && s[i+1] == '\n') {
            *w++ = '\r'; *w++ = '\n';
            line_len = 0;
            i++; /* skip \n */
        } else if (c == '\n') {
            *w++ = '\n';
            line_len = 0;
        } else if (c == '\t' || (c >= 33 && c <= 126 && c != '=')) {
            *w++ = (char)c;
            line_len++;
        } else {
            /* Encode as =XX */
            if (line_len >= 73) { *w++ = '='; *w++ = '\n'; line_len = 0; }
            sprintf(w, "=%02X", c);
            w += 3;
            line_len += 3;
        }
    }
    *w = '\0';
    return r;
}

/* ------------------------------------------------------------------ *
 * Regex — POSIX ERE via regex.h                                      *
 *                                                                     *
 * wolf_preg_match  — returns 1 if pattern matches, 0 if not          *
 * wolf_preg_match_all — returns count of all matches as JSON array   *
 * wolf_preg_replace — replace first match with replacement string    *
 * wolf_preg_split  — split string by pattern, returns wolf array     *
 *                                                                     *
 * Pattern format: the Wolf source passes "/pattern/flags" like PHP.  *
 * We strip the delimiters and parse flags (i = REG_ICASE).           *
 * ------------------------------------------------------------------ */

/* Strip /pattern/flags delimiters, return flags via out_flags */
static char* wolf_regex_parse_pattern(const char* pat, int* out_flags) {
    *out_flags = REG_EXTENDED;
    if (!pat || pat[0] != '/') {
        /* No delimiters — treat as raw pattern */
        return wolf_req_strdup(pat ? pat : "");
    }
    /* Find closing delimiter */
    size_t len = strlen(pat);
    size_t close = len - 1;
    while (close > 0 && pat[close] != '/') close--;
    if (close == 0) return wolf_req_strdup(pat + 1); /* malformed */

    /* Extract pattern between first and last / */
    size_t plen = close - 1;
    char* p = (char*)wolf_req_alloc(plen + 1);
    memcpy(p, pat + 1, plen);
    p[plen] = '\0';

    /* Parse flags after closing / */
    for (size_t i = close + 1; i < len; i++) {
        if (pat[i] == 'i') *out_flags |= REG_ICASE;
    }
    return p;
}

int wolf_preg_match(const char* pattern, const char* s) {
    if (!pattern || !s) return 0;
    int flags;
    char* pat = wolf_regex_parse_pattern(pattern, &flags);
    regex_t re;
    if (regcomp(&re, pat, flags) != 0) return 0;
    int result = (regexec(&re, s, 0, NULL, 0) == 0) ? 1 : 0;
    regfree(&re);
    return result;
}

/* wolf_preg_match_captures — returns JSON array of capture groups for first match */
const char* wolf_preg_match_captures(const char* pattern, const char* s) {
    if (!pattern || !s) return wolf_req_strdup("[]");
    int flags;
    char* pat = wolf_regex_parse_pattern(pattern, &flags);
    regex_t re;
    if (regcomp(&re, pat, flags) != 0) return wolf_req_strdup("[]");

    /* Count subexpressions — re.re_nsub gives the count */
    size_t nmatch = re.re_nsub + 1; /* +1 for full match */
    regmatch_t* matches = (regmatch_t*)wolf_req_alloc(nmatch * sizeof(regmatch_t));

    /* Build JSON array of matches */
    size_t out_cap = 256;
    char* json = (char*)wolf_req_alloc(out_cap);
    char* out = json;
    if (regexec(&re, s, nmatch, matches, 0) == 0) {
        char* w = json;
        *w++ = '[';
        for (size_t i = 0; i < nmatch; i++) {
            if (i > 0) *w++ = ',';
            if (matches[i].rm_so < 0) {
                memcpy(w, "null", 4); w += 4;
            } else {
                size_t mlen = matches[i].rm_eo - matches[i].rm_so;
                /* Ensure space */
                size_t used = w - json;
                if (used + mlen + 4 >= out_cap) {
                    out_cap = (used + mlen + 4) * 2;
                    char* nj = (char*)wolf_req_alloc(out_cap);
                    memcpy(nj, json, used);
                    json = nj; w = json + used;
                }
                *w++ = '"';
                memcpy(w, s + matches[i].rm_so, mlen); w += mlen;
                *w++ = '"';
            }
        }
        *w++ = ']'; *w = '\0';
        out = json;
    } else {
        out = wolf_req_strdup("[]");
    }

    regfree(&re);
    return out;
}

/* wolf_preg_match_all — returns count of all non-overlapping matches */
int64_t wolf_preg_match_all(const char* pattern, const char* s) {
    if (!pattern || !s) return 0;
    int flags;
    char* pat = wolf_regex_parse_pattern(pattern, &flags);
    regex_t re;
    if (regcomp(&re, pat, flags) != 0) return 0;

    int64_t count = 0;
    const char* p = s;
    regmatch_t m;
    while (regexec(&re, p, 1, &m, 0) == 0) {
        count++;
        if (m.rm_eo == m.rm_so) p++; /* zero-length match — advance */
        else                     p += m.rm_eo;
        if (!*p) break;
    }
    regfree(&re);
    return count;
}

/* wolf_preg_replace — replace all matches of pattern with replacement */
const char* wolf_preg_replace(const char* pattern, const char* rep, const char* s) {
    if (!pattern || !s) return s ? wolf_req_strdup(s) : wolf_req_strdup("");
    if (!rep) rep = "";
    int flags;
    char* pat = wolf_regex_parse_pattern(pattern, &flags);
    regex_t re;
    if (regcomp(&re, pat, flags) != 0) return wolf_req_strdup(s);

    size_t sl = strlen(s), rl = strlen(rep);
    size_t out_cap = sl * 2 + rl + 64;
    char* result = (char*)wolf_req_alloc(out_cap);
    char* w = result;
    const char* p = s;
    regmatch_t m;

    while (*p && regexec(&re, p, 1, &m, 0) == 0) {
        size_t used = w - result;
        size_t prefix = (size_t)m.rm_so;
        size_t needed = used + prefix + rl + (strlen(p) - m.rm_eo) + 4;
        if (needed >= out_cap) {
            out_cap = needed * 2;
            char* nr = (char*)wolf_req_alloc(out_cap);
            memcpy(nr, result, used);
            result = nr; w = result + used;
        }
        /* Copy prefix */
        memcpy(w, p, prefix); w += prefix;
        /* Copy replacement */
        memcpy(w, rep, rl); w += rl;
        /* Advance past match */
        if (m.rm_eo == m.rm_so) {
            if (*p) *w++ = *p++; /* avoid infinite loop on zero-length match */
        } else {
            p += m.rm_eo;
        }
    }
    /* Copy remainder */
    size_t rest = strlen(p);
    memcpy(w, p, rest); w += rest;
    *w = '\0';

    regfree(&re);
    return result;
}

/* wolf_preg_split — split string by regex pattern, returns wolf array */
void* wolf_preg_split(const char* pattern, const char* s) {
    void* result = wolf_array_create();
    if (!pattern || !s) {
        wolf_array_push(result, wolf_req_strdup(s ? s : ""));
        return result;
    }
    int flags;
    char* pat = wolf_regex_parse_pattern(pattern, &flags);
    regex_t re;
    if (regcomp(&re, pat, flags) != 0) {
        wolf_array_push(result, wolf_req_strdup(s));
        return result;
    }

    const char* p = s;
    regmatch_t m;
    while (*p && regexec(&re, p, 1, &m, 0) == 0) {
        size_t chunk = (size_t)m.rm_so;
        char* part = (char*)wolf_req_alloc(chunk + 1);
        memcpy(part, p, chunk); part[chunk] = '\0';
        wolf_array_push(result, part);
        if (m.rm_eo == m.rm_so) p++;
        else                     p += m.rm_eo;
    }
    /* Push remainder */
    wolf_array_push(result, wolf_req_strdup(p));

    regfree(&re);
    return result;
}


/* ========== Math Extras ========== */

double wolf_deg2rad(double deg) { return deg * 3.14159265358979323846 / 180.0; }
double wolf_rad2deg(double rad) { return rad * 180.0 / 3.14159265358979323846; }
double wolf_clamp(double n, double mn, double mx) { return n<mn?mn:(n>mx?mx:n); }

/* ========== Phase 3 Stdlib — Math Additions ========== */
 
/* rand_float — cryptographically secure random float 0.0–1.0 */
double wolf_rand_float(void) {
    wolf_crypto_init();
    uint32_t rnd;
    randombytes_buf(&rnd, sizeof(rnd));
    return (double)rnd / (double)UINT32_MAX;
}
 
/* rand_secure — cryptographically secure random integer in [min, max] */
int64_t wolf_rand_secure(int64_t min, int64_t max) {
    if (min >= max) return min;
    wolf_crypto_init();
    uint64_t range = (uint64_t)(max - min) + 1;
    uint64_t rnd;
    randombytes_buf(&rnd, sizeof(rnd));
    return min + (int64_t)(rnd % range);
}
 
/* wolf_math_pi already exists — add INF and NAN as functions
 * since Wolf doesn't have preprocessor constants yet */
double wolf_math_inf(void)  { return 1.0 / 0.0;  }
double wolf_math_nan(void)  { return 0.0 / 0.0;  }
int    wolf_is_nan(double v) { return v != v ? 1 : 0; }
int    wolf_is_inf(double v) { return (v == 1.0/0.0 || v == -1.0/0.0) ? 1 : 0; }
int    wolf_is_finite(double v) { return (!wolf_is_nan(v) && !wolf_is_inf(v)) ? 1 : 0; }
 
/* intdiv already exists — add log with base */
double wolf_math_log_base(double n, double base) {
    if (base <= 0.0 || base == 1.0) return 0.0;
    return log(n) / log(base);
}
 
/* hypot — hypotenuse */
double wolf_math_hypot(double a, double b) { return sqrt(a*a + b*b); }
 
/* ========== Type Casting ========== */

int64_t     wolf_intval(const char* s)  { return s ? (int64_t)atoll(s) : 0; }
double      wolf_floatval(const char* s){ return s ? atof(s) : 0.0; }
const char* wolf_strval(int64_t n)      { return wolf_int_to_string(n); }
int         wolf_boolval(const char* s) {
    if (!s||!*s||strcmp(s,"0")==0||strcmp(s,"false")==0) return 0;
    return 1;
}
int64_t wolf_intdiv(int64_t a, int64_t b) { return b==0?0:a/b; }
const char* wolf_gettype(const char* val) { return val ? "string" : "null"; }
int wolf_is_numeric(const char* s) {
    if (!s||!*s) return 0;
    if (*s=='-'||*s=='+') s++;
    int has_dot=0;
    while (*s) {
        if (*s=='.') { if (has_dot) return 0; has_dot=1; }
        else if (!isdigit((unsigned char)*s)) return 0;
        s++;
    }
    return 1;
}

/* ========== Encoding ========== */

static const char b64_table[] =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

const char* wolf_base64_encode(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len=strlen(s);
    size_t out_len=4*((len+2)/3);
    char* out=(char*)wolf_req_alloc(out_len+1); char* p=out;
    for (size_t i=0;i<len;i+=3) {
        uint32_t n=((uint32_t)(unsigned char)s[i])<<16;
        if (i+1<len) n|=((uint32_t)(unsigned char)s[i+1])<<8;
        if (i+2<len) n|=(uint32_t)(unsigned char)s[i+2];
        *p++=b64_table[(n>>18)&0x3F];
        *p++=b64_table[(n>>12)&0x3F];
        *p++=(i+1<len)?b64_table[(n>>6)&0x3F]:'=';
        *p++=(i+2<len)?b64_table[n&0x3F]:'=';
    }
    *p='\0'; return out;
}

static int b64_decode_char(char c) {
    if (c>='A'&&c<='Z') return c-'A';
    if (c>='a'&&c<='z') return c-'a'+26;
    if (c>='0'&&c<='9') return c-'0'+52;
    if (c=='+') return 62; if (c=='/') return 63; return -1;
}

const char* wolf_base64_decode(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len=strlen(s);
    char* out=(char*)wolf_req_alloc(len*3/4+1); char* p=out;
    for (size_t i=0;i<len;i+=4) {
        int a=b64_decode_char(s[i]),b=b64_decode_char(s[i+1]);
        int c=(i+2<len&&s[i+2]!='=')?b64_decode_char(s[i+2]):0;
        int d=(i+3<len&&s[i+3]!='=')?b64_decode_char(s[i+3]):0;
        if (a<0||b<0) break;
        *p++=(char)((a<<2)|(b>>4));
        if (s[i+2]!='=') *p++=(char)(((b&0xF)<<4)|(c>>2));
        if (s[i+3]!='=') *p++=(char)(((c&0x3)<<6)|d);
    }
    *p='\0'; return out;
}

const char* wolf_url_encode(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len=strlen(s);
    char* r=(char*)wolf_req_alloc(len*3+1); char* w=r;
    for (size_t i=0;i<len;i++) {
        unsigned char c=(unsigned char)s[i];
        if (isalnum(c)||c=='-'||c=='_'||c=='.'||c=='~') *w++=c;
        else { sprintf(w,"%%%02X",c); w+=3; }
    }
    *w='\0'; return r;
}

const char* wolf_url_decode(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len=strlen(s);
    char* r=(char*)wolf_req_alloc(len+1); char* w=r;
    for (size_t i=0;i<len;i++) {
        if (s[i]=='%'&&i+2<len) {
            char hex[3]={s[i+1],s[i+2],0};
            *w++=(char)strtol(hex,NULL,16); i+=2;
        } else if (s[i]=='+') { *w++=' '; }
        else { *w++=s[i]; }
    }
    *w='\0'; return r;
}

/* ================================================================ *
 * Wolf Crypto Layer                                                 *
 * Real implementations via libsodium + OpenSSL                     *
 * Linker flags: -lsodium -lssl -lcrypto                            *
 * ================================================================ */
 
void wolf_crypto_init(void) {
    if (sodium_init() < 0) {
        fprintf(stderr, "[WOLF-CRYPTO] libsodium init failed\n");
    }
}
 
/* --- Hashing --- */
 
static const char* wolf_evp_hash(const EVP_MD* md, const char* s) {
    if (!s) s = "";
    unsigned char digest[EVP_MAX_MD_SIZE];
    unsigned int  digest_len = 0;
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    if (!ctx) return wolf_req_strdup("");
    EVP_DigestInit_ex(ctx, md, NULL);
    EVP_DigestUpdate(ctx, s, strlen(s));
    EVP_DigestFinal_ex(ctx, digest, &digest_len);
    EVP_MD_CTX_free(ctx);
    char* hex = (char*)wolf_req_alloc(digest_len * 2 + 1);
    for (unsigned int i = 0; i < digest_len; i++)
        sprintf(hex + i * 2, "%02x", digest[i]);
    hex[digest_len * 2] = '\0';
    return hex;
}
 
const char* wolf_md5(const char* s)    { return wolf_evp_hash(EVP_md5(),    s); }
const char* wolf_sha256(const char* s) { return wolf_evp_hash(EVP_sha256(), s); }
const char* wolf_sha512(const char* s) { return wolf_evp_hash(EVP_sha512(), s); }
 
const char* wolf_hash(const char* algo, const char* s) {
    if (!algo) return wolf_req_strdup("");
    const EVP_MD* md = EVP_get_digestbyname(algo);
    if (!md) { fprintf(stderr, "[WOLF-CRYPTO] Unknown hash algo: %s\n", algo); return wolf_req_strdup(""); }
    return wolf_evp_hash(md, s);
}
 
const char* wolf_hash_hmac(const char* algo, const char* data, const char* key) {
    if (!algo || !data || !key) return wolf_req_strdup("");
    const EVP_MD* md = EVP_get_digestbyname(algo);
    if (!md) md = EVP_sha256();
    unsigned char digest[EVP_MAX_MD_SIZE];
    unsigned int  digest_len = 0;
    HMAC(md, key, strlen(key), (const unsigned char*)data, strlen(data), digest, &digest_len);
    char* hex = (char*)wolf_req_alloc(digest_len * 2 + 1);
    for (unsigned int i = 0; i < digest_len; i++) sprintf(hex + i * 2, "%02x", digest[i]);
    hex[digest_len * 2] = '\0';
    return hex;
}
 
int wolf_hash_equals(const char* known, const char* user) {
    if (!known || !user) return 0;
    size_t kl = strlen(known), ul = strlen(user);
    if (kl != ul) return 0;
    return sodium_memcmp(known, user, kl) == 0 ? 1 : 0;
}
 
/* --- Password Hashing — Argon2id --- */
 
const char* wolf_password_hash(const char* password) {
    if (!password) return wolf_req_strdup("");
    wolf_crypto_init();
    char* hash = (char*)wolf_req_alloc(crypto_pwhash_STRBYTES);
    if (crypto_pwhash_str(hash, password, strlen(password),
                          crypto_pwhash_OPSLIMIT_INTERACTIVE,
                          crypto_pwhash_MEMLIMIT_INTERACTIVE) != 0) {
        fprintf(stderr, "[WOLF-CRYPTO] Argon2id hashing failed\n");
        return wolf_req_strdup("");
    }
    return hash;
}
 
int wolf_password_verify(const char* password, const char* hash) {
    if (!password || !hash) return 0;
    wolf_crypto_init();
    return crypto_pwhash_str_verify(hash, password, strlen(password)) == 0 ? 1 : 0;
}
 
int wolf_password_needs_rehash(const char* hash) {
    if (!hash) return 1;
    wolf_crypto_init();
    return crypto_pwhash_str_needs_rehash(hash,
               crypto_pwhash_OPSLIMIT_INTERACTIVE,
               crypto_pwhash_MEMLIMIT_INTERACTIVE) == 1 ? 1 : 0;
}
 
/* --- Symmetric Encryption — XSalsa20-Poly1305 --- */
 
const char* wolf_encrypt(const char* data, const char* key) {
    if (!data || !key) return wolf_req_strdup("");
    wolf_crypto_init();
    unsigned char derived_key[crypto_secretbox_KEYBYTES];
    unsigned int dk_len = 0;
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    EVP_DigestInit_ex(ctx, EVP_sha256(), NULL);
    EVP_DigestUpdate(ctx, key, strlen(key));
    EVP_DigestFinal_ex(ctx, derived_key, &dk_len);
    EVP_MD_CTX_free(ctx);
    unsigned char nonce[crypto_secretbox_NONCEBYTES];
    randombytes_buf(nonce, sizeof(nonce));
    size_t data_len = strlen(data);
    size_t cipher_len = data_len + crypto_secretbox_MACBYTES;
    unsigned char* cipher = (unsigned char*)wolf_req_alloc(cipher_len);
    if (crypto_secretbox_easy(cipher, (const unsigned char*)data, data_len, nonce, derived_key) != 0)
        return wolf_req_strdup("");
    size_t blob_len = crypto_secretbox_NONCEBYTES + cipher_len;
    unsigned char* blob = (unsigned char*)wolf_req_alloc(blob_len);
    memcpy(blob, nonce, crypto_secretbox_NONCEBYTES);
    memcpy(blob + crypto_secretbox_NONCEBYTES, cipher, cipher_len);
    size_t b64_len = sodium_base64_encoded_len(blob_len, sodium_base64_VARIANT_ORIGINAL);
    char* b64 = (char*)wolf_req_alloc(b64_len);
    sodium_bin2base64(b64, b64_len, blob, blob_len, sodium_base64_VARIANT_ORIGINAL);
    return b64;
}
 
const char* wolf_decrypt(const char* data, const char* key) {
    if (!data || !key) return wolf_req_strdup("");
    wolf_crypto_init();
    unsigned char derived_key[crypto_secretbox_KEYBYTES];
    unsigned int dk_len = 0;
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    EVP_DigestInit_ex(ctx, EVP_sha256(), NULL);
    EVP_DigestUpdate(ctx, key, strlen(key));
    EVP_DigestFinal_ex(ctx, derived_key, &dk_len);
    EVP_MD_CTX_free(ctx);
    size_t b64_maxdec = strlen(data) * 3 / 4 + 4;
    unsigned char* blob = (unsigned char*)wolf_req_alloc(b64_maxdec);
    size_t blob_len = 0;
    if (sodium_base642bin(blob, b64_maxdec, data, strlen(data), NULL, &blob_len, NULL,
                          sodium_base64_VARIANT_ORIGINAL) != 0)
        return wolf_req_strdup("");
    if (blob_len < crypto_secretbox_NONCEBYTES + crypto_secretbox_MACBYTES)
        return wolf_req_strdup("");
    unsigned char* nonce  = blob;
    unsigned char* cipher = blob + crypto_secretbox_NONCEBYTES;
    size_t cipher_len     = blob_len - crypto_secretbox_NONCEBYTES;
    size_t plain_len      = cipher_len - crypto_secretbox_MACBYTES;
    char* plain = (char*)wolf_req_alloc(plain_len + 1);
    if (crypto_secretbox_open_easy((unsigned char*)plain, cipher, cipher_len,
                                   nonce, derived_key) != 0)
        return wolf_req_strdup("");
    plain[plain_len] = '\0';
    return plain;
}
 
/* --- Random / Token Generation --- */
 
const char* wolf_rand_bytes(int64_t length) {
    if (length <= 0) return wolf_req_strdup("");
    wolf_crypto_init();
    unsigned char* buf = (unsigned char*)wolf_req_alloc(length);
    randombytes_buf(buf, length);
    char* hex = (char*)wolf_req_alloc(length * 2 + 1);
    for (int64_t i = 0; i < length; i++) sprintf(hex + i * 2, "%02x", buf[i]);
    hex[length * 2] = '\0';
    return hex;
}
 
const char* wolf_rand_hex(int64_t length) {
    if (length <= 0) return wolf_req_strdup("");
    wolf_crypto_init();
    unsigned char* buf = (unsigned char*)wolf_req_alloc(length);
    randombytes_buf(buf, length);
    char* hex = (char*)wolf_req_alloc(length * 2 + 1);
    for (int64_t i = 0; i < length; i++) sprintf(hex + i * 2, "%02x", buf[i]);
    hex[length * 2] = '\0';
    return hex;
}
 
const char* wolf_rand_token(void) {
    wolf_crypto_init();
    unsigned char buf[32];
    randombytes_buf(buf, sizeof(buf));
    size_t b64_len = sodium_base64_encoded_len(sizeof(buf),
                         sodium_base64_VARIANT_URLSAFE_NO_PADDING);
    char* token = (char*)wolf_req_alloc(b64_len);
    sodium_bin2base64(token, b64_len, buf, sizeof(buf),
                      sodium_base64_VARIANT_URLSAFE_NO_PADDING);
    return token;
}
 
/* --- ID Generation --- */
 
const char* wolf_uuid_v4(void) {
    wolf_crypto_init();
    unsigned char rnd[16];
    randombytes_buf(rnd, sizeof(rnd));
    rnd[6] = (rnd[6] & 0x0F) | 0x40;
    rnd[8] = (rnd[8] & 0x3F) | 0x80;
    char* r = (char*)wolf_req_alloc(37);
    snprintf(r, 37,
        "%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
        rnd[0],rnd[1],rnd[2],rnd[3],rnd[4],rnd[5],rnd[6],rnd[7],
        rnd[8],rnd[9],rnd[10],rnd[11],rnd[12],rnd[13],rnd[14],rnd[15]);
    return r;
}
 
const char* wolf_uuid_v7(void) {
    wolf_crypto_init();
    struct timespec ts; clock_gettime(CLOCK_REALTIME, &ts);
    uint64_t ms = (uint64_t)ts.tv_sec * 1000ULL + (uint64_t)ts.tv_nsec / 1000000ULL;
    unsigned char rnd[10]; randombytes_buf(rnd, sizeof(rnd));
    unsigned char uuid[16];
    uuid[0]=(ms>>40)&0xFF; uuid[1]=(ms>>32)&0xFF; uuid[2]=(ms>>24)&0xFF;
    uuid[3]=(ms>>16)&0xFF; uuid[4]=(ms>>8)&0xFF;  uuid[5]=ms&0xFF;
    uuid[6]=(rnd[0]&0x0F)|0x70; uuid[7]=rnd[1];
    uuid[8]=(rnd[2]&0x3F)|0x80; uuid[9]=rnd[3];
    memcpy(uuid+10, rnd+4, 6);
    char* r = (char*)wolf_req_alloc(37);
    snprintf(r, 37,
        "%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
        uuid[0],uuid[1],uuid[2],uuid[3],uuid[4],uuid[5],uuid[6],uuid[7],
        uuid[8],uuid[9],uuid[10],uuid[11],uuid[12],uuid[13],uuid[14],uuid[15]);
    return r;
}
 
const char* wolf_nanoid(int64_t size) {
    wolf_crypto_init();
    if (size <= 0) size = 21;
    static const char alphabet[] =
        "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_-";
    unsigned char* rnd = (unsigned char*)wolf_req_alloc(size);
    randombytes_buf(rnd, size);
    char* id = (char*)wolf_req_alloc(size + 1);
    for (int64_t i = 0; i < size; i++) id[i] = alphabet[rnd[i] & 63];
    id[size] = '\0';
    return id;
}
 
const char* wolf_custom_id(const char* prefix, int64_t entropy) {
    if (!prefix) prefix = "";
    if (entropy <= 0) entropy = 12;
    const char* nano = wolf_nanoid(entropy);
    size_t total = strlen(prefix) + 1 + strlen(nano) + 1;
    char* id = (char*)wolf_req_alloc(total);
    snprintf(id, total, "%s_%s", prefix, nano);
    return id;
}
 
/* --- Encoding --- */
 
const char* wolf_base64_url_encode(const char* s) {
    if (!s) return wolf_req_strdup("");
    wolf_crypto_init();
    size_t len = strlen(s);
    size_t b64_len = sodium_base64_encoded_len(len, sodium_base64_VARIANT_URLSAFE_NO_PADDING);
    char* out = (char*)wolf_req_alloc(b64_len);
    sodium_bin2base64(out, b64_len, (const unsigned char*)s, len,
                      sodium_base64_VARIANT_URLSAFE_NO_PADDING);
    return out;
}
 
const char* wolf_base64_url_decode(const char* s) {
    if (!s) return wolf_req_strdup("");
    wolf_crypto_init();
    size_t max_len = strlen(s) * 3 / 4 + 4;
    unsigned char* buf = (unsigned char*)wolf_req_alloc(max_len + 1);
    size_t bin_len = 0;
    if (sodium_base642bin(buf, max_len, s, strlen(s), NULL, &bin_len, NULL,
                          sodium_base64_VARIANT_URLSAFE_NO_PADDING) != 0)
        return wolf_req_strdup("");
    buf[bin_len] = '\0';
    return (const char*)buf;
}
 
const char* wolf_hex_encode(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len = strlen(s);
    char* hex = (char*)wolf_req_alloc(len * 2 + 1);
    sodium_bin2hex(hex, len * 2 + 1, (const unsigned char*)s, len);
    return hex;
}
 
const char* wolf_hex_decode(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t hex_len = strlen(s);
    size_t bin_max = hex_len / 2 + 1;
    unsigned char* bin = (unsigned char*)wolf_req_alloc(bin_max + 1);
    size_t bin_len = 0;
    if (sodium_hex2bin(bin, bin_max, s, hex_len, NULL, &bin_len, NULL) != 0)
        return wolf_req_strdup("");
    bin[bin_len] = '\0';
    return (const char*)bin;
}
 
/* --- JWT — HMAC-SHA256 --- */
 
static const char* wolf_b64url(const unsigned char* data, size_t len) {
    wolf_crypto_init();
    size_t b64_len = sodium_base64_encoded_len(len, sodium_base64_VARIANT_URLSAFE_NO_PADDING);
    char* out = (char*)wolf_req_alloc(b64_len);
    sodium_bin2base64(out, b64_len, data, len, sodium_base64_VARIANT_URLSAFE_NO_PADDING);
    return out;
}
 
const char* wolf_jwt_encode(const char* payload, const char* secret) {
    if (!payload || !secret) return wolf_req_strdup("");
    wolf_crypto_init();
    const char* header_json = "{\"alg\":\"HS256\",\"typ\":\"JWT\"}";
    const char* header_b64  = wolf_b64url((const unsigned char*)header_json, strlen(header_json));
    const char* payload_b64 = wolf_b64url((const unsigned char*)payload, strlen(payload));
    size_t signing_len = strlen(header_b64) + 1 + strlen(payload_b64) + 1;
    char* signing_input = (char*)wolf_req_alloc(signing_len);
    snprintf(signing_input, signing_len, "%s.%s", header_b64, payload_b64);
    unsigned char sig[EVP_MAX_MD_SIZE]; unsigned int sig_len = 0;
    HMAC(EVP_sha256(), secret, strlen(secret),
         (const unsigned char*)signing_input, strlen(signing_input),
         sig, &sig_len);
    const char* sig_b64 = wolf_b64url(sig, sig_len);
    size_t token_len = strlen(header_b64) + 1 + strlen(payload_b64) + 1 + strlen(sig_b64) + 1;
    char* token = (char*)wolf_req_alloc(token_len);
    snprintf(token, token_len, "%s.%s.%s", header_b64, payload_b64, sig_b64);
    return token;
}
 
const char* wolf_jwt_decode(const char* token, const char* secret) {
    if (!token || !secret) return NULL;
    wolf_crypto_init();
    const char* dot1 = strchr(token, '.');
    if (!dot1) return NULL;
    const char* dot2 = strchr(dot1 + 1, '.');
    if (!dot2) return NULL;
    size_t header_len  = dot1 - token;
    size_t payload_len = dot2 - dot1 - 1;
    size_t sig_len     = strlen(dot2 + 1);
    char* header_b64  = (char*)wolf_req_alloc(header_len  + 1);
    char* payload_b64 = (char*)wolf_req_alloc(payload_len + 1);
    char* sig_b64     = (char*)wolf_req_alloc(sig_len     + 1);
    memcpy(header_b64,  token,    header_len);  header_b64[header_len]   = '\0';
    memcpy(payload_b64, dot1 + 1, payload_len); payload_b64[payload_len] = '\0';
    memcpy(sig_b64,     dot2 + 1, sig_len);     sig_b64[sig_len]         = '\0';
    size_t signing_len = header_len + 1 + payload_len + 1;
    char* signing_input = (char*)wolf_req_alloc(signing_len);
    snprintf(signing_input, signing_len, "%s.%s", header_b64, payload_b64);
    unsigned char expected_sig[EVP_MAX_MD_SIZE]; unsigned int expected_sig_len = 0;
    HMAC(EVP_sha256(), secret, strlen(secret),
         (const unsigned char*)signing_input, strlen(signing_input),
         expected_sig, &expected_sig_len);
    const char* expected_b64 = wolf_b64url(expected_sig, expected_sig_len);
    if (!wolf_hash_equals(expected_b64, sig_b64)) return NULL;
    size_t max_dec = payload_len * 3 / 4 + 4;
    unsigned char* decoded = (unsigned char*)wolf_req_alloc(max_dec + 1);
    size_t dec_len = 0;
    if (sodium_base642bin(decoded, max_dec, payload_b64, payload_len, NULL, &dec_len, NULL,
                          sodium_base64_VARIANT_URLSAFE_NO_PADDING) != 0)
        return NULL;
    decoded[dec_len] = '\0';
    return (const char*)decoded;
}
 
const char* wolf_jwt_decode_unverified(const char* token) {
    if (!token) return NULL;
    wolf_crypto_init();
    const char* dot1 = strchr(token, '.');
    if (!dot1) return NULL;
    const char* dot2 = strchr(dot1 + 1, '.');
    if (!dot2) return NULL;
    size_t payload_len = dot2 - dot1 - 1;
    size_t max_dec = payload_len * 3 / 4 + 4;
    unsigned char* decoded = (unsigned char*)wolf_req_alloc(max_dec + 1);
    size_t dec_len = 0;
    if (sodium_base642bin(decoded, max_dec, dot1 + 1, payload_len, NULL, &dec_len, NULL,
                          sodium_base64_VARIANT_URLSAFE_NO_PADDING) != 0)
        return NULL;
    decoded[dec_len] = '\0';
    return (const char*)decoded;
}
 
int wolf_jwt_expired(const char* token) {
    if (!token) return 1;
    const char* payload = wolf_jwt_decode_unverified(token);
    if (!payload) return 1;
    const char* exp_field = strstr(payload, "\"exp\":");
    if (!exp_field) return 0;
    exp_field += 6;
    while (*exp_field == ' ') exp_field++;
    int64_t exp = (int64_t)atoll(exp_field);
    return wolf_time_now() > exp ? 1 : 0;
}
 
/* --- Curve25519 ECDH --- */
 
const char* wolf_curve25519_keypair(void) {
    wolf_crypto_init();
    unsigned char pk[crypto_box_PUBLICKEYBYTES];
    unsigned char sk[crypto_box_SECRETKEYBYTES];
    crypto_box_keypair(pk, sk);
    char* pk_hex = (char*)wolf_req_alloc(crypto_box_PUBLICKEYBYTES * 2 + 1);
    char* sk_hex = (char*)wolf_req_alloc(crypto_box_SECRETKEYBYTES * 2 + 1);
    sodium_bin2hex(pk_hex, crypto_box_PUBLICKEYBYTES * 2 + 1, pk, crypto_box_PUBLICKEYBYTES);
    sodium_bin2hex(sk_hex, crypto_box_SECRETKEYBYTES * 2 + 1, sk, crypto_box_SECRETKEYBYTES);
    size_t out_len = strlen(pk_hex) + strlen(sk_hex) + 32;
    char* out = (char*)wolf_req_alloc(out_len);
    snprintf(out, out_len, "{\"public\":\"%s\",\"secret\":\"%s\"}", pk_hex, sk_hex);
    return out;
}
 
const char* wolf_curve25519_shared(const char* my_secret_hex, const char* their_public_hex) {
    if (!my_secret_hex || !their_public_hex) return wolf_req_strdup("");
    wolf_crypto_init();
    unsigned char sk[crypto_box_SECRETKEYBYTES];
    unsigned char pk[crypto_box_PUBLICKEYBYTES];
    unsigned char shared[crypto_scalarmult_BYTES];
    size_t sk_len = 0, pk_len = 0;
    sodium_hex2bin(sk, sizeof(sk), my_secret_hex,    strlen(my_secret_hex),    NULL, &sk_len, NULL);
    sodium_hex2bin(pk, sizeof(pk), their_public_hex, strlen(their_public_hex), NULL, &pk_len, NULL);
    if (crypto_scalarmult(shared, sk, pk) != 0) return wolf_req_strdup("");
    char* hex = (char*)wolf_req_alloc(crypto_scalarmult_BYTES * 2 + 1);
    sodium_bin2hex(hex, crypto_scalarmult_BYTES * 2 + 1, shared, crypto_scalarmult_BYTES);
    return hex;
}
 
/* --- RSA Sign / Verify --- */
 
const char* wolf_sign(const char* data, const char* privkey_pem) {
    if (!data || !privkey_pem) {
        WOLF_LOG("[WOLF-CRYPTO] sign error: missing data or key\n");
        return wolf_req_strdup("");
    }
    wolf_crypto_init();
    BIO* bio = BIO_new_mem_buf(privkey_pem, -1);
    EVP_PKEY* pkey = PEM_read_bio_PrivateKey(bio, NULL, NULL, NULL);
    BIO_free(bio);
    if (!pkey) {
        WOLF_LOG("[WOLF-CRYPTO] sign error: failed to read private key\n");
        return wolf_req_strdup("");
    }
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    if (!ctx) {
        EVP_PKEY_free(pkey);
        WOLF_LOG("[WOLF-CRYPTO] sign error: failed to create MD context\n");
        return wolf_req_strdup("");
    }
    if (EVP_DigestSignInit(ctx, NULL, EVP_sha256(), NULL, pkey) <= 0) {
        EVP_MD_CTX_free(ctx);
        EVP_PKEY_free(pkey);
        WOLF_LOG("[WOLF-CRYPTO] sign error: DigestSignInit failed\n");
        return wolf_req_strdup("");
    }
    if (EVP_DigestSignUpdate(ctx, data, strlen(data)) <= 0) {
        EVP_MD_CTX_free(ctx);
        EVP_PKEY_free(pkey);
        WOLF_LOG("[WOLF-CRYPTO] sign error: DigestSignUpdate failed\n");
        return wolf_req_strdup("");
    }
    size_t sig_len = 0;
    EVP_DigestSignFinal(ctx, NULL, &sig_len);
    unsigned char* sig = (unsigned char*)wolf_req_alloc(sig_len);
    if (!sig) {
        EVP_MD_CTX_free(ctx);
        EVP_PKEY_free(pkey);
        return wolf_req_strdup("");
    }
    if (EVP_DigestSignFinal(ctx, sig, &sig_len) <= 0) {
        EVP_MD_CTX_free(ctx);
        EVP_PKEY_free(pkey);
        WOLF_LOG("[WOLF-CRYPTO] sign error: DigestSignFinal failed\n");
        return wolf_req_strdup("");
    }
    EVP_MD_CTX_free(ctx);
    EVP_PKEY_free(pkey);
    return wolf_b64url(sig, sig_len);
}
 
int wolf_verify(const char* data, const char* signature_b64, const char* pubkey_pem) {
    if (!data || !signature_b64 || !pubkey_pem) {
        WOLF_LOG("[WOLF-CRYPTO] verify error: missing arguments\n");
        return 0;
    }
    wolf_crypto_init();
    BIO* bio = BIO_new_mem_buf(pubkey_pem, -1);
    EVP_PKEY* pkey = PEM_read_bio_PUBKEY(bio, NULL, NULL, NULL);
    BIO_free(bio);
    if (!pkey) {
        WOLF_LOG("[WOLF-CRYPTO] verify error: failed to read public key\n");
        return 0;
    }
    size_t sig_max = strlen(signature_b64) * 3 / 4 + 4;
    unsigned char* sig = (unsigned char*)wolf_req_alloc(sig_max);
    if (!sig) {
        EVP_PKEY_free(pkey);
        return 0;
    }
    size_t sig_len = 0;
    if (sodium_base642bin(sig, sig_max, signature_b64, strlen(signature_b64), NULL, &sig_len, NULL,
                      sodium_base64_VARIANT_URLSAFE_NO_PADDING) != 0) {
        EVP_PKEY_free(pkey);
        WOLF_LOG("[WOLF-CRYPTO] verify error: base64 decode failed\n");
        return 0;
    }
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    if (!ctx) {
        EVP_PKEY_free(pkey);
        return 0;
    }
    if (EVP_DigestVerifyInit(ctx, NULL, EVP_sha256(), NULL, pkey) <= 0) {
        EVP_MD_CTX_free(ctx);
        EVP_PKEY_free(pkey);
        WOLF_LOG("[WOLF-CRYPTO] verify error: DigestVerifyInit failed\n");
        return 0;
    }
    if (EVP_DigestVerifyUpdate(ctx, data, strlen(data)) <= 0) {
        EVP_MD_CTX_free(ctx);
        EVP_PKEY_free(pkey);
        return 0;
    }
    int result = EVP_DigestVerifyFinal(ctx, sig, sig_len) == 1 ? 1 : 0;
    EVP_MD_CTX_free(ctx);
    EVP_PKEY_free(pkey);
    return result;
}
 
/* ========== Output & Debugging ========== */

void wolf_dump(const char* val) {
    if (val) printf("[dump] string(%zu) \"%s\"\n", strlen(val), val);
    else printf("[dump] NULL\n");
}
void wolf_dd(const char* val) { wolf_dump(val); exit(0); }
void wolf_log_info(const char* msg)    { if (msg) fprintf(stderr,"[INFO] %s\n",msg); }
void wolf_log_warning(const char* msg) { if (msg) fprintf(stderr,"[WARN] %s\n",msg); }
void wolf_log_error(const char* msg)   { if (msg) fprintf(stderr,"[ERROR] %s\n",msg); }

const char* wolf_json_pretty(const char* json) {
    return json ? wolf_req_strdup(json) : wolf_req_strdup("null");
}

void* wolf_json_decode(const char* json) {
    if (!json) return wolf_map_create();
    while (*json && (*json==' '||*json=='\t'||*json=='\n')) json++;

    if (*json == '[') {
        void* arr = wolf_array_create();
        json++;
        while (*json) {
            while (*json&&(*json==' '||*json=='\t'||*json=='\n'||*json==',')) json++;
            if (*json==']') break;
            if (*json=='"') {
                json++;
                size_t vcap=256; char* val=(char*)wolf_req_alloc(vcap); int vi=0;
                while (*json&&*json!='"') {
                    if ((size_t)vi>=vcap-8) { size_t old_vcap=vcap; vcap*=2; char* nv=(char*)json_decode_realloc(val,old_vcap,vcap); if(!nv){break;} val=nv; }
                    if (*json=='\\'&&*(json+1)) {
                        json++;
                        switch(*json) {
                            case 'n': val[vi++]='\n'; break; case 't': val[vi++]='\t'; break;
                            case 'r': val[vi++]='\r'; break; case '"': val[vi++]='"';  break;
                            case '\\':val[vi++]='\\'; break; case '/': val[vi++]='/';  break;
                            case 'u': {
                                json++; unsigned int cp=0;
                                for(int ci=0;ci<4&&*json;ci++,json++){cp<<=4;char hx=*json;if(hx>='0'&&hx<='9')cp|=(hx-'0');else if(hx>='a'&&hx<='f')cp|=(hx-'a'+10);else if(hx>='A'&&hx<='F')cp|=(hx-'A'+10);}
                                json--;
                                if(cp>=0xD800&&cp<=0xDBFF&&json[1]=='\\'&&json[2]=='u') {
                                    json+=3; unsigned int low_cp=0;
                                    for(int ci=0;ci<4&&*json;ci++,json++){low_cp<<=4;char hx=*json;if(hx>='0'&&hx<='9')low_cp|=(hx-'0');else if(hx>='a'&&hx<='f')low_cp|=(hx-'a'+10);else if(hx>='A'&&hx<='F')low_cp|=(hx-'A'+10);}
                                    json--;
                                    if(low_cp>=0xDC00&&low_cp<=0xDFFF) cp=0x10000+((cp-0xD800)<<10)+(low_cp-0xDC00);
                                }
                                if((size_t)vi>=vcap-8){ size_t old_vcap=vcap; vcap*=2; char* nv=(char*)json_decode_realloc(val,old_vcap,vcap); if(!nv){break;} val=nv; }
                                if(cp<0x80){val[vi++]=(char)cp;}else if(cp<0x800){val[vi++]=(char)(0xC0|(cp>>6));val[vi++]=(char)(0x80|(cp&0x3F));}else if(cp<0x10000){val[vi++]=(char)(0xE0|(cp>>12));val[vi++]=(char)(0x80|((cp>>6)&0x3F));val[vi++]=(char)(0x80|(cp&0x3F));}else{val[vi++]=(char)(0xF0|(cp>>18));val[vi++]=(char)(0x80|((cp>>12)&0x3F));val[vi++]=(char)(0x80|((cp>>6)&0x3F));val[vi++]=(char)(0x80|(cp&0x3F));}
                                break;
                            }
                            default: val[vi++]=*json; break;
                        }
                    } else { val[vi++]=*json; }
                    json++;
                }
                if (val) val[vi]='\0';
                if (*json=='"') json++;
                wolf_value_t* sv=wolf_val_make(WOLF_TYPE_STRING);
                sv->val.s=val?wolf_req_strdup(val):wolf_req_strdup("");
                wolf_array_push(arr,sv);
            } else if (*json=='{') {
                const char* start=json; int depth=0;
                while(*json){if(*json=='{')depth++;else if(*json=='}'){depth--;if(depth==0){json++;break;}}json++;}
                char* nested=strndup(start,json-start); wolf_array_push(arr,wolf_json_decode(nested)); free(nested);
            } else if (*json=='[') {
                const char* start=json; int depth=0;
                while(*json){if(*json=='[')depth++;else if(*json==']'){depth--;if(depth==0){json++;break;}}json++;}
                char* nested=strndup(start,json-start); wolf_array_push(arr,wolf_json_decode(nested)); free(nested);
            } else if (*json=='t'&&strncmp(json,"true",4)==0) {
                json+=4; wolf_value_t* v=wolf_val_make(WOLF_TYPE_BOOL); v->val.b=1; wolf_array_push(arr,v);
            } else if (*json=='f'&&strncmp(json,"false",5)==0) {
                json+=5; wolf_value_t* v=wolf_val_make(WOLF_TYPE_BOOL); v->val.b=0; wolf_array_push(arr,v);
            } else if (*json=='n'&&strncmp(json,"null",4)==0) {
                json+=4; wolf_array_push(arr,NULL);
            } else {
                char num[64]={0}; int ni=0,is_float=0;
                if(*json=='-') num[ni++]=*json++;
                while(*json&&((*json>='0'&&*json<='9')||*json=='.')&&ni<63){if(*json=='.') is_float=1; num[ni++]=*json++;}
                wolf_value_t* v=wolf_val_make(is_float?WOLF_TYPE_FLOAT:WOLF_TYPE_INT);
                if(is_float) v->val.f=atof(num); else v->val.i=atoll(num);
                wolf_array_push(arr,v);
            }
        }
        return arr;
    }

    if (*json != '{') return wolf_map_create();

    void* map = wolf_map_create();
    json++;

    while (*json) {
        while (*json&&(*json==' '||*json=='\t'||*json=='\n'||*json==',')) json++;
        if (*json=='}') break;
        if (*json!='"') break;
        json++;

        size_t key_cap=256; char* key=(char*)wolf_req_alloc(key_cap); if(!key) break;
        int ki=0;
        while (*json&&*json!='"') {
            if((size_t)ki>=key_cap-1){size_t old_sz=key_cap;key_cap*=2;char* nk=(char*)json_decode_realloc(key,old_sz,key_cap);if(!nk){free(key);break;}key=nk;}
            key[ki++]=*json++;
        }
        key[ki]='\0';
        if (*json=='"') json++;
        while (*json&&(*json==' '||*json=='\t'||*json==':')) json++;

        if (*json=='"') {
            json++;
            size_t val_cap=256; char* val=(char*)wolf_req_alloc(val_cap); if(!val){break;} int vi=0;
            while (*json&&*json!='"') {
                if((size_t)vi>=val_cap-8){size_t old_sz=val_cap;val_cap*=2;char* nv=(char*)json_decode_realloc(val,old_sz,val_cap);if(!nv){free(val);val=NULL;break;}val=nv;}
                if(*json=='\\'&&*(json+1)){
                    json++;
                    switch(*json){
                        case 'n':val[vi++]='\n';break; case 't':val[vi++]='\t';break;
                        case 'r':val[vi++]='\r';break; case '"':val[vi++]='"'; break;
                        case '\\':val[vi++]='\\';break; case '/':val[vi++]='/'; break;
                        case 'u': {
                            json++; unsigned int cp=0;
                            for(int ci=0;ci<4&&*json;ci++,json++){cp<<=4;char hx=*json;if(hx>='0'&&hx<='9')cp|=(hx-'0');else if(hx>='a'&&hx<='f')cp|=(hx-'a'+10);else if(hx>='A'&&hx<='F')cp|=(hx-'A'+10);}
                            json--;
                            if(cp>=0xD800&&cp<=0xDBFF&&json[1]=='\\'&&json[2]=='u') {
                                json+=3; unsigned int low_cp=0;
                                for(int ci=0;ci<4&&*json;ci++,json++){low_cp<<=4;char hx=*json;if(hx>='0'&&hx<='9')low_cp|=(hx-'0');else if(hx>='a'&&hx<='f')low_cp|=(hx-'a'+10);else if(hx>='A'&&hx<='F')low_cp|=(hx-'A'+10);}
                                json--;
                                if(low_cp>=0xDC00&&low_cp<=0xDFFF) cp=0x10000+((cp-0xD800)<<10)+(low_cp-0xDC00);
                            }
                            if((size_t)vi>=val_cap-8){size_t old_sz=val_cap;val_cap*=2;char* nv=(char*)json_decode_realloc(val,old_sz,val_cap);if(!nv){free(val);val=NULL;break;}val=nv;}
                            if(cp<0x80){val[vi++]=(char)cp;}else if(cp<0x800){val[vi++]=(char)(0xC0|(cp>>6));val[vi++]=(char)(0x80|(cp&0x3F));}else if(cp<0x10000){val[vi++]=(char)(0xE0|(cp>>12));val[vi++]=(char)(0x80|((cp>>6)&0x3F));val[vi++]=(char)(0x80|(cp&0x3F));}else{val[vi++]=(char)(0xF0|(cp>>18));val[vi++]=(char)(0x80|((cp>>12)&0x3F));val[vi++]=(char)(0x80|((cp>>6)&0x3F));val[vi++]=(char)(0x80|(cp&0x3F));}
                            break;
                        }
                        default: val[vi++]=*json; break;
                    }
                } else { val[vi++]=*json; }
                json++;
            }
            if(val) val[vi]='\0';
            if(*json=='"') json++;
            wolf_value_t* sv=wolf_val_make(WOLF_TYPE_STRING);
            sv->val.s=(val&&vi>0)?wolf_req_strdup(val):wolf_req_strdup("");
            wolf_map_set(map,key,sv);
        } else if (*json=='t'&&strncmp(json,"true",4)==0)  { json+=4; wolf_map_set_bool(map,key,1); }
        else if (*json=='f'&&strncmp(json,"false",5)==0)   { json+=5; wolf_map_set_bool(map,key,0); }
        else if (*json=='n'&&strncmp(json,"null",4)==0)    { json+=4; wolf_map_set(map,key,NULL); }
        else if (*json=='[') {
            const char* start=json; int depth=0;
            while(*json){if(*json=='[')depth++;else if(*json==']'){depth--;if(depth==0){json++;break;}}json++;}
            char* nested=strndup(start,json-start);
            wolf_value_t* v=wolf_val_make(WOLF_TYPE_ARRAY); v->val.ptr=wolf_json_decode(nested);
            free(nested); wolf_map_set(map,key,v);
        } else if (*json=='{') {
            int depth=0; const char* start=json;
            while(*json){if(*json=='{')depth++;else if(*json=='}'){depth--;if(depth==0){json++;break;}}json++;}
            char* nested=strndup(start,json-start);
            wolf_map_set(map,key,wolf_json_decode(nested)); free(nested);
        } else {
            char num[64]={0}; int ni=0,is_float=0;
            if(*json=='-') num[ni++]=*json++;
            while(*json&&((*json>='0'&&*json<='9')||*json=='.')&&ni<63){if(*json=='.') is_float=1; num[ni++]=*json++;}
            if(is_float) wolf_map_set_float(map,key,atof(num));
            else         wolf_map_set_int(map,key,atoll(num));
        }
    }
    return map;
}

/* ========== Phase 3 Stdlib — Date/Time Extras ========== */

int64_t wolf_time_ms(void) {
    struct timespec ts; clock_gettime(CLOCK_REALTIME,&ts);
    return (int64_t)(ts.tv_sec*1000LL+ts.tv_nsec/1000000LL);
}
int64_t wolf_time_ns(void) {
    struct timespec ts; clock_gettime(CLOCK_REALTIME,&ts);
    return (int64_t)(ts.tv_sec*1000000000LL+ts.tv_nsec);
}
int64_t wolf_mktime(int64_t hour,int64_t min,int64_t sec,int64_t mon,int64_t day,int64_t year) {
    struct tm t={0};
    t.tm_hour=(int)hour; t.tm_min=(int)min; t.tm_sec=(int)sec;
    t.tm_mon=(int)mon-1; t.tm_mday=(int)day; t.tm_year=(int)year-1900; t.tm_isdst=-1;
    return (int64_t)mktime(&t);
}
int64_t wolf_date_diff(int64_t ts1, int64_t ts2) { return ts2>ts1?ts2-ts1:ts1-ts2; }
const char* wolf_date_format(int64_t timestamp, const char* format) { return wolf_time_date(format,timestamp); }
int64_t wolf_day_of_week(int64_t timestamp) {
    time_t t=(time_t)timestamp; struct tm* info=localtime(&t);
    return (int64_t)info->tm_wday;
}
int64_t wolf_days_in_month(int64_t month, int64_t year) {
    int days_per_month[]={31,28,31,30,31,30,31,31,30,31,30,31};
    if(month<1||month>12) return 0;
    int d=days_per_month[month-1];
    if(month==2&&(year%4==0&&(year%100!=0||year%400==0))) d=29;
    return (int64_t)d;
}
int64_t wolf_is_leap_year(int64_t year) {
    return (year%4==0&&(year%100!=0||year%400==0));
}
int64_t wolf_strtotime(const char* str) {
    if (!str) return 0;
    struct tm t={0};
    if (sscanf(str,"%d-%d-%d %d:%d:%d",&t.tm_year,&t.tm_mon,&t.tm_mday,&t.tm_hour,&t.tm_min,&t.tm_sec)>=3)
        { t.tm_year-=1900; t.tm_mon-=1; t.tm_isdst=-1; return (int64_t)mktime(&t); }
    if (sscanf(str,"%d-%d-%d",&t.tm_year,&t.tm_mon,&t.tm_mday)==3)
        { t.tm_year-=1900; t.tm_mon-=1; t.tm_isdst=-1; return (int64_t)mktime(&t); }
    if (strcmp(str,"now")==0) return (int64_t)time(NULL);
    if (strcmp(str,"today")==0) { time_t now=time(NULL); struct tm* tn=localtime(&now); tn->tm_hour=tn->tm_min=tn->tm_sec=0; return (int64_t)mktime(tn); }
    if (strcmp(str,"tomorrow")==0) return (int64_t)time(NULL)+86400;
    if (strcmp(str,"yesterday")==0) return (int64_t)time(NULL)-86400;
    int n;
    if (sscanf(str,"+%d days",&n)==1)  return (int64_t)time(NULL)+n*86400;
    if (sscanf(str,"-%d days",&n)==1)  return (int64_t)time(NULL)-n*86400;
    if (sscanf(str,"+%d hours",&n)==1) return (int64_t)time(NULL)+n*3600;
    if (sscanf(str,"-%d hours",&n)==1) return (int64_t)time(NULL)-n*3600;
    return 0;
}

/* ========== Phase 3 Stdlib — Validation ========== */

int wolf_is_email(const char* s) {
    if (!s) return 0;
    const char* at=strchr(s,'@');
    if (!at||at==s) return 0;
    const char* dot=strchr(at,'.');
    if (!dot||dot==at+1||*(dot+1)=='\0') return 0;
    for (const char* p=s;*p;p++) if (isspace((unsigned char)*p)) return 0;
    return 1;
}
int wolf_is_url(const char* s) {
    if (!s) return 0;
    return (strncmp(s,"http://",7)==0||strncmp(s,"https://",8)==0||strncmp(s,"ftp://",6)==0);
}
int wolf_is_phone(const char* s) {
    if (!s) return 0;
    const char* p=s; if (*p=='+') p++;
    int digits=0;
    while (*p) { if(isdigit((unsigned char)*p)) digits++; else if(*p!=' '&&*p!='-'&&*p!='('&&*p!=')') return 0; p++; }
    return digits>=7&&digits<=15;
}
int wolf_is_uuid(const char* s) {
    if (!s||strlen(s)!=36) return 0;
    for (int i=0;i<36;i++) {
        if (i==8||i==13||i==18||i==23) { if(s[i]!='-') return 0; }
        else if (!isxdigit((unsigned char)s[i])) return 0;
    }
    return 1;
}
int wolf_is_json(const char* s) {
    if (!s||!*s) return 0;
    while (isspace((unsigned char)*s)) s++;
    return *s=='{'||*s=='[';
}
int wolf_is_ip(const char* s) {
    if (!s) return 0;
    int parts=0,val=0,has_digit=0;
    while (*s) {
        if (isdigit((unsigned char)*s)) { val=val*10+(*s-'0'); has_digit=1; if(val>255) return 0; }
        else if (*s=='.') { if(!has_digit) return 0; parts++; val=0; has_digit=0; }
        else return 0;
        s++;
    }
    return has_digit&&parts==3;
}
int wolf_is_alpha(const char* s) {
    if (!s||!*s) return 0;
    while (*s) { if(!isalpha((unsigned char)*s)) return 0; s++; }
    return 1;
}
int wolf_is_alpha_num(const char* s) {
    if (!s||!*s) return 0;
    while (*s) { if(!isalnum((unsigned char)*s)) return 0; s++; }
    return 1;
}

/* ========== Phase 3 Stdlib — File System ========== */

int wolf_file_exists(const char* path) {
    if (!path) return 0;
    FILE* f=fopen(path,"r"); if(f){fclose(f);return 1;} return 0;
}
const char* wolf_file_read(const char* path) {
    if (!path) return wolf_req_strdup("");
    FILE* f=fopen(path,"rb"); if(!f) return wolf_req_strdup("");
    fseek(f,0,SEEK_END); long size=ftell(f); fseek(f,0,SEEK_SET);
    char* buf=(char*)wolf_req_alloc(size+1);
    fread(buf,1,size,f); buf[size]='\0'; fclose(f);
    return buf;
}
int wolf_file_write(const char* path, const char* data) {
    if (!path||!data) return 0;
    FILE* f=fopen(path,"wb"); if(!f) return 0;
    fwrite(data,1,strlen(data),f); fclose(f); return 1;
}

/* wolf_file_save — decodes a base64 data blob and writes raw binary to path.
 * Usage: $ok = wolf_file_save("/uploads/avatar.jpg", $meta["data"])
 * This is the Wolf_File::Save companion to wolf_http_req_file(). */
int wolf_file_save(const char* path, const char* b64_data) {
    if (!path || !b64_data || !*b64_data) return 0;
    /* Base64 decode */
    size_t b64_len = strlen(b64_data);
    size_t out_max = (b64_len * 3) / 4 + 4;
    unsigned char* out = (unsigned char*)wolf_req_alloc(out_max);
    size_t out_len = 0;
    static const char b64_inv[256] = {
        ['+'] = 62, ['/'] = 63,
        ['0']=52,['1']=53,['2']=54,['3']=55,['4']=56,
        ['5']=57,['6']=58,['7']=59,['8']=60,['9']=61,
        ['A']=0,['B']=1,['C']=2,['D']=3,['E']=4,['F']=5,['G']=6,
        ['H']=7,['I']=8,['J']=9,['K']=10,['L']=11,['M']=12,
        ['N']=13,['O']=14,['P']=15,['Q']=16,['R']=17,['S']=18,
        ['T']=19,['U']=20,['V']=21,['W']=22,['X']=23,['Y']=24,['Z']=25,
        ['a']=26,['b']=27,['c']=28,['d']=29,['e']=30,['f']=31,['g']=32,
        ['h']=33,['i']=34,['j']=35,['k']=36,['l']=37,['m']=38,
        ['n']=39,['o']=40,['p']=41,['q']=42,['r']=43,['s']=44,
        ['t']=45,['u']=46,['v']=47,['w']=48,['x']=49,['y']=50,['z']=51,
    };
    for (size_t i = 0; i + 3 < b64_len; i += 4) {
        uint32_t n =  ((uint32_t)(unsigned char)b64_inv[(unsigned char)b64_data[i]])   << 18
                    | ((uint32_t)(unsigned char)b64_inv[(unsigned char)b64_data[i+1]]) << 12
                    | ((uint32_t)(unsigned char)b64_inv[(unsigned char)b64_data[i+2]]) << 6
                    |  (uint32_t)(unsigned char)b64_inv[(unsigned char)b64_data[i+3]];
        out[out_len++] = (n >> 16) & 0xFF;
        if (b64_data[i+2] != '=') out[out_len++] = (n >> 8) & 0xFF;
        if (b64_data[i+3] != '=') out[out_len++] =  n       & 0xFF;
    }
    /* Write raw bytes */
    FILE* f = fopen(path, "wb");
    if (!f) return 0;
    size_t written = fwrite(out, 1, out_len, f);
    fclose(f);
    return (int)(written == out_len);
}
int wolf_file_append(const char* path, const char* data) {
    if (!path||!data) return 0;
    FILE* f=fopen(path,"ab"); if(!f) return 0;
    fwrite(data,1,strlen(data),f); fclose(f); return 1;
}
int wolf_file_delete(const char* path) { return path&&remove(path)==0; }
int64_t wolf_file_size(const char* path) {
    if (!path) return -1;
    FILE* f=fopen(path,"rb"); if(!f) return -1;
    fseek(f,0,SEEK_END); long size=ftell(f); fclose(f); return (int64_t)size;
}
const char* wolf_file_extension(const char* path) {
    if (!path) return wolf_req_strdup("");
    const char* dot=strrchr(path,'.'); if(!dot||dot==path) return wolf_req_strdup("");
    return wolf_req_strdup(dot+1);
}
const char* wolf_file_basename(const char* path) {
    if (!path) return wolf_req_strdup("");
    const char* slash=strrchr(path,'/'); if(!slash) return wolf_req_strdup(path);
    return wolf_req_strdup(slash+1);
}
const char* wolf_file_dirname(const char* path) {
    if (!path) return wolf_req_strdup("");
    const char* slash=strrchr(path,'/'); if(!slash) return wolf_req_strdup(".");
    size_t len=slash-path; char* r=(char*)wolf_req_alloc(len+1); memcpy(r,path,len); r[len]='\0';
    return r;
}
int wolf_dir_exists(const char* path) {
    if (!path) return 0;
    struct stat st; return (stat(path,&st)==0&&S_ISDIR(st.st_mode));
}

/* ========== Phase 3 Stdlib — Slug & Truncate ========== */

const char* wolf_slug(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len=strlen(s); char* r=(char*)wolf_req_alloc(len+1); char* w=r;
    for (size_t i=0;i<len;i++) {
        unsigned char c=(unsigned char)s[i];
        if (isalnum(c)) *w++=(char)tolower(c);
        else if (c==' '||c=='_') { if(w>r&&*(w-1)!='-') *w++='-'; }
    }
    if (w>r&&*(w-1)=='-') w--;
    *w='\0'; return r;
}

const char* wolf_truncate(const char* s, int64_t len, const char* suffix) {
    if (!s) return wolf_req_strdup("");
    if (!suffix) suffix="...";
    size_t sl=strlen(s);
    if ((int64_t)sl<=len) return wolf_req_strdup(s);
    size_t suf_len=strlen(suffix);
    int64_t cut=len-(int64_t)suf_len; if(cut<0) cut=0;
    char* r=(char*)wolf_req_alloc(cut+suf_len+1);
    memcpy(r,s,cut); memcpy(r+cut,suffix,suf_len); r[cut+suf_len]='\0';
    return r;
}

/* ========== Sanitization Functions ========== */

const char* wolf_sanitize_string(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len=strlen(s); char* r=(char*)wolf_req_alloc(len+1); char* w=r;
    int in_tag=0;
    for (size_t i=0;i<len;i++) {
        if(s[i]=='<'){in_tag=1;continue;} if(s[i]=='>'){in_tag=0;continue;}
        if(!in_tag&&s[i]!='\''&&s[i]!='"') *w++=s[i];
    }
    *w='\0'; return r;
}
const char* wolf_sanitize_email(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len=strlen(s); char* r=(char*)wolf_req_alloc(len+1); char* w=r;
    for (size_t i=0;i<len;i++) {
        unsigned char c=(unsigned char)s[i];
        if(isalnum(c)||c=='@'||c=='.'||c=='_'||c=='-'||c=='+') *w++=s[i];
    }
    *w='\0'; return r;
}
const char* wolf_sanitize_url(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len=strlen(s); char* r=(char*)wolf_req_alloc(len+1); char* w=r;
    for (size_t i=0;i<len;i++) {
        unsigned char c=(unsigned char)s[i];
        if(isalnum(c)||c==':'||c=='/'||c=='.'||c=='-'||c=='_'||c=='~'||c=='?'||c=='#'||
           c=='['||c==']'||c=='@'||c=='!'||c=='$'||c=='&'||c=='\''||c=='('||c==')'||
           c=='*'||c=='+'||c==','||c==';'||c=='='||c=='%') *w++=s[i];
    }
    *w='\0'; return r;
}
const char* wolf_sanitize_int(const char* s) {
    if (!s) return wolf_req_strdup("0");
    size_t len=strlen(s); char* r=(char*)wolf_req_alloc(len+1); char* w=r;
    for (size_t i=0;i<len;i++)
        if(isdigit((unsigned char)s[i])||(i==0&&s[i]=='-')) *w++=s[i];
    *w='\0';
    if(w==r) return wolf_req_strdup("0");
    return r;
}
const char* wolf_sanitize_float(const char* s) {
    if (!s) return wolf_req_strdup("0");
    size_t len=strlen(s); char* r=(char*)wolf_req_alloc(len+1); char* w=r;
    int has_dot=0;
    for (size_t i=0;i<len;i++) {
        if(isdigit((unsigned char)s[i])||(i==0&&s[i]=='-')) *w++=s[i];
        else if(s[i]=='.'&&!has_dot) { *w++=s[i]; has_dot=1; }
    }
    *w='\0';
    if(w==r) return wolf_req_strdup("0");
    return r;
}


/* --- STDLIB-04: Date Object Implementation --- */

int64_t wolf_date_create(const char* str) {
    if (!str || !*str) return 0;
    
    /* Try parsing as a pure timestamp first */
    char* endptr;
    int64_t ts = strtoll(str, &endptr, 10);
    if (*endptr == '\0') return ts;

    /* Try parsing ISO-8601: YYYY-MM-DD */
    struct tm tm;
    memset(&tm, 0, sizeof(struct tm));
    if (strptime(str, "%Y-%m-%d", &tm)) {
        return (int64_t)mktime(&tm);
    }
    
    /* Try parsing ISO-8601 with time: YYYY-MM-DDTHH:MM:SS */
    memset(&tm, 0, sizeof(struct tm));
    if (strptime(str, "%Y-%m-%dT%H:%M:%S", &tm)) {
        return (int64_t)mktime(&tm);
    }

    return 0;
}

int64_t wolf_date_add_days(int64_t ts, int64_t days) {
    return ts + (days * 86400);
}

int64_t wolf_date_add_months(int64_t ts, int64_t months) {
    time_t t = (time_t)ts;
    struct tm* tm = localtime(&t);
    if (!tm) return ts;

    int total_months = tm->tm_mon + (int)months;
    tm->tm_year += total_months / 12;
    tm->tm_mon = total_months % 12;
    if (tm->tm_mon < 0) {
        tm->tm_mon += 12;
        tm->tm_year -= 1;
    }

    return (int64_t)mktime(tm);
}

int64_t wolf_date_diff_days(int64_t ts1, int64_t ts2) {
    return (ts2 - ts1) / 86400;
}

int wolf_date_is_past(int64_t ts) {
    return ts < (int64_t)time(NULL);
}

int wolf_date_is_future(int64_t ts) {
    return ts > (int64_t)time(NULL);
}

const char* wolf_date_to_iso(int64_t ts) {
    time_t t = (time_t)ts;
    struct tm* tm = gmtime(&t); /* UTC for ISO strings */
    if (!tm) return "";

    char buf[64];
    strftime(buf, sizeof(buf), "%Y-%m-%dT%H:%M:%SZ", tm);
    return wolf_req_strdup(buf);
}

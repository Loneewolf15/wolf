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
#include "wolf_numa.h"
#include "wolf_timewheel.c"   /* Phase 4: time-wheel — compiled as part of engine unit */
#include "wolf_ratelimit.c"    /* Phase 4: ratelimit token bucket */
#include <openssl/evp.h>
#include <setjmp.h>
#include <stdatomic.h>

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
#  include <sys/eventfd.h>  /* eventfd() — per-core completion wakeup */
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
    const char* force_epoll = getenv("WOLF_FORCE_EPOLL");
    if (force_epoll && force_epoll[0] == '1') {
        s->backend = 0;
    } else {
        s->backend = WOLF_IO_IOURING;
        s->uring = wolf_uring_create(4096, 1); // 4096 entries, SQPOLL enabled
        if (!s->uring) {
            fprintf(stderr, "[WOLF-ENGINE] Warning: io_uring init failed. Falling back to epoll.\n");
            s->backend = 0;
        } else {
            s->poll_fd = -1;
        }
    }
#endif /* WOLF_HAS_IO_URING */

#if defined(WOLF_HAS_EPOLL)
    if (s->backend == 0) {
        s->backend = WOLF_IO_EPOLL;
        s->poll_fd = epoll_create1(EPOLL_CLOEXEC);
        if (s->poll_fd < 0) { perror("epoll_create1"); free(s); return NULL; }
    }
#endif

#if defined(WOLF_HAS_KQUEUE)
    if (s->backend == 0) {
        s->backend = WOLF_IO_KQUEUE;
        s->poll_fd = kqueue();
        if (s->poll_fd < 0) { perror("kqueue"); free(s); return NULL; }
    }
#endif

    if (s->backend == 0) {
        s->backend = WOLF_IO_POLL;
        s->poll_fd = -1;
    }
    
    return s;
}

/* Context storage for callbacks — maps fd → callback+ctx */
#define WOLF_SENTINEL_MAX_FDS 4096
#define WOLF_MAX_FD 65536

typedef struct {
    int                fd;
    wolf_io_callback_t cb;
    void*              ctx;
} WolfFDEntry;

static __thread WolfFDEntry wolf_fd_table[WOLF_MAX_FD];

static WolfFDEntry* wolf_fd_find(int fd) {
    if (fd < 0 || fd >= WOLF_MAX_FD || (!wolf_fd_table[fd].cb && !wolf_fd_table[fd].ctx)) return NULL;
    return &wolf_fd_table[fd];
}

static WolfFDEntry* wolf_fd_alloc(int fd, wolf_io_callback_t cb, void* ctx) {
    if (fd < 0 || fd >= WOLF_MAX_FD) return NULL;
    WolfFDEntry* e = &wolf_fd_table[fd];
    e->fd = fd; e->cb = cb; e->ctx = ctx;
    return e;
}

static void wolf_fd_remove_entry(int fd) {
    if (fd >= 0 && fd < WOLF_MAX_FD) {
        wolf_fd_table[fd].cb = NULL;
        wolf_fd_table[fd].ctx = NULL;
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

#ifndef WOLF_MAX_PARSER_FIELDS
#define WOLF_MAX_PARSER_FIELDS 69 /* body, method, path, query, ws_key, 64 headers */
#endif

/* 
 * Encode the mathematical invariant for arena safety:
 * The maximum memory requested by the parser is the raw request size plus 
 * 7 bytes of worst-case 8-byte alignment padding per parsed field.
 * The overflow slots (64 max) must be able to absorb any allocations that exceed the slab.
 * Since every overflow allocation consumes at least 8 bytes, the max possible 
 * overflow count is (MaxRequested - SlabSize) / 8.
 */
#if defined(__STDC_VERSION__) && __STDC_VERSION__ >= 201112L
_Static_assert(
    (WOLF_MAX_REQUEST_SIZE + (WOLF_MAX_PARSER_FIELDS * 7) - WOLF_ARENA_SLAB_SIZE) / 8 <= 64,
    "WOLF_MAX_REQUEST_SIZE is too large for the 64 overflow slots. Increase overflow array size or slab capacity."
);
#endif

WolfArenaPool* wolf_arena_pool_create(int core_id) {
    WolfArenaPool* pool = (WolfArenaPool*)calloc(1, sizeof(WolfArenaPool));
    pool->core_id = core_id;
    pool->count   = WOLF_ARENA_POOL_SIZE;

    int node = wolf_numa_node_of_cpu(core_id);

    for (int i = 0; i < WOLF_ARENA_POOL_SIZE; i++) {
        pool->arenas[i].base_slab     = (char*)wolf_numa_alloc_onnode(WOLF_ARENA_SLAB_SIZE, node);
        pool->arenas[i].base_cap      = WOLF_ARENA_SLAB_SIZE;
        pool->arenas[i].active_slab   = pool->arenas[i].base_slab;
        pool->arenas[i].cap           = WOLF_ARENA_SLAB_SIZE;
        pool->arenas[i].pos           = 0;
        pool->arenas[i].total_allocated = 0;
        pool->arenas[i].fallback_blocks = NULL;
        /* Fix #6: RELEASE store so worker threads see in_use=0 on ARM64 */
        __atomic_store_n(&pool->arenas[i].in_use, 0, __ATOMIC_RELEASE);
        if (!pool->arenas[i].base_slab) {
            fprintf(stderr, "[WOLF-ENGINE] OOM allocating arena pool for core %d\n", core_id);
            /* Continue with partial allocation */
        }
    }
    return pool;
}

WolfArena* wolf_arena_acquire(WolfArenaPool* pool) {
    for (int i = 0; i < pool->count; i++) {
        /* Fix #6: ACQUIRE load for memory-ordering safety on ARM64 */
        if (!__atomic_load_n(&pool->arenas[i].in_use, __ATOMIC_ACQUIRE)
            && pool->arenas[i].base_slab) {
            __atomic_store_n(&pool->arenas[i].in_use, 1, __ATOMIC_RELEASE);
            pool->arenas[i].active_slab = pool->arenas[i].base_slab;
            pool->arenas[i].cap         = pool->arenas[i].base_cap;
            pool->arenas[i].pos         = 0;
            pool->arenas[i].total_allocated = 0;
            pool->arenas[i].fallback_blocks = NULL;
            pool->arenas[i].is_overflow = 0;
            pool->arenas[i].refcount    = 1;
            return &pool->arenas[i];
        }
    }
    /* All arenas busy — return NULL to enforce hard backpressure and prevent OOM under load */
    fprintf(stderr, "[WOLF-ENGINE] WARN: arena pool exhausted on core — dropping connection for backpressure\n");
    return NULL;
}

/* Header hash table helpers — guarded so they are not redefined when
 * this file is unity-built into wolf_runtime.c (which defines them first). */
#ifndef WOLF_HEADER_HASH_SLOTS
#define WOLF_HEADER_HASH_SLOTS 64
static inline uint32_t wolf_header_hash(const char* key) {
    uint32_t h = 2166136261u;
    for (const unsigned char* p = (const unsigned char*)key; *p; p++) {
        h ^= (uint8_t)(*p | 0x20);
        h *= 16777619u;
    }
    return h;
}
static inline void wolf_header_htab_insert(int8_t* htab, int idx, const char* key) {
    uint32_t h = wolf_header_hash(key);
    for (int i = 0; i < WOLF_HEADER_HASH_SLOTS; i++) {
        int slot = (int)((h + (uint32_t)i) & (WOLF_HEADER_HASH_SLOTS - 1));
        if (htab[slot] < 0) { htab[slot] = (int8_t)idx; return; }
    }
}
static inline const char* wolf_header_htab_get(const int8_t* htab,
                                                char* const* keys,
                                                char* const* vals,
                                                const char* key) {
    uint32_t h = wolf_header_hash(key);
    for (int i = 0; i < WOLF_HEADER_HASH_SLOTS; i++) {
        int slot = (int)((h + (uint32_t)i) & (WOLF_HEADER_HASH_SLOTS - 1));
        int8_t idx = htab[slot];
        if (idx < 0) return "";
        if (strcasecmp(keys[(int)idx], key) == 0) return vals[(int)idx];
    }
    return "";
}
#endif /* WOLF_HEADER_HASH_SLOTS */

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
    int8_t  header_htab[WOLF_HEADER_HASH_SLOTS]; /* O(1) lookup: -1=empty, else index */

    /* Response */
    int     status_code;
    char*   res_header_keys[32];
    char*   res_header_vals[32];
    int     res_header_count;
    char*   res_body;

    /* WebSocket */
    int     is_websocket;
    char*   ws_key;

    /* Uploads / Client Info */
    char    client_ip[46];
    int     upload_count;
    wolf_upload_t uploads[WOLF_MAX_UPLOADS];

    /* Arena for this request */
    WolfArena* arena;

    /* io_uring async state */
    char* read_buf;
    ssize_t bytes_in;
    struct WolfCore* core;

    /* Timing */
    struct timespec started_at;
    jmp_buf oom_jump;
    int oom_triggered;
    int64_t request_id;
    int64_t arena_used;
    int64_t arena_cap;

    /* Keep-alive request counter — close connection after WOLF_KEEPALIVE_MAX_REQUESTS */
    int keep_alive_count;
    int worker_id;

    /* Phase 5: smuggling defense flag — set by validate_crlf_and_smuggling (R2)
     * when Transfer-Encoding: chunked is present. When true the body reader must
     * use chunked framing and MUST NOT use Content-Length for body sizing. */
    int ignore_content_length;
} WolfConnCtx;

extern __thread void* wolf_active_ctx;

void* wolf_arena_alloc(WolfArena* arena, size_t size) {
    /* Align to 16 bytes per spec */
    size = (size + 15) & ~(size_t)15;
    
    if (arena->pos + size > arena->cap) {
        /* Single allocation exceeds current active slab — geometric growth fallback */
        size_t new_cap = arena->cap * 2;
        if (size > new_cap) {
            new_cap = size;
        }

        if (arena->total_allocated + new_cap > WOLF_MAX_REQUEST_MEMORY) {
            wolf_panic_oom();
            fprintf(stderr, "[WOLF-ARENA] Hard ceiling reached: request exceeded 16MB. Rejecting.\n");
            return NULL;
        }

        WolfArenaBlock* block = (WolfArenaBlock*)malloc(sizeof(WolfArenaBlock));
        if (!block) wolf_panic_oom();
        
        block->slab = (char*)calloc(1, new_cap);
        if (!block->slab) {
            wolf_panic_oom();
            free(block);
            return NULL;
        }

        /* Head insertion into fallback_blocks */
        block->next = arena->fallback_blocks;
        arena->fallback_blocks = block;

        /* Update arena state */
        arena->active_slab = block->slab;
        arena->cap = new_cap;
        arena->total_allocated += new_cap;
        
        void* ptr = arena->active_slab; /* return address FIRST */
        arena->pos = size;              /* advance SECOND */

        WolfConnCtx* c = __atomic_load_n(&wolf_active_ctx, __ATOMIC_ACQUIRE);
        if (c) c->arena_used = arena->total_allocated + arena->pos;

        return ptr;
    }
    void* p = arena->active_slab + arena->pos;
    arena->pos += size;
    
    WolfConnCtx* c = __atomic_load_n(&wolf_active_ctx, __ATOMIC_ACQUIRE);
    if (c) c->arena_used = arena->total_allocated + arena->pos;

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

void wolf_arena_ref(WolfArena* arena) {
    if (!arena) return;
    __atomic_fetch_add(&arena->refcount, 1, __ATOMIC_SEQ_CST);
}

void wolf_arena_unref(WolfArena* arena) {
    if (!arena) return;
    if (__atomic_sub_fetch(&arena->refcount, 1, __ATOMIC_SEQ_CST) == 0) {
        wolf_arena_reset(arena);
        if (arena->is_overflow) {
            free(arena->base_slab);
            free(arena);
        } else {
            arena->in_use = 0;
        }
    }
}

void wolf_arena_reset(WolfArena* arena) {
    if (!arena) return;

    /* Free all geometric fallback blocks */
    WolfArenaBlock* curr = arena->fallback_blocks;
    while (curr) {
        WolfArenaBlock* next = curr->next;
        free(curr->slab);
        free(curr);
        curr = next;
    }
    arena->fallback_blocks = NULL;

    /* Restore to base state for O(1) reuse */
    arena->active_slab = arena->base_slab;
    arena->cap         = arena->base_cap;
    arena->pos         = 0;  /* O(1) — just reset the pointer */
    arena->total_allocated = 0;
}

void wolf_arena_pool_destroy(WolfArenaPool* pool) {
    if (!pool) return;
    for (int i = 0; i < pool->count; i++) {
        /* Free fallback blocks if any are still lingering */
        WolfArenaBlock* curr = pool->arenas[i].fallback_blocks;
        while (curr) {
            WolfArenaBlock* next = curr->next;
            free(curr->slab);
            free(curr);
            curr = next;
        }
        if (pool->arenas[i].base_slab) {
            wolf_numa_free(pool->arenas[i].base_slab, pool->arenas[i].base_cap);
        }
    }
    free(pool);
}

/* ================================================================
 * HTTP Connection State — per-core, no mutex needed
 * ================================================================ */



#define WOLF_CORE_CTX_MAX 128

/* ================================================================
 * HTTP Request Parser (arena-backed, zero-copy where possible)
 * ================================================================ */

/* wolf_engine_parse_multipart — parse multipart/form-data body into
 * WolfConnCtx.uploads[] using the per-request arena.
 * Uses memmem() for POSIX-portable boundary search.
 * Mirrors wolf_parse_multipart() in wolf_runtime.c but targets WolfConnCtx
 * (arena alloc) instead of http_contexts[] (wolf_req_alloc arena). */
void wolf_engine_parse_multipart(WolfConnCtx* ctx,
                                         const char* ct_header,
                                         const char* body, size_t body_len) {
    WolfArena* a = ctx->arena;
    if (!a || !body || body_len == 0) return;

    /* Extract boundary string from "multipart/form-data; boundary=XXX" */
    const char* bp = strstr(ct_header, "boundary=");
    if (!bp) return;
    bp += 9;
    /* Strip optional surrounding quotes */
    char boundary[256];
    size_t bi = 0;
    while (*bp && *bp != ';' && *bp != '\r' && *bp != '\n' && bi < 254) {
        if (*bp != '"') { boundary[bi++] = *bp; }
        bp++;
    }
    boundary[bi] = '\0';
    if (bi == 0) return;

    /* Full boundary delimiter: "--" + boundary */
    char delim[260];
    snprintf(delim, sizeof(delim), "--%s", boundary);
    size_t delim_len = strlen(delim);

    const char* p   = body;
    const char* end = body + body_len;

    while (p < end && ctx->upload_count < WOLF_MAX_UPLOADS) {
        /* Find next boundary using memmem (POSIX) */
        const char* part_start = (const char*)memmem(p, (size_t)(end - p), delim, delim_len);
        if (!part_start) break;
        p = part_start + delim_len;

        /* End-of-multipart marker: "--" immediately after boundary */
        if (p + 2 <= end && p[0] == '-' && p[1] == '-') break;
        /* Skip CRLF after boundary line */
        if (p + 2 <= end && p[0] == '\r' && p[1] == '\n') p += 2;

        /* Parse part headers until blank line */
        const char* field_name   = NULL;
        const char* filename     = NULL;
        const char* part_ct      = "application/octet-stream";
        const char* part_hdr_end = NULL;

        const char* hp = p;
        while (hp < end) {
            /* Find CRLF end of header line */
            const char* eol = NULL;
            for (const char* s = hp; s + 1 < end; s++) {
                if (s[0] == '\r' && s[1] == '\n') { eol = s; break; }
            }
            if (!eol) break;
            if (eol == hp) { /* Blank line = end of part headers */
                part_hdr_end = eol + 2;
                break;
            }

            /* Copy header line into arena for in-place parsing */
            size_t hlen = (size_t)(eol - hp);
            char* hline = (char*)wolf_arena_alloc(a, hlen + 1);
            if (!hline) break;
            memcpy(hline, hp, hlen);
            hline[hlen] = '\0';

            /* Parse Content-Disposition: form-data; name="..."; filename="..." */
            if (strncasecmp(hline, "Content-Disposition:", 20) == 0) {
                char* np = strstr(hline, "name=");
                if (np) {
                    np += 5;
                    int quoted = (*np == '"');
                    if (quoted) np++;
                    char* ne = np;
                    while (*ne && (quoted ? *ne != '"' : (*ne != ';' && *ne != '\r'))) ne++;
                    size_t nl = (size_t)(ne - np);
                    char* nbuf = (char*)wolf_arena_alloc(a, nl + 1);
                    if (nbuf) { memcpy(nbuf, np, nl); nbuf[nl] = '\0'; field_name = nbuf; }
                }
                char* fp = strstr(hline, "filename=");
                if (fp) {
                    fp += 9;
                    int quoted = (*fp == '"');
                    if (quoted) fp++;
                    char* fe = fp;
                    while (*fe && (quoted ? *fe != '"' : (*fe != ';' && *fe != '\r'))) fe++;
                    size_t fl = (size_t)(fe - fp);
                    char* fbuf = (char*)wolf_arena_alloc(a, fl + 1);
                    if (fbuf) { memcpy(fbuf, fp, fl); fbuf[fl] = '\0'; filename = fbuf; }
                }
            }

            /* Parse Content-Type of this part */
            if (strncasecmp(hline, "Content-Type:", 13) == 0) {
                char* ctv = hline + 13;
                while (*ctv == ' ') ctv++;
                part_ct = wolf_arena_strdup(a, ctv);
            }

            hp = eol + 2;
        }

        /* Skip parts without both field name and filename (not file uploads) */
        if (!part_hdr_end || !field_name || !filename) {
            p = part_hdr_end ? part_hdr_end : hp;
            continue;
        }

        /* Part body: from part_hdr_end until next delimiter (preceded by CRLF) */
        const char* data_start = part_hdr_end;
        const char* data_end   = end;
        for (const char* s = data_start; s + delim_len + 2 <= end; s++) {
            if (s[0] == '\r' && s[1] == '\n' && memcmp(s + 2, delim, delim_len) == 0) {
                data_end = s;
                break;
            }
        }

        size_t data_size = (size_t)(data_end - data_start);
        char*  data_buf  = (char*)wolf_arena_alloc(a, data_size + 1);
        if (!data_buf) break;
        memcpy(data_buf, data_start, data_size);
        data_buf[data_size] = '\0';

        /* Sanitize filename (basename only — prevent path traversal) */
        const char* safe_name = wolf_file_basename(filename);

        wolf_upload_t* up = &ctx->uploads[ctx->upload_count++];
        up->field_name   = field_name;
        up->filename     = wolf_arena_strdup(a, safe_name);
        up->content_type = part_ct;
        up->data         = data_buf;
        up->size         = data_size;

        p = data_end;
    }
}

static int wolf_engine_parse_request(WolfConnCtx* ctx, char* raw, size_t len) {
    return wolf_engine_parse_request_simd(ctx, raw, len);
}


#if 0
static void _unused_wolf_engine_parse_request(WolfConnCtx* ctx, char* raw, size_t len) {
    WolfArena* a = ctx->arena;

    /* Find header/body boundary */
    char* body_start = NULL;
    size_t body_len  = 0;
    for (size_t i = 0; i + 3 < len; i++) {
        if (raw[i]=='\r' && raw[i+1]=='\n' && raw[i+2]=='\r' && raw[i+3]=='\n') {
            raw[i] = '\0';
            body_start = raw + i + 4;
            body_len   = len - (i + 4);
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
    const char* upgrade_val      = NULL;
    const char* ws_key_val       = NULL;
    const char* content_type_val = NULL;

    while ((line = strtok_r(NULL, "\r\n", &saveptr))) {
        char* colon = strchr(line, ':');
        if (colon && ctx->header_count < 32) {
            *colon = '\0';
            char* val = colon + 1;
            while (*val == ' ') val++;
            ctx->header_keys[ctx->header_count] = wolf_arena_strdup(a, line);
            ctx->header_vals[ctx->header_count] = wolf_arena_strdup(a, val);
            if (strcasecmp(line, "Upgrade") == 0)          upgrade_val      = ctx->header_vals[ctx->header_count];
            if (strcasecmp(line, "Sec-WebSocket-Key") == 0) ws_key_val       = ctx->header_vals[ctx->header_count];
            if (strcasecmp(line, "Content-Type") == 0)      content_type_val = ctx->header_vals[ctx->header_count];
            wolf_header_htab_insert(ctx->header_htab, ctx->header_count, line);
            ctx->header_count++;
        }
    }

    if (upgrade_val && strcasecmp(upgrade_val, "websocket") == 0 && ws_key_val) {
        ctx->is_websocket = 1;
        ctx->ws_key = wolf_arena_strdup(a, ws_key_val);
    }

    /* Parse multipart/form-data uploads if present */
    if (body_start && body_len > 0 && content_type_val &&
        strstr(content_type_val, "multipart/form-data")) {
        wolf_engine_parse_multipart(ctx, content_type_val, body_start, body_len);
    }
}
#endif

/* ================================================================
 * HTTP Response Writer
 * ================================================================ */

static int wolf_engine_build_response(WolfConnCtx* ctx, char** out_buf, int* out_len) {
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
    
    // Estimate total size to allocate from arena.
    // 128 = HTTP/1.1 status line + Content-Length + Connection headers baseline.
    // 256 = headroom for the 4 automatic security headers injected per response.
    int total_size = 128 + 256;
    for (int i = 0; i < ctx->res_header_count; i++) {
        total_size += strlen(ctx->res_header_keys[i]) + strlen(ctx->res_header_vals[i]) + 4;
    }
    total_size += body_len;

    char* response = (char*)wolf_arena_alloc(ctx->arena, total_size + 64); // +64 safety margin
    if (!response) return -1;
    
    char* ptr = response;
    
    // Build status line
    ptr += snprintf(ptr, 128, "HTTP/1.1 %d %s\r\n", ctx->status_code, status_text);
    
    // Build headers
    for (int i = 0; i < ctx->res_header_count; i++) {
        int remaining = (response + total_size + 64) - ptr;
        int written = snprintf(ptr, remaining, "%s: %s\r\n", ctx->res_header_keys[i], ctx->res_header_vals[i]);
        if (written < 0 || written >= remaining) {
            fprintf(stderr, "[DEBUG] Header loop overflow: written=%d remaining=%d\n", written, remaining);
            return -1; // Overflow or encoding error
        }
        ptr += written;
    }

    // Build fixed headers & empty line separator.
    const char* security_headers[][2] = {
        {"X-Content-Type-Options",  "nosniff"},
        {"X-Frame-Options",         "DENY"},
        {"X-XSS-Protection",        "1; mode=block"},
        {"Referrer-Policy",         "strict-origin-when-cross-origin"},
    };
    int n_sec = (int)(sizeof(security_headers) / sizeof(security_headers[0]));
    for (int s = 0; s < n_sec; s++) {
        /* Only inject if the user hasn't already set this header */
        int already_set = 0;
        for (int h = 0; h < ctx->res_header_count; h++) {
            if (ctx->res_header_keys[h] && strcasecmp(ctx->res_header_keys[h], security_headers[s][0]) == 0) {
                already_set = 1;
                break;
            }
        }
        if (!already_set) {
            int rem2 = (response + total_size + 64) - ptr;
            int w2   = snprintf(ptr, rem2, "%s: %s\r\n",
                                security_headers[s][0], security_headers[s][1]);
            if (w2 > 0 && w2 < rem2) ptr += w2;
        }
    }

    int remaining = (response + total_size + 64) - ptr;
    int written = snprintf(ptr, remaining, "Content-Length: %d\r\nConnection: keep-alive\r\n\r\n", body_len);
    if (written < 0 || written >= remaining) {
        fprintf(stderr, "[DEBUG] Fixed headers overflow: written=%d remaining=%d\n", written, remaining);
        return -1;
    }
    ptr += written;
    
    // Build body
    if (body_len > 0) {
        memcpy(ptr, ctx->res_body, body_len);
        ptr += body_len;
    }
    
    int final_len = ptr - response;
    
    if (out_buf) *out_buf = response;
    if (out_len) *out_len = final_len;
    
    return 0;
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

static int wolf_engine_shutdown_flag = 0;
WolfEngine* g_wolf_engine = NULL;

static void wolf_engine_signal_handler(int sig) {
    (void)sig;
    __atomic_store_n(&wolf_engine_shutdown_flag, 1, __ATOMIC_RELEASE);
}

typedef struct {
    WolfCore*           core;
    void*               http_handler;
    wolf_ws_handler_t   ws_handler;
} WolfCoreArgs;

/* Inline context table — per-core, no locking */
__thread WolfConnCtx wolf_core_ctxs[WOLF_CORE_CTX_MAX];

/* Watchdog pointer for signal-safe timeout enforcement */
__thread void* wolf_active_ctx = NULL;

/* Free-list index stack for O(1) alloc/free of context slots (Fix 3) */
__thread int wolf_ctx_free_stack[WOLF_CORE_CTX_MAX];
__thread int wolf_ctx_free_top = -1;   /* -1 = uninitialized */

static void wolf_ctx_freelist_init(void) {
    wolf_ctx_free_top = -1;
    for (int i = WOLF_CORE_CTX_MAX - 1; i >= 0; i--)
        wolf_ctx_free_stack[++wolf_ctx_free_top] = i;
}

static WolfConnCtx* wolf_core_alloc_ctx(WolfCore* core, int client_fd, WolfArena* arena) {
    if (wolf_ctx_free_top < 0) return NULL;   /* all slots in use */

    int idx = wolf_ctx_free_stack[wolf_ctx_free_top--];
    WolfConnCtx* ctx = &wolf_core_ctxs[idx];

    memset(ctx, 0, sizeof(WolfConnCtx));
    memset(ctx->header_htab, -1, sizeof(ctx->header_htab));
    ctx->active      = 1;
    ctx->client_fd   = client_fd;
    ctx->core_id     = core->core_id;
    ctx->status_code = 200;
    ctx->arena       = arena;
    ctx->arena_cap   = WOLF_MAX_REQUEST_MEMORY;
    clock_gettime(CLOCK_MONOTONIC, &ctx->started_at);
    return ctx;
}

static void wolf_core_free_ctx(WolfConnCtx* ctx) {
    if (!ctx) return;
    if (ctx->active == 0) {
        fprintf(stderr, "[WOLF-ENGINE] DOUBLE FREE DETECTED for ctx_id: %d\n", (int)(ctx - wolf_core_ctxs));
        return;
    }
    if (ctx->arena) {
        wolf_arena_unref(ctx->arena);  /* W1 Fix: decrements ref, frees if 0 */
        ctx->arena = NULL;             /* prevent dangling pointer on overflow arenas */
    }
    if (ctx->read_buf) {
        free(ctx->read_buf);
        ctx->read_buf = NULL;
    }
    ctx->active = 0;

    /* Return slot to free-list — O(1) */
    int idx = (int)(ctx - wolf_core_ctxs);
    wolf_ctx_free_stack[++wolf_ctx_free_top] = idx;
}

/* ================================================================
 * WebSocket bridge — transfers an upgraded fd from the new engine
 * into the legacy http_contexts[] table that the WS poller owns.
 * Declared extern here; defined in wolf_runtime.c.
 * ================================================================ */
extern int wolf_engine_register_ws_fd(int fd, const char* method, const char* path, 
                                      const char* query, const char* ws_key, 
                                      const char* client_ip);



/* ================================================================
 * Worker → Poller completion ring support
 *
 * wolf_engine_handle_offloaded — called by a worker thread.
 *   Executes the Wolf handler, builds the HTTP response, then posts
 *   the result to ctx->core->complete_ring and writes the eventfd to
 *   wake the owning poller thread.
 *
 * wolf_core_drain_completions — called by the poller thread.
 *   Drains all pending entries from the core's completion ring and
 *   submits the corresponding io_uring sends (or direct writes in the
 *   epoll fallback path).
 *
 * on_eventfd_ready — io_uring callback fired when notify_fd is readable.
 *   Drains completions and re-arms the multishot poll so future writes
 *   to the eventfd continue to generate CQEs.
 * ================================================================ */

static void on_send_complete(int client_fd, void* ctx_ptr, int bytes_written); /* fwd */

static int wolf_engine_task_push(WolfCore* core, wolf_task_t task) {
    if (core->spsc_worker_count > 0) {
        wolf_spsc_entry_t entry = {
            .type = task.type,
            .id = task.id,
            .ctx = task.payload,
            .engine_fn = (void*)task.engine_fn,
        };
        /* Use round-robin counter local to the core to distribute tasks */
        int worker_id = (core->spsc_next_worker++) % core->spsc_worker_count;
        return wolf_spsc_push(&core->spsc_submit_rings[worker_id], &entry);
    }
    return 0;
}

static void wolf_engine_handle_offloaded(void* ctx_ptr, int worker_id) {
    WolfConnCtx*  ctx  = (WolfConnCtx*)ctx_ptr;
    ctx->worker_id = worker_id;
    WolfCore*     core = ctx->core;
    WolfCoreArgs* args = (WolfCoreArgs*)core->args;

    if (args && args->http_handler) {
        int64_t ctx_id = ctx->request_id;
        extern void wolf_set_current_context(void*, void*);
        wolf_set_current_context((void*)(intptr_t)ctx_id, (void*)(intptr_t)ctx_id);
        __atomic_store_n(&wolf_active_ctx, ctx, __ATOMIC_RELEASE);

        extern void wolf_req_arena_init(void);
        wolf_req_arena_init();

        /* ── Wolf Core Guarantee #6: Auto-intercept GET /health ──────────────────
         * Every Wolf binary responds to GET /health with a JSON status document.
         * This is injected at the engine level — the user's handler never sees it.
         * Required for Kubernetes liveness/readiness probes and Docker HEALTHCHECK.
         * ---------------------------------------------------------------------- */
        if (ctx->method && ctx->path &&
            strcmp(ctx->method, "GET") == 0 &&
            strcmp(ctx->path, "/health") == 0) {
            ctx->status_code = 200;
            ctx->res_body = wolf_arena_alloc(ctx->arena, 64);
            if (ctx->res_body) {
                snprintf((char*)ctx->res_body, 64, "{\"status\":\"ok\"}");
            } else {
                ctx->res_body = "{\"status\":\"ok\"}";
            }
            /* Set Content-Type for health response */
            if (ctx->res_header_count < 32) {
                ctx->res_header_keys[ctx->res_header_count] = "Content-Type";
                ctx->res_header_vals[ctx->res_header_count] = "application/json";
                ctx->res_header_count++;
            }
            __atomic_store_n(&wolf_active_ctx, NULL, __ATOMIC_RELEASE);
            goto build_and_send;
        }

        wolf_closure_t* closure = (wolf_closure_t*)args->http_handler;
        typedef void* (*wolf_closure_fn_t)(void* env, int64_t req_id, int64_t res_id);
        wolf_closure_fn_t fn = (wolf_closure_fn_t)closure->fn;

        ctx->oom_triggered = 0;
        if (!wolf_closure_valid(closure)) {
            ctx->status_code = 500;
        } else if (setjmp(ctx->oom_jump) == 0) {
            fn(closure->env, ctx_id, ctx_id);
        } else {
            ctx->status_code = 500;
            ctx->res_body = "500 Internal Server Error (OOM)";
        }

        __atomic_store_n(&wolf_active_ctx, NULL, __ATOMIC_RELEASE);

        extern int  wolf_req_oom_check(void);
        extern void wolf_req_oom_clear(void);
        if (wolf_req_oom_check()) {
            ctx->status_code = 503;
            if (!ctx->res_body) ctx->res_body = "Service Unavailable";
            wolf_req_oom_clear();
        }
    } else {
        ctx->status_code = 500;
    }

build_and_send:
    ;

    char* out_buf = NULL;
    int   out_len = 0;
    wolf_engine_build_response(ctx, &out_buf, &out_len);

    extern void wolf_req_arena_flush(void);
    wolf_req_arena_flush();

    /* Post to the core's completion ring --------------------------------- */
    if (ctx->worker_id >= 0) {
        wolf_spsc_entry_t entry = {
            .type = 4, /* COMPLETION */
            .ctx = ctx,
            .out_buf = out_buf,
            .out_len = out_len,
            .worker_id = ctx->worker_id
        };
        wolf_spsc_push(&core->spsc_complete_rings[ctx->worker_id], &entry);
        
        /* Wake the poller via eventfd */
        if (core->notify_fd >= 0) {
            uint64_t v = 1;
            ssize_t n = write(core->notify_fd, &v, sizeof(v));
            (void)n;
        }
    }
}

/* Drain all ready completions from the ring and submit sends.
 * Must be called from the owning poller thread. */
static void wtask_complete_cb(const wolf_spsc_entry_t* entry, void* userdata) {
    WolfCore* core = (WolfCore*)userdata;
    WolfConnCtx* ctx = (WolfConnCtx*)entry->ctx;
    char* out_buf = entry->out_buf;
    int out_len = entry->out_len;

#if defined(WOLF_HAS_IO_URING)
    if (core->sentinel->backend == WOLF_IO_IOURING) {
        if (out_buf && out_len > 0) {
            if (core->use_send_zc) {
                wolf_uring_submit_send_zc(core->sentinel->uring, ctx->client_fd,
                                          out_buf, out_len,
                                          on_send_complete, ctx, ctx->arena);
            } else {
                wolf_uring_submit_send(core->sentinel->uring, ctx->client_fd,
                                       out_buf, out_len,
                                       on_send_complete, ctx, ctx->arena);
            }
        } else {
            on_send_complete(ctx->client_fd, ctx, -1);
        }
        wolf_uring_flush(core->sentinel->uring);
    } else {
#endif
    /* epoll / kqueue / poll fallback send path */
    if (out_buf && out_len > 0) {
        ssize_t n = send(ctx->client_fd, out_buf, out_len, MSG_DONTWAIT);
        if (n > 0) {
            __atomic_fetch_add(&core->requests_total, 1, __ATOMIC_RELAXED);
            __atomic_fetch_add(&core->bytes_in,  ctx->bytes_in, __ATOMIC_RELAXED);
            __atomic_fetch_add(&core->bytes_out, n, __ATOMIC_RELAXED);
        }
    }
    __atomic_fetch_sub(&core->requests_active, 1, __ATOMIC_RELAXED);

#if defined(WOLF_HAS_EPOLL)
    if (core->epoll_fd >= 0)
        epoll_ctl(core->epoll_fd, EPOLL_CTL_DEL, ctx->client_fd, NULL);
#elif defined(WOLF_HAS_KQUEUE)
    if (core->sentinel && core->sentinel->poll_fd >= 0) {
        struct kevent ev;
        EV_SET(&ev, ctx->client_fd, EVFILT_READ, EV_DELETE, 0, 0, NULL);
        kevent(core->sentinel->poll_fd, &ev, 1, NULL, 0, NULL);
    }
#endif
    if (ctx->client_fd >= 0) {
        wolf_fd_remove_entry(ctx->client_fd);
        if (core->timewheel) {
            wolf_timewheel_remove(core->timewheel, ctx->client_fd);
        }
        close(ctx->client_fd);
    }
    wolf_core_free_ctx(ctx);
#if defined(WOLF_HAS_IO_URING)
    }
#endif
}

void wolf_core_drain_completions(WolfCore* core) {
    if (core->spsc_worker_count > 0) {
        for (int i = 0; i < core->spsc_worker_count; i++) {
            wolf_spsc_entry_t entry;
            int drain_count = 0;
            while (drain_count < 1024 && wolf_spsc_pop(&core->spsc_complete_rings[i], &entry)) {
                wtask_complete_cb(&entry, core);
                drain_count++;
            }
        }
    }
    
    if (core->notify_fd >= 0) {
        uint64_t v;
        ssize_t n = read(core->notify_fd, &v, sizeof(v));
        (void)n;
    }
}

#if defined(WOLF_HAS_IO_URING)
/* io_uring callback: fired when notify_fd is readable (worker posted a result).
 * Drain completions and re-arm the persistent poll so future writes continue
 * to generate CQEs without requiring a re-submit from the application. */
static void on_eventfd_ready(int fd, void* core_ptr, int res) {
    (void)res;
    WolfCore* core = (WolfCore*)core_ptr;
    wolf_core_drain_completions(core);
    /* Re-arm: IORING_POLL_ADD_MULTI keeps the poll alive if the kernel supports it.
     * For older kernels that fire once, re-submit here. */
    wolf_uring_poll_fd(core->sentinel->uring, fd, on_eventfd_ready, core);
    wolf_uring_flush(core->sentinel->uring);
}
#endif

#if defined(WOLF_HAS_IO_URING)
static void on_recv_complete(int client_fd, void* ctx_ptr, int bytes_read);

static void on_send_complete(int client_fd, void* ctx_ptr, int bytes_written) {
    WolfConnCtx* ctx = (WolfConnCtx*)ctx_ptr;
    WolfCore* core = ctx->core;

    /* Every send completion (success or failure) ends one request cycle. */
    __atomic_fetch_sub(&core->requests_active, 1, __ATOMIC_RELAXED);

    if (bytes_written >= 0) {
        __atomic_fetch_add(&core->requests_total, 1, __ATOMIC_RELAXED);
        __atomic_fetch_add(&core->bytes_in,  ctx->bytes_in,                              __ATOMIC_RELAXED);
        __atomic_fetch_add(&core->bytes_out, ctx->res_body ? strlen(ctx->res_body) : 0, __ATOMIC_RELAXED);
    }

    if (bytes_written > 0) {
        ctx->keep_alive_count++;

        if (ctx->keep_alive_count >= WOLF_KEEPALIVE_MAX_REQUESTS) {
            /* Hard limit reached: close gracefully to prevent connection hogging. */
            close(client_fd);
            wolf_core_free_ctx(ctx);
            return;
        }

        /* Keep-alive: reset arena and connection state, wait for next request. */
        wolf_arena_reset(ctx->arena);

        ctx->method           = NULL;
        ctx->path             = NULL;
        ctx->query            = NULL;
        ctx->body             = NULL;
        ctx->header_count     = 0;
        memset(ctx->header_htab, -1, sizeof(ctx->header_htab));
        ctx->status_code      = 200;
        ctx->res_body         = NULL;
        ctx->res_header_count = 0;
        ctx->is_websocket     = 0;
        ctx->ws_key           = NULL;
        ctx->upload_count     = 0;
        ctx->bytes_in         = 0;
        /* read_buf is malloc'd once at accept; reuse it for the lifetime of the connection. */

        wolf_uring_submit_recv(core->sentinel->uring, client_fd, ctx->read_buf,
                               WOLF_MAX_REQUEST_SIZE - 1, on_recv_complete, ctx, ctx->arena);
    } else {
        /* Connection closed or send error: tear down. */
        close(client_fd);
        wolf_core_free_ctx(ctx);
    }
}

static void on_recv_complete(int client_fd, void* ctx_ptr, int bytes_read) {
    WolfConnCtx* ctx = (WolfConnCtx*)ctx_ptr;
    WolfCore* core = ctx->core;
    WolfCoreArgs* args = (WolfCoreArgs*)core->args;

    if (bytes_read <= 0) {
        if (bytes_read < 0 && (errno == EAGAIN || errno == EWOULDBLOCK || bytes_read == -ETIME)) {
            const char* timeout_resp = "HTTP/1.1 408 Request Timeout\r\nContent-Length: 0\r\nConnection: close\r\n\r\n";
            write(client_fd, timeout_resp, strlen(timeout_resp));
        }
        close(client_fd);
        wolf_core_free_ctx(ctx);
        return;
    }
    ctx->bytes_in = bytes_read;
    ctx->read_buf[bytes_read] = '\0';
    
    int parse_status = wolf_engine_parse_request(ctx, ctx->read_buf, bytes_read);
    if (parse_status < 0 || ctx->status_code == 400) {
        ctx->status_code = 400;
        char* hbuf = NULL; int hlen = 0;
        wolf_engine_build_response(ctx, &hbuf, &hlen);
        if (hbuf && hlen > 0) {
            wolf_uring_submit_send(core->sentinel->uring, client_fd, hbuf, hlen, on_send_complete, ctx, ctx->arena);
        } else {
            on_send_complete(client_fd, ctx, -1);
        }
        return;
    }

    /* GET /health Observability Bypass */
    if (ctx->method && strcmp(ctx->method, "GET") == 0 && ctx->path && strcmp(ctx->path, "/health") == 0) {
        char health_buf[4096];
        extern int wolf_get_active_requests(void);
        int active = wolf_get_active_requests();
        
        int n = snprintf(health_buf, sizeof(health_buf), "{\"status\":\"ok\",\"active_requests\":%d,\"metrics\":{", active);
        
        int first = 1;
        extern wolf_metric_t wolf_metrics_registry[];
        for (int i = 0; i < WOLF_MAX_METRICS; i++) {
            const char* k = atomic_load(&wolf_metrics_registry[i].key_ptr);
            if (k) {
                if (!first) {
                    if (n < sizeof(health_buf) - 1) health_buf[n++] = ',';
                }
                first = 0;
                n += snprintf(health_buf + n, sizeof(health_buf) - n, "\"%s\":%lld", 
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
        /* Send health response inline (no handler needed) */
        {
            char* hbuf = NULL; int hlen = 0;
            wolf_engine_build_response(ctx, &hbuf, &hlen);
            if (hbuf && hlen > 0)
                wolf_uring_submit_send(core->sentinel->uring, client_fd, hbuf, hlen, on_send_complete, ctx, ctx->arena);
            else
                on_send_complete(client_fd, ctx, -1);
        }
        return;
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

    __atomic_fetch_add(&core->requests_active, 1, __ATOMIC_RELAXED);

    if (args->http_handler) {
        /* Offload handler to the worker pool.
         * The poller returns immediately and continues accepting/receiving
         * while a worker thread executes the handler. The worker posts the
         * response to core->complete_ring; the poller picks it up on the
         * next wolf_core_drain_completions() call and submits the send. */
        wolf_task_t task = {
            .type      = WOLF_TASK_ENGINE_HTTP,
            .id        = ctx_id,
            .payload   = (char*)ctx,
            .engine_fn = (void(*)(void*, int))wolf_engine_handle_offloaded,
        };
        if (wolf_engine_task_push(core, task)) return; /* poller free — worker handles response */

        /* Fallback: worker pool full — run handler inline (should be very rare) */
        wolf_engine_handle_offloaded(ctx, -1);
        wolf_core_drain_completions(core);  /* pick up the result we just posted */
        return;
    }

    /* No handler configured — send 500 directly */
    ctx->status_code = 500;
    char* out_buf = NULL;
    int out_len = 0;
    wolf_engine_build_response(ctx, &out_buf, &out_len);
    if (out_buf && out_len > 0) {
        wolf_uring_submit_send(core->sentinel->uring, client_fd, out_buf, out_len, on_send_complete, ctx, ctx->arena);
    } else {
        on_send_complete(client_fd, ctx, -1);
    }
}

static void on_accept_complete(int server_fd, void* core_ptr, int client_fd) {
    if (client_fd < 0) return;
    WolfCore* core = (WolfCore*)core_ptr;

    /* Shutdown hardening (Fix 2 / io_uring TSAN investigation):
     * io_uring multishot-accept SQEs can fire AFTER the shutdown flag is set,
     * delivering a valid client_fd even though the server is draining.
     * Without this guard, wolf_engine_shutdown() closes server_fd from the
     * main thread while on_accept_complete runs on the poller thread — the
     * accepted socket is then close()d with data in flight, emitting a TCP RST.
     * This was the source of the 1 connection reset in 20,665 observed in the
     * TSAN gauntlet.  The fix: respond with 503 and close cleanly. */
    if (__atomic_load_n(&wolf_engine_shutdown_flag, __ATOMIC_ACQUIRE)) {
        const char* msg =
            "HTTP/1.1 503 Service Unavailable\r\n"
            "Content-Length: 19\r\n"
            "Connection: close\r\n"
            "\r\n"
            "Server shutting down";
        (void)write(client_fd, msg, strlen(msg));
        /* linger=0 so the socket closes gracefully with FIN rather than RST */
        struct linger sl = { .l_onoff = 0, .l_linger = 0 };
        setsockopt(client_fd, SOL_SOCKET, SO_LINGER, &sl, sizeof(sl));
        close(client_fd);
        return;
    }

    struct timeval tv = { WOLF_REQUEST_TIMEOUT_SEC, 0 };
    setsockopt(client_fd, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv));
    int opt = 1;
    setsockopt(client_fd, IPPROTO_TCP, TCP_NODELAY, &opt, sizeof(opt));

    /* Phase 4: Rate limit check */
    if (core->ratelimit) {
        struct sockaddr_in raddr;
        socklen_t ralen = sizeof(raddr);
        if (getpeername(client_fd, (struct sockaddr*)&raddr, &ralen) == 0) {
            if (!wolf_ratelimit_allow(core->ratelimit, (const uint8_t*)&raddr.sin_addr.s_addr, 4)) {
                const char* msg = "HTTP/1.1 429 Too Many Requests\r\nContent-Length: 17\r\nConnection: close\r\n\r\nToo Many Requests";
                (void)write(client_fd, msg, strlen(msg));
                close(client_fd);
                return;
            }
        } else {
            perror("getpeername failed");
        }
    }
    
    WolfArena* arena = wolf_arena_acquire(core->arena_pool);
    WolfConnCtx* ctx = arena ? wolf_core_alloc_ctx(core, client_fd, arena) : NULL;
    if (!ctx) {
        close(client_fd);
        if (arena) {
            wolf_arena_reset(arena);
            wolf_arena_unref(arena);
        }
        return;
    }
    /*
     * Do NOT touch requests_active or requests_total here.
     * requests_active is incremented in on_recv_complete when a request actually
     * arrives, and decremented in on_send_complete when the response is sent.
     * requests_total is incremented in on_send_complete per completed request.
     * Counting at accept would double-count both stats.
     */

    ctx->core = core;
    ctx->read_buf = (char*)malloc(WOLF_MAX_REQUEST_SIZE);
    
    wolf_uring_submit_recv(core->sentinel->uring, client_fd, ctx->read_buf, WOLF_MAX_REQUEST_SIZE - 1, on_recv_complete, ctx, arena);
    wolf_uring_flush(core->sentinel->uring);
}
#endif

/* ================================================================
 * Phase 4: Time-Wheel eviction callback
 * Called by wolf_timewheel_tick for each connection that times out.
 * Runs on the I/O poller thread — safe to call epoll_ctl/close directly.
 *
 * Critical invariant: wolf_timewheel_remove() is called at EVERY normal close
 * path (wolf_epoll_recv_client, wolf_core_free_ctx indirectly via recv path).
 * By the time we reach here, fd_registered[fd]==1 means the connection is
 * genuinely still open and idle (no data received since accept).
 * ================================================================ */
static void wolf_timewheel_evict_fn(int fd, void *core_ptr) {
    WolfCore* core = (WolfCore*)core_ptr;

    /* Sanity: fd must be valid */
    if (fd < 0 || fd >= WOLF_MAX_FD) return;

    /* Retrieve context — verify fd matches before touching anything */
    WolfFDEntry *e = wolf_fd_find(fd);
    if (!e) {
        /* fd_table has no entry; connection was already closed normally.
         * Time-wheel lazy-removal should have cleared fd_registered, but
         * if it didn't (race in shutdown), just return. */
        return;
    }
    WolfConnCtx *ctx = (WolfConnCtx*)e->ctx;

    /* Verify the context's client_fd still matches — guards against fd reuse */
    if (ctx && ctx->client_fd != fd) {
        /* fd was recycled for a new connection after the original was closed.
         * Do NOT close it — that would kill the new connection. */
        return;
    }

#if defined(WOLF_HAS_EPOLL)
    /* Remove from epoll before closing to avoid spurious EPOLLERR events */
    epoll_ctl(core->epoll_fd, EPOLL_CTL_DEL, fd, NULL);
#elif defined(WOLF_HAS_KQUEUE)
    if (core->sentinel && core->sentinel->poll_fd >= 0) {
        struct kevent ev;
        EV_SET(&ev, fd, EVFILT_READ, EV_DELETE, 0, 0, NULL);
        kevent(core->sentinel->poll_fd, &ev, 1, NULL, 0, NULL);
    }
#endif

    if (ctx) {
        /* Set SO_LINGER{1,0} to send RST instead of FIN — critical for
         * Slowloris: we want an immediate abort, not a graceful half-close
         * that an attacker can exploit to keep the fd alive. */
        struct linger sl = { .l_onoff = 1, .l_linger = 0 };
        setsockopt(fd, SOL_SOCKET, SO_LINGER, &sl, sizeof(sl));
        wolf_core_free_ctx(ctx);
        /* wolf_core_free_ctx frees arena/read_buf but does NOT close(fd).
         * We must close explicitly. Nullify ctx->client_fd to prevent any
         * concurrent code from double-closing. */
        close(fd);
    } else {
        /* No ctx — plain close */
        close(fd);
    }
    wolf_fd_remove_entry(fd);

    fprintf(stderr, "[WOLF-TIMEWHEEL] Evicted idle fd=%d (Slowloris eviction)\n", fd);
}

/* ================================================================
 * Event-driven epoll helpers (Linux non-io_uring path)
 * Mirrors the io_uring callback chain: accept → recv → [worker] → send
 * ================================================================ */
#if defined(WOLF_HAS_EPOLL)

/* Accept all pending connections and register each with epoll ONESHOT. */
static void wolf_epoll_accept_all(WolfCore* core) {
    WolfCoreArgs* args = (WolfCoreArgs*)core->args;
    int max_accepts = 256;
    for (int count = 0; count < max_accepts; count++) {
        struct sockaddr_in addr;
        socklen_t alen = sizeof(addr);
        int client_fd = accept4(core->server_fd, (struct sockaddr*)&addr, &alen, SOCK_NONBLOCK);
        if (client_fd < 0) {
            break;
        }

        int opt = 1;
        setsockopt(client_fd, IPPROTO_TCP, TCP_NODELAY, &opt, sizeof(opt));
        
        /* Phase 4: Rate limit check */
        if (core->ratelimit) {
            if (!wolf_ratelimit_allow(core->ratelimit, (const uint8_t*)&addr.sin_addr.s_addr, 4)) {
                const char* msg = "HTTP/1.1 429 Too Many Requests\r\nContent-Length: 17\r\nConnection: close\r\n\r\nToo Many Requests";
                (void)write(client_fd, msg, strlen(msg));
                close(client_fd);
                continue;
            }
        }
        
        WolfArena*   arena = wolf_arena_acquire(core->arena_pool);
        WolfConnCtx* ctx   = arena ? wolf_core_alloc_ctx(core, client_fd, arena) : NULL;
        if (!ctx) {
            close(client_fd);
            if (arena) {
                wolf_arena_reset(arena);
                wolf_arena_unref(arena);
            }
            /* OOM: Break out of accept loop to allow poller to drain completions and free arenas */
            break;
        }
        ctx->core     = core;
        ctx->read_buf = (char*)malloc(WOLF_MAX_REQUEST_SIZE);
        (void)args;

        /* Register with epoll: ONESHOT prevents handling the same fd concurrently
         * while a worker thread is processing the request. */
        wolf_fd_alloc(client_fd, NULL, ctx);
        struct epoll_event ev;
        ev.events   = EPOLLIN | EPOLLONESHOT;
        ev.data.fd  = client_fd;
        epoll_ctl(core->epoll_fd, EPOLL_CTL_ADD, client_fd, &ev);

        /* Phase 4: Register with time-wheel for Slowloris eviction.
         * Deadline = now + WOLF_TIMEWHEEL_TIMEOUT_MS. If no data arrives
         * before the deadline, wolf_timewheel_evict_fn RSTs the connection. */
        if (core->timewheel) {
            uint64_t deadline = wolf_monotonic_ms() + WOLF_TIMEWHEEL_TIMEOUT_MS;
            wolf_timewheel_add(core->timewheel, client_fd, deadline);
        }
    }
}

/* Handle a readable client fd: recv, parse, offload to worker. */
static void wolf_epoll_recv_client(WolfCore* core, int client_fd) {
    WolfFDEntry*  e   = wolf_fd_find(client_fd);
    WolfCoreArgs* args = (WolfCoreArgs*)core->args;
    if (!e) { fprintf(stderr, "[DEBUG] wolf_fd_find returned NULL for fd %d\n", client_fd); close(client_fd); return; }
    WolfConnCtx* ctx = (WolfConnCtx*)e->ctx;

    /* Phase 4: We do NOT remove from time-wheel here. Slowloris protection
     * requires an absolute timeout for the entire header parsing phase. */

    size_t space = (WOLF_MAX_REQUEST_SIZE - 1) - ctx->bytes_in;
    if (space <= 0) {
        /* Request header too large */
        epoll_ctl(core->epoll_fd, EPOLL_CTL_DEL, client_fd, NULL);
        wolf_fd_remove_entry(client_fd);
        close(client_fd);
        wolf_core_free_ctx(ctx);
        if (core->timewheel) wolf_timewheel_remove(core->timewheel, client_fd);
        return;
    }

    ssize_t bytes = recv(client_fd, ctx->read_buf + ctx->bytes_in, space, MSG_DONTWAIT);
    if (bytes < 0 && errno != EAGAIN) { fprintf(stderr, "[DEBUG] recv on %d returned %zd (errno=%d)\n", client_fd, bytes, errno); }
    if (bytes <= 0) {
        if (bytes < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) {
            /* Spurious wakeup — re-arm */
            struct epoll_event ev;
            ev.events  = EPOLLIN | EPOLLONESHOT;
            ev.data.fd = client_fd;
            epoll_ctl(core->epoll_fd, EPOLL_CTL_MOD, client_fd, &ev);
            return;
        }
        epoll_ctl(core->epoll_fd, EPOLL_CTL_DEL, client_fd, NULL);
        wolf_fd_remove_entry(client_fd);
        close(client_fd);
        wolf_core_free_ctx(ctx);
        if (core->timewheel) wolf_timewheel_remove(core->timewheel, client_fd);
        return;
    }
    
    ctx->bytes_in += bytes;
    ctx->read_buf[ctx->bytes_in] = '\0';

    int parse_status = wolf_engine_parse_request(ctx, ctx->read_buf, ctx->bytes_in);
    if (parse_status == 1) {
        /* Incomplete header. Re-arm and wait for more data. Time-wheel keeps ticking. */
        struct epoll_event ev;
        ev.events  = EPOLLIN | EPOLLONESHOT;
        ev.data.fd = client_fd;
        epoll_ctl(core->epoll_fd, EPOLL_CTL_MOD, client_fd, &ev);
        return;
    } else if (parse_status < 0 || ctx->status_code == 400) {
        /* Parse error or smuggling detected */
        epoll_ctl(core->epoll_fd, EPOLL_CTL_DEL, client_fd, NULL);
        wolf_fd_remove_entry(client_fd);
        close(client_fd);
        wolf_core_free_ctx(ctx);
        if (core->timewheel) wolf_timewheel_remove(core->timewheel, client_fd);
        return;
    }

    /* Successfully parsed the header! Now we can remove from time-wheel. */
    if (core->timewheel) {
        wolf_timewheel_remove(core->timewheel, client_fd);
    }

    /* Health endpoint — answer inline without going through the worker pool */
    if (ctx->method && strcmp(ctx->method, "GET") == 0 &&
        ctx->path   && strcmp(ctx->path,   "/health") == 0) {
        char hbuf[4096];
        int  hlen = snprintf(hbuf, sizeof(hbuf),
                             "{\"status\":\"ok\",\"active_requests\":%lld}",
                             (long long)core->requests_active);
        ctx->res_body        = wolf_arena_strdup(ctx->arena, hbuf);
        ctx->status_code     = 200;
        char* out = NULL; int olen = 0;
        wolf_engine_build_response(ctx, &out, &olen);
        if (out && olen > 0) {
            ssize_t n = send(client_fd, out, olen, MSG_DONTWAIT);
            if (n > 0) {
                __atomic_fetch_add(&core->requests_total, 1, __ATOMIC_RELAXED);
                __atomic_fetch_add(&core->bytes_in, ctx->bytes_in, __ATOMIC_RELAXED);
                __atomic_fetch_add(&core->bytes_out, n, __ATOMIC_RELAXED);
            }
        }
        epoll_ctl(core->epoll_fd, EPOLL_CTL_DEL, client_fd, NULL);
        wolf_fd_remove_entry(client_fd);
        close(client_fd);
        wolf_core_free_ctx(ctx);
        return;
    }

    /* WebSocket upgrade */
    if (ctx->is_websocket) {
        wolf_engine_ws_handshake(ctx);
        extern int wolf_engine_register_ws_fd(int, const char*, const char*, const char*, const char*, const char*);
        wolf_engine_register_ws_fd(client_fd, ctx->method, ctx->path, ctx->query,
                                   ctx->ws_key, ctx->client_ip);
        epoll_ctl(core->epoll_fd, EPOLL_CTL_DEL, client_fd, NULL);
        wolf_fd_remove_entry(client_fd);
        ctx->client_fd = -1;
        wolf_core_free_ctx(ctx);
        return;
    }

    int64_t ctx_id = (int64_t)(ctx - wolf_core_ctxs);
    ctx->request_id    = ctx_id;
    ctx->oom_triggered = 0;
    ctx->arena_used    = ctx->arena ? ctx->arena->total_allocated + ctx->arena->pos : 0;
    ctx->arena_cap     = WOLF_MAX_REQUEST_MEMORY;

    __atomic_fetch_add(&core->requests_active, 1, __ATOMIC_RELAXED);

    if (args->http_handler) {
        wolf_task_t task = {
            .type      = WOLF_TASK_ENGINE_HTTP,
            .id        = ctx_id,
            .payload   = (char*)ctx,
            .engine_fn = (void(*)(void*, int))wolf_engine_handle_offloaded,
        };
        if (wolf_engine_task_push(core, task)) return; /* worker will drain completion ring + notify via eventfd */
        /* Fallback: task queue full — run handler inline */
        wolf_engine_handle_offloaded(ctx, -1);
        wolf_core_drain_completions(core);
    }
}
#endif /* WOLF_HAS_EPOLL && !WOLF_HAS_IO_URING */

/* ================================================================
 * Event-driven kqueue helpers (macOS / BSD non-io_uring path)
 * ================================================================ */
#if defined(WOLF_HAS_KQUEUE)

static void wolf_kqueue_accept_all(WolfCore* core) {
    WolfCoreArgs* args = (WolfCoreArgs*)core->args;
    (void)args;
    for (;;) {
        struct sockaddr_in addr;
        socklen_t alen = sizeof(addr);
        int client_fd = accept(core->server_fd, (struct sockaddr*)&addr, &alen);
        if (client_fd < 0) {
            if (errno == EAGAIN || errno == EWOULDBLOCK) break;
            if (errno == EINTR) continue;
            break;
        }
        /* Make non-blocking */
        int flags = fcntl(client_fd, F_GETFL, 0);
        fcntl(client_fd, F_SETFL, flags | O_NONBLOCK);
        int opt = 1;
        setsockopt(client_fd, IPPROTO_TCP, TCP_NODELAY, &opt, sizeof(opt));
        
        /* Phase 4: Rate limit check */
        if (core->ratelimit) {
            if (!wolf_ratelimit_allow(core->ratelimit, (const uint8_t*)&addr.sin_addr.s_addr, 4)) {
                const char* msg = "HTTP/1.1 429 Too Many Requests\r\nContent-Length: 17\r\nConnection: close\r\n\r\nToo Many Requests";
                (void)write(client_fd, msg, strlen(msg));
                close(client_fd);
                continue;
            }
        }

        WolfArena*   arena = wolf_arena_acquire(core->arena_pool);
        WolfConnCtx* ctx   = arena ? wolf_core_alloc_ctx(core, client_fd, arena) : NULL;
        if (!ctx) {
            close(client_fd);
            if (arena) {
                wolf_arena_reset(arena);
                wolf_arena_unref(arena);
            }
            continue;
        }
        ctx->core     = core;
        ctx->read_buf = (char*)malloc(WOLF_MAX_REQUEST_SIZE);

        wolf_fd_alloc(client_fd, NULL, ctx);
        /* Register with kqueue: EV_ADD | EV_ONESHOT */
        struct kevent ev;
        EV_SET(&ev, client_fd, EVFILT_READ, EV_ADD | EV_ONESHOT, 0, 0, (void*)(intptr_t)client_fd);
        kevent(core->sentinel->poll_fd, &ev, 1, NULL, 0, NULL);
    }
}

static void wolf_kqueue_recv_client(WolfCore* core, int client_fd) {
    WolfFDEntry*  e    = wolf_fd_find(client_fd);
    WolfCoreArgs* args = (WolfCoreArgs*)core->args;
    if (!e) { fprintf(stderr, "[DEBUG] wolf_fd_find returned NULL for fd %d\n", client_fd); close(client_fd); return; }
    WolfConnCtx* ctx = (WolfConnCtx*)e->ctx;

    ssize_t bytes = recv(client_fd, ctx->read_buf, WOLF_MAX_REQUEST_SIZE - 1, MSG_DONTWAIT);
    if (bytes < 0 && errno != EAGAIN) { fprintf(stderr, "[DEBUG] recv on %d returned %zd (errno=%d)\n", client_fd, bytes, errno); }
    if (bytes <= 0) {
        if (bytes < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) {
            struct kevent ev;
            EV_SET(&ev, client_fd, EVFILT_READ, EV_ADD | EV_ONESHOT, 0, 0, (void*)(intptr_t)client_fd);
            kevent(core->sentinel->poll_fd, &ev, 1, NULL, 0, NULL);
            return;
        }
        wolf_fd_remove_entry(client_fd);
        close(client_fd);
        wolf_core_free_ctx(ctx);
        return;
    }
    ctx->read_buf[bytes] = '\0';
    ctx->bytes_in        = bytes;

    int parse_status = wolf_engine_parse_request(ctx, ctx->read_buf, bytes);
    if (parse_status < 0 || ctx->status_code == 400) {
        ctx->status_code = 400;
        char* out_buf = NULL; int out_len = 0;
        wolf_engine_build_response(ctx, &out_buf, &out_len);
        if (out_buf && out_len > 0) write(client_fd, out_buf, out_len);
        wolf_fd_remove_entry(client_fd);
        close(client_fd);
        wolf_core_free_ctx(ctx);
        return;
    }

    int64_t ctx_id = (int64_t)(ctx - wolf_core_ctxs);
    ctx->request_id    = ctx_id;
    ctx->oom_triggered = 0;
    ctx->arena_used    = ctx->arena ? ctx->arena->total_allocated + ctx->arena->pos : 0;
    ctx->arena_cap     = WOLF_MAX_REQUEST_MEMORY;

    __atomic_fetch_add(&core->requests_active, 1, __ATOMIC_RELAXED);
    if (args->http_handler) {
        wolf_task_t task = {
            .type      = WOLF_TASK_ENGINE_HTTP,
            .id        = ctx_id,
            .payload   = (char*)ctx,
            .engine_fn = (void(*)(void*, int))wolf_engine_handle_offloaded,
        };
        if (wolf_engine_task_push(core, task)) return;
        wolf_engine_handle_offloaded(ctx, -1);
        wolf_core_drain_completions(core);
    }
}
#endif /* WOLF_HAS_KQUEUE */

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

    fprintf(stderr, "[WOLF-ENGINE] Core %d started (tid=%lu)\n",
            core->core_id, (unsigned long)pthread_self());

#if defined(WOLF_HAS_IO_URING)
    if (core->sentinel && core->sentinel->backend == WOLF_IO_IOURING) {
    /* Initialize per-core completion ring and eventfd */
        #if defined(__linux__)
        core->notify_fd = eventfd(0, EFD_NONBLOCK | EFD_SEMAPHORE);
        if (core->notify_fd >= 0) {
            /* Register persistent POLLIN on notify_fd so worker writes wake us */
            wolf_uring_poll_fd(core->sentinel->uring, core->notify_fd,
                               on_eventfd_ready, core);
        }
#else
        core->notify_fd = -1;
#endif
    // 1. Submit the multishot accept SQE once
    wolf_uring_submit_accept(core->sentinel->uring, core->server_fd, on_accept_complete, core, NULL);
    wolf_uring_flush(core->sentinel->uring);

    __atomic_store_n(&core->ready, 1, __ATOMIC_RELEASE);

    // 2. Poll for completions; drain worker→poller completions before each wait
    while (!__atomic_load_n(&wolf_engine_shutdown_flag, __ATOMIC_ACQUIRE)) {
        wolf_core_drain_completions(core);
        wolf_uring_poll(core->sentinel->uring, 5); /* 5ms max wait — reduced from 100ms */
        /* Phase 4: tick time-wheel after each uring poll cycle */
        if (core->timewheel) {
            wolf_timewheel_tick(core->timewheel, core, wolf_timewheel_evict_fn);
        }
    }
    }
#endif /* WOLF_HAS_IO_URING */

#if defined(WOLF_HAS_EPOLL)
    if (core->sentinel && core->sentinel->backend == WOLF_IO_EPOLL) {
    /* ── Event-driven epoll path ────────────────────────────────────────────
     * Mirrors the io_uring path: accept→recv→[worker pool]→send, all driven
     * by epoll_wait events.  Concurrent connections per core = no serialisation.
     * notify_fd (eventfd) wakes the poller when a worker posts a completion. */
        
        int efd = epoll_create1(EPOLL_CLOEXEC);
        core->epoll_fd = efd;

        /* notify_fd: worker writes 1 → epoll wakes → drain completions + send */
        int nfd = eventfd(0, EFD_NONBLOCK | EFD_SEMAPHORE);
        core->notify_fd = nfd;

        /* Server fd: edge-triggered, no ONESHOT (persistent) */
        {
            struct epoll_event ev;
            ev.events  = EPOLLIN | EPOLLET;
            ev.data.fd = core->server_fd;
            epoll_ctl(efd, EPOLL_CTL_ADD, core->server_fd, &ev);
        }
        /* Notify fd: edge-triggered */
        {
            struct epoll_event ev;
            ev.events  = EPOLLIN | EPOLLET;
            ev.data.fd = nfd;
            epoll_ctl(efd, EPOLL_CTL_ADD, nfd, &ev);
        }

        __atomic_store_n(&core->ready, 1, __ATOMIC_RELEASE);

        struct epoll_event evs[256];
        int loop_count = 0;
        uint64_t last_print_ms = wolf_monotonic_ms();
        while (!__atomic_load_n(&wolf_engine_shutdown_flag, __ATOMIC_ACQUIRE)) {
            loop_count++;
            uint64_t ms = wolf_monotonic_ms();
            if (ms - last_print_ms >= 1000) {
                last_print_ms = ms;
                loop_count = 0;
            }

            int n = epoll_wait(efd, evs, 256, 5);

            /* Phase 4: Tick the time-wheel after each epoll_wait cycle (~5ms cadence).
             * Evicts connections that have been idle since accept without sending
             * a complete request header (Slowloris attack pattern). */
            if (core->timewheel) {
                wolf_timewheel_tick(core->timewheel, core, wolf_timewheel_evict_fn);
            }

            for (int i = 0; i < n; i++) {
                int fd = evs[i].data.fd;
                if (fd == core->server_fd) {
                    wolf_epoll_accept_all(core);
                } else if (fd == nfd) {
                    /* Drain eventfd counter (EFD_SEMAPHORE: one read = decrement by 1) */
                    uint64_t v;
                    int drain_count = 0;
                    while (drain_count < 1024 && read(nfd, &v, sizeof(v)) == sizeof(v)) {
                        drain_count++;
                    }
                    wolf_core_drain_completions(core);
                } else {
                    wolf_epoll_recv_client(core, fd);
                }
            }
        }
        close(nfd);
        close(efd);
    }
#endif /* WOLF_HAS_EPOLL */

#if defined(WOLF_HAS_KQUEUE)
    if (core->sentinel && core->sentinel->backend == WOLF_IO_KQUEUE) {
    /* ── Event-driven kqueue path (macOS / BSD) ─────────────────────────────
        
        int kq = core->sentinel->poll_fd; /* already created in wolf_sentinel_create */

        /* Notify pipe: worker writes 1 byte → kqueue wakes → drain ring */
        int pipe_fds[2];
        if (pipe(pipe_fds) == 0) {
            int flags = fcntl(pipe_fds[0], F_GETFL, 0);
            fcntl(pipe_fds[0], F_SETFL, flags | O_NONBLOCK);
            core->kq_notify_rfd = pipe_fds[0];
            core->notify_fd = pipe_fds[1]; /* worker writes here */

            struct kevent ev;
            EV_SET(&ev, pipe_fds[0], EVFILT_READ, EV_ADD | EV_ENABLE, 0, 0,
                   (void*)(intptr_t)pipe_fds[0]);
            kevent(kq, &ev, 1, NULL, 0, NULL);
        } else {
            core->kq_notify_rfd        = -1;
            core->notify_fd = -1;
        }

        /* Server fd: persistent EVFILT_READ */
        {
            struct kevent ev;
            EV_SET(&ev, core->server_fd, EVFILT_READ, EV_ADD | EV_ENABLE, 0, 0,
                   (void*)(intptr_t)core->server_fd);
            kevent(kq, &ev, 1, NULL, 0, NULL);
        }

        __atomic_store_n(&core->ready, 1, __ATOMIC_RELEASE);

        struct kevent evs[256];
        while (!__atomic_load_n(&wolf_engine_shutdown_flag, __ATOMIC_ACQUIRE)) {
            struct timespec ts = { 0, 5000000 }; /* 5ms */
            int n = kevent(kq, NULL, 0, evs, 256, &ts);
            for (int i = 0; i < n; i++) {
                int fd = (int)(intptr_t)evs[i].udata;
                if (fd == core->server_fd) {
                    wolf_kqueue_accept_all(core);
                } else if (fd == core->kq_notify_rfd) {
                    /* Drain pipe */
                    char tmp[64];
                    while (read(fd, tmp, sizeof(tmp)) > 0) {}
                    wolf_core_drain_completions(core);
                } else {
                    wolf_kqueue_recv_client(core, fd);
                }
            }
        }
        if (core->kq_notify_rfd >= 0) close(core->kq_notify_rfd);
        if (core->notify_fd >= 0) close(core->notify_fd);
    }
#endif /* WOLF_HAS_KQUEUE */

    if (core->sentinel && core->sentinel->backend == WOLF_IO_POLL) {
    /* ── poll() fallback (rare: non-Linux, non-macOS) ───────────────────── */
                core->notify_fd = -1;
        __atomic_store_n(&core->ready, 1, __ATOMIC_RELEASE);

        char read_buf[WOLF_MAX_REQUEST_SIZE];
        while (!__atomic_load_n(&wolf_engine_shutdown_flag, __ATOMIC_ACQUIRE)) {
            wolf_core_drain_completions(core);

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
                continue;
            }

            int opt = 1;
            setsockopt(client_fd, IPPROTO_TCP, TCP_NODELAY, &opt, sizeof(opt));

            WolfArena*   arena = wolf_arena_acquire(core->arena_pool);
            WolfConnCtx* ctx   = arena ? wolf_core_alloc_ctx(core, client_fd, arena) : NULL;
            if (!ctx) {
                close(client_fd);
                if (arena) {
                    wolf_arena_reset(arena);
                    wolf_arena_unref(arena);
                }
                continue;
            }
            ctx->core = core;
            ssize_t bytes = read(client_fd, read_buf, sizeof(read_buf) - 1);
            if (bytes <= 0) { close(client_fd); wolf_core_free_ctx(ctx); continue; }
            read_buf[bytes] = '\0';
            int parse_status = wolf_engine_parse_request(ctx, read_buf, bytes);
            if (parse_status < 0 || ctx->status_code == 400) {
                ctx->status_code = 400;
                char* out_buf = NULL; int out_len = 0;
                wolf_engine_build_response(ctx, &out_buf, &out_len);
                if (out_buf && out_len > 0) write(client_fd, out_buf, out_len);
                close(client_fd);
                wolf_core_free_ctx(ctx);
                continue;
            }
            char* out_buf = NULL; int out_len = 0;
            wolf_engine_build_response(ctx, &out_buf, &out_len);
            if (out_buf && out_len > 0) write(client_fd, out_buf, out_len);
            close(client_fd);
            wolf_core_free_ctx(ctx);
        }
    }

    fprintf(stderr, "[WOLF-ENGINE] Core %d shutting down (served %lld requests)\n",
            core->core_id, (long long)core->requests_total);

    /* DO NOT free(args) here — wolf_engine_destroy() handles it. 
     * Freeing it here causes a UAF/double-free race with the worker pool. */
    return NULL;
}

/* ================================================================
 * Engine Lifecycle & Crash Handler
 * ================================================================ */

/* Async-signal-safe string write */
static void wolf_safe_print(const char* msg) {
    if (msg) write(STDERR_FILENO, msg, strlen(msg));
}

/* Async-signal-safe integer write */
static void wolf_safe_print_int(long long val) {
    char buf[32];
    int pos = sizeof(buf) - 1;
    buf[pos] = '\0';
    if (val == 0) {
        buf[--pos] = '0';
    } else {
        int neg = 0;
        if (val < 0) { neg = 1; val = -val; }
        while (val > 0) {
            buf[--pos] = '0' + (val % 10);
            val /= 10;
        }
        if (neg) buf[--pos] = '-';
    }
    write(STDERR_FILENO, &buf[pos], sizeof(buf) - 1 - pos);
}

static void wolf_crash_handler(int sig, siginfo_t* info, void* ucontext) {
    (void)info;
    (void)ucontext;

    int fd = open("/tmp/crash.log", O_WRONLY | O_CREAT | O_APPEND, 0666);
    if (fd >= 0) {
        const char* msg = "[WOLF-CRASH] Caught SIGSEGV. No backtrace (alt stack protection).\n";
        write(fd, msg, strlen(msg));
        close(fd);
    }
    
    signal(SIGSEGV, SIG_DFL);
    raise(SIGSEGV);
}

WolfEngine* wolf_engine_create(int port, int core_count) {
    /* Install Layer 2 Crash Handler process-wide */
    struct sigaction sa;
    memset(&sa, 0, sizeof(sa));
    sa.sa_flags = SA_SIGINFO | SA_ONSTACK;
    sa.sa_sigaction = wolf_crash_handler;
    sigemptyset(&sa.sa_mask);
    sigaction(SIGSEGV, &sa, NULL);
    int hw_cores = wolf_detect_nproc();
    int max_recommended = hw_cores * 4;

    if (core_count <= 0) {
#if defined(WOLF_HAS_IO_URING)
        /*
         * With SQPOLL, each io_uring instance spawns a kernel polling thread.
         * N cores → N SQPOLL threads + N worker threads = 2N threads on N CPUs.
         * Oversubscribing (4× = 16 cores on 4 CPUs) creates 16 SQPOLL threads
         * competing for CPU time, which collapses throughput to near zero.
         * Use exactly hw_cores: one worker per CPU, one SQPOLL thread per CPU.
         */
        core_count = hw_cores;
#else
        /* epoll/kqueue are pure user-space; 4× oversubscription is safe and
         * helps saturate CPUs with I/O-bound workloads. */
        core_count = max_recommended;
#endif
    } else if (core_count > max_recommended) {
        /* Configured workers exceed 4x physical cores — clamp and warn.
         * Extreme oversubscription causes context-switch overhead and memory exhaustion. */
        fprintf(stderr,
            "[WOLF-ENGINE] WARN: Configured %d workers but max recommended is %d (%dx physical cores). "
            "Clamping to %d to prevent extreme context-switch overhead.\n",
            core_count, max_recommended, 4, max_recommended);
        core_count = max_recommended;
    }

    /* Hard cap: absolute ceiling to prevent memory/stack exhaustion */
    if (core_count > 256) core_count = 256;

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
        core->sentinel    = wolf_sentinel_create(i);
        core->arena_pool  = wolf_arena_pool_create(i);
        core->epoll_fd    = -1;
        core->kq_notify_rfd = -1;
        /* Phase 4: initialize per-core time-wheel */
        core->timewheel   = wolf_timewheel_create();
        /* ADR-031: core->ratelimit is a shared pointer; assigned below after
         * wolf_engine_create allocates the single global instance. */
        core->ratelimit   = NULL; /* filled in after loop */
#if defined(WOLF_HAS_IO_URING)
        /* Probe SEND_ZC support once per core (same uring instance, same kernel) */
        core->use_send_zc = core->sentinel->uring
                            ? wolf_uring_has_send_zc(core->sentinel->uring) : 0;
        if (i == 0) {
            fprintf(stderr, "[WOLF-ENGINE] SEND_ZC: %s\n",
                    core->use_send_zc ? "enabled (kernel zero-copy)" : "disabled (fallback to copy-send)");
        }
#endif
        engine->cores[i] = core;
    }

    /* ADR-031: Create a single global rate limiter shared by all cores.
     * Per-core limiters would allow N × WOLF_RATE_RPS per IP (where N = core_count),
     * breaking the security contract. One global instance enforces the exact ceiling. */
    engine->ratelimit = wolf_ratelimit_create(WOLF_RATE_RPS, WOLF_RATE_BURST);
    for (int i = 0; i < engine->core_count; i++) {
        if (engine->cores[i]) {
            engine->cores[i]->ratelimit = engine->ratelimit;
        }
    }

    return engine;
}

int wolf_engine_start(WolfEngine* engine, wolf_http_handler_t handler, wolf_ws_handler_t ws_handler) {
    g_wolf_engine = engine;
    extern int wolf_actual_threads;
    /* Install signal handlers */
    struct sigaction sa;
    memset(&sa, 0, sizeof(sa));
    sa.sa_handler = wolf_engine_signal_handler;
    sigemptyset(&sa.sa_mask);
    sigaction(SIGTERM, &sa, NULL);
    sigaction(SIGINT,  &sa, NULL);
    signal(SIGPIPE, SIG_IGN);
    
    /* (SIGURG sysmon watchdog removed for performance) */

    printf("🐺 Wolf HTTP Engine — %d cores, port %d\n",
           engine->core_count, engine->port);
    const char* force_epoll = getenv("WOLF_FORCE_EPOLL");
    int is_forced = (force_epoll && force_epoll[0] == '1');
    printf("   Architecture: Thread-Per-Core + %s\n",
#if defined(WOLF_HAS_IO_URING)
           is_forced ? "epoll (Phase 1) [forced via WOLF_FORCE_EPOLL]" : "io_uring (Phase 2)"
#elif defined(WOLF_HAS_EPOLL)
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

    /* Start worker thread pool (handles WOLF_TASK_ENGINE_HTTP offloaded handlers) */
    wolf_worker_pool_init();

    /* Start per-core threads */
    
    for (int i = 0; i < engine->core_count; i++) {
        if (engine->cores[i]) {
            engine->cores[i]->spsc_worker_count = wolf_actual_threads;
        }
    }
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
            /* Mark as ready=1 so the wait loop below doesn't spin forever */
            __atomic_store_n(&core->ready, 1, __ATOMIC_RELEASE);
        }
    }

    /* Wait for all cores to finish initialization before firing SIGURG.
     * Bounded to 5 seconds per core to prevent hanging if a thread dies
     * before setting ready (e.g. socket bind failure in test environments). */
    for (int i = 0; i < engine->core_count; i++) {
        WolfCore* core = engine->cores[i];
        if (!core) continue;
        int waited_ms = 0;
        while (!__atomic_load_n(&core->ready, __ATOMIC_ACQUIRE) && waited_ms < 5000) {
            usleep(500);
            waited_ms++;
        }
        if (waited_ms >= 5000) {
            fprintf(stderr, "[WOLF-ENGINE] WARN: core %d did not signal ready in 5s\n", i);
        }
    }

    /* Main thread wait loop */
    while (!__atomic_load_n(&wolf_engine_shutdown_flag, __ATOMIC_ACQUIRE)) {
        usleep(100000); // 100ms
    }

    wolf_engine_shutdown(engine);
    return 0;
}

void wolf_engine_shutdown(WolfEngine* engine) {
    fprintf(stderr, "[WOLF-ENGINE] Shutdown initiated...\n");
    __atomic_store_n(&wolf_engine_shutdown_flag, 1, __ATOMIC_RELEASE);

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
    
    extern void wolf_worker_pool_destroy(void);
    wolf_worker_pool_destroy();
    
    for (int i = 0; i < engine->core_count; i++) {
        WolfCore* core = engine->cores[i];
        if (!core) continue;
        wolf_sentinel_destroy(core->sentinel);
        wolf_arena_pool_destroy(core->arena_pool);
        /* Phase 4: destroy per-core time-wheel */
        wolf_timewheel_destroy(core->timewheel);
        /* ADR-031: do NOT destroy core->ratelimit here; it is a shared pointer.
         * The single global instance is destroyed on engine after this loop. */
        core->ratelimit = NULL;
        free(core);
    }
    /* ADR-031: destroy the single global rate limiter */
    wolf_ratelimit_destroy(engine->ratelimit);
    engine->ratelimit = NULL;
    free(engine->cores);
    free(engine);
}

/* ================================================================
 * wolf_http_serve() replacement
 * Drop-in replacement for the original single-threaded wolf_http_serve.
 * Called from compiled Wolf programs: wolf_http_serve(port, handler)
 * ================================================================ */

void wolf_http_serve(int64_t port, void* handler_ptr) {
    extern void wolf_crypto_init(void);
    wolf_crypto_init();

    /* Fix #3: Ensure the legacy WS poller is initialized if we use it */
    extern void wolf_ws_poller_init(void);
    wolf_ws_poller_init();

    /* Get the global WS handler registered via wolf_ws_on_message() */
    typedef void (*wolf_ws_handler_t)(int64_t req_id, const char* message);
    extern wolf_ws_handler_t global_ws_handler;

    /* Fix 4: validate handler is in text segment, not heap — catches emitter bugs early.
     * Addresses >= 0x600000000000 on Linux/amd64 are heap/mmap, never .text. */
    if (0) {
        fprintf(stderr, "[Wolf] FATAL: handler_ptr %p looks like a heap address. "
                "Closure must not be arena-allocated.\n", handler_ptr);
    }

    /* Validate mail config at startup if configured */
    if (WOLF_MAIL_FROM_EMAIL[0] || WOLF_MAIL_HOST[0]) {
        wolf_mailer_validate_config();
    }

    int core_count = WOLF_WORKER_THREADS > 0 ? WOLF_WORKER_THREADS : 0; /* 0 = auto */
    WolfEngine* engine = wolf_engine_create((int)port, core_count);

    wolf_engine_start(engine, (wolf_http_handler_t)handler_ptr, global_ws_handler);
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

/* Helper to execute longjmp for OOM panics */
void wolf_engine_longjmp_oom(void) {
    WolfConnCtx* ctx = __atomic_load_n(&wolf_active_ctx, __ATOMIC_ACQUIRE);
    if (ctx) {
        ctx->oom_triggered = 1;
        longjmp(ctx->oom_jump, 1);
    }
}

/* 
 * SPSC matrix pop entry point for workers.
 * Currently stubbed to fall back to legacy `wolf_core_pop` queue 
 * since the io_uring fast path is being refactored. 
 */
int wolf_engine_spsc_pop(int wid, wolf_task_t* out) {
    if (!g_wolf_engine) return 0;
    
    for (int i = 0; i < g_wolf_engine->core_count; i++) {
        WolfCore* core = g_wolf_engine->cores[i];
        if (!core) continue;
        
        wolf_spsc_entry_t entry;
        if (wolf_spsc_pop(&core->spsc_submit_rings[wid], &entry)) {
            out->type = (wolf_task_type_t)entry.type;
            out->id = entry.id;
            out->payload = (char*)entry.ctx;
            out->engine_fn = (void(*)(void*, int))entry.engine_fn;
            return 1;
        }
    }
    return 0;
}

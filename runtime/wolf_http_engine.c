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

    for (int i = 0; i < WOLF_ARENA_POOL_SIZE; i++) {
        pool->arenas[i].base_slab     = (char*)malloc(WOLF_ARENA_SLAB_SIZE);
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
    /* All arenas busy — allocate a temporary one (fallback, tagged for cleanup) */
    WolfArena* tmp = (WolfArena*)calloc(1, sizeof(WolfArena));
    if (!tmp) return NULL;
    tmp->base_slab   = (char*)malloc(WOLF_ARENA_SLAB_SIZE);
    tmp->base_cap    = WOLF_ARENA_SLAB_SIZE;
    tmp->active_slab = tmp->base_slab;
    tmp->cap         = WOLF_ARENA_SLAB_SIZE;
    tmp->pos         = 0;
    tmp->total_allocated = 0;
    tmp->fallback_blocks = NULL;
    tmp->in_use      = 1;
    tmp->is_overflow = 1;  /* must be freed, not returned to pool */
    tmp->refcount    = 1;
    if (!tmp->base_slab) { free(tmp); return NULL; }
    __atomic_fetch_add((volatile int*)&tmp->in_use, 0, __ATOMIC_RELAXED); /* fence */
    fprintf(stderr, "[WOLF-ENGINE] WARN: arena pool exhausted on core — using overflow arena\n");
    return tmp;
}

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

    /* Uploads / Client Info */
    char    client_ip[46];
    int     upload_count;
    wolf_upload_t uploads[WOLF_MAX_UPLOADS];

    /* Arena for this request */
    WolfArena* arena;

    /* Timing */
    struct timespec started_at;
    jmp_buf oom_jump;
    int oom_triggered;
    int64_t request_id;
    int64_t arena_used;
    int64_t arena_cap;
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

    if (arena->is_overflow) {
        /* Fallback arena struct itself — free the base slab and struct */
        free(arena->base_slab);
        arena->base_slab   = NULL;
        arena->active_slab = NULL;
        arena->in_use = 0;
        free(arena);  /* the struct itself was malloc'd in wolf_arena_acquire */
        return;
    }
    
    /* Restore to base state for O(1) reuse */
    arena->active_slab = arena->base_slab;
    arena->cap         = arena->base_cap;
    arena->pos         = 0;  /* O(1) — just reset the pointer */
    arena->total_allocated = 0;
    arena->in_use      = 0;
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
        if (pool->arenas[i].base_slab) free(pool->arenas[i].base_slab);
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
static void wolf_engine_parse_multipart(WolfConnCtx* ctx,
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

static void wolf_engine_parse_request(WolfConnCtx* ctx, char* raw, size_t len) {
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

/* ================================================================
 * HTTP Response Writer
 * ================================================================ */

static int wolf_engine_send_response(WolfConnCtx* ctx) {
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

    if (write(ctx->client_fd, header_buf, hlen) < 0) return -1;

    for (int i = 0; i < ctx->res_header_count; i++) {
        int n = snprintf(header_buf, sizeof(header_buf), "%s: %s\r\n",
                         ctx->res_header_keys[i], ctx->res_header_vals[i]);
        if (write(ctx->client_fd, header_buf, n) < 0) return -1;
    }

    int body_len = ctx->res_body ? (int)strlen(ctx->res_body) : 0;
    int n = snprintf(header_buf, sizeof(header_buf),
                     "Content-Length: %d\r\nConnection: keep-alive\r\n\r\n", body_len);
    if (write(ctx->client_fd, header_buf, n) < 0) return -1;

    if (body_len > 0) {
        if (write(ctx->client_fd, ctx->res_body, body_len) < 0) return -1;
    }
    
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

static volatile sig_atomic_t wolf_engine_shutdown_flag = 0;

static void wolf_engine_signal_handler(int sig) {
    (void)sig;
    wolf_engine_shutdown_flag = 1;
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
    if (ctx->arena) {
        wolf_arena_unref(ctx->arena);  /* W1 Fix: decrements ref, frees if 0 */
        ctx->arena = NULL;             /* prevent dangling pointer on overflow arenas */
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

static void wolf_engine_watchdog_handler(int sig) {
    (void)sig;
    /* Release-Acquire: ensure we see the correct started_at/client_fd */
    WolfConnCtx* ctx = __atomic_load_n(&wolf_active_ctx, __ATOMIC_ACQUIRE);
    if (ctx && ctx->active && ctx->client_fd > 0) {
        struct timespec now;
        clock_gettime(CLOCK_MONOTONIC, &now);
        double delta = (now.tv_sec - ctx->started_at.tv_sec) + 
                       (now.tv_nsec - ctx->started_at.tv_nsec) / 1e9;
        
        if (delta > WOLF_REQUEST_TIMEOUT_SEC) {
            /* Soft kill: force blocking I/O to fail. 
             * This unblocks the core thread and triggers normal cleanup. */
            shutdown(ctx->client_fd, SHUT_RDWR);
        }
    }
}

static void* wolf_core_thread(void* arg) {
    WolfCoreArgs* args = (WolfCoreArgs*)arg;
    WolfCore*     core = args->core;

    /* Initialize O(1) context free-list for this thread (Fix 3) */
    wolf_ctx_freelist_init();

    /* Set up sigaltstack for this specific thread */
    stack_t altstack;
    altstack.ss_sp = malloc(SIGSTKSZ);
    if (altstack.ss_sp) {
        altstack.ss_size = SIGSTKSZ;
        altstack.ss_flags = 0;
        sigaltstack(&altstack, NULL);
    }

    /* Pin to core */
    wolf_pin_to_core(core->core_id);

    fprintf(stderr, "[WOLF-ENGINE] Core %d started (tid=%lu)\n",
            core->core_id, (unsigned long)pthread_self());

    /* Add server fd to sentinel for edge-triggered accept */
    wolf_sentinel_add(core->sentinel, core->server_fd, NULL, NULL);

    /* Signal to sysmon that this thread is fully initialized */
    __atomic_store_n(&core->ready, 1, __ATOMIC_RELEASE);

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
            wolf_arena_reset(arena);  /* frees if overflow */
            arena = NULL;
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

        /* GET /health Observability Bypass */
        if (ctx->method && strcmp(ctx->method, "GET") == 0 && ctx->path && strcmp(ctx->path, "/health") == 0) {
            char health_buf[4096];
            extern int wolf_get_active_requests(void);
            int active = wolf_get_active_requests();
            
            /* Dump lock-free metrics + active requests */
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
            goto send_and_cleanup;
        }



        /* WebSocket upgrade */
        if (ctx->is_websocket) {
            wolf_engine_ws_handshake(ctx);

            /* Transfer fd ownership to the legacy WS poller (wolf_runtime.c).
             * The poller manages the connection lifecycle from here.
             * We must NOT close client_fd — the poller owns it now.
             * We MUST free our engine ctx (its arena is no longer needed;
             * the poller uses its own http_contexts[] storage). */
            wolf_engine_register_ws_fd(ctx->client_fd,
                                       ctx->method, ctx->path, ctx->query,
                                       ctx->ws_key,
                                       ctx->client_ip);

            /* Release engine ctx but do NOT close client_fd */
            ctx->client_fd = -1;  /* prevent wolf_core_free_ctx from closing it */
            wolf_core_free_ctx(ctx);
            continue;
        }

        /* Set thread-local request context (legacy API compatibility) */
        int64_t ctx_id = (int64_t)(ctx - wolf_core_ctxs);
        ctx->request_id = ctx_id;
        ctx->oom_triggered = 0;
        ctx->arena_used = ctx->arena ? ctx->arena->total_allocated + ctx->arena->pos : 0;
        ctx->arena_cap = WOLF_MAX_REQUEST_MEMORY;
        wolf_set_current_context((void*)(intptr_t)ctx_id, (void*)(intptr_t)ctx_id);

        /* Test crash endpoint for SIGSEGV handling */
        if (ctx->method && strcmp(ctx->method, "GET") == 0 && ctx->path && strcmp(ctx->path, "/crash") == 0) {
            __atomic_store_n(&wolf_active_ctx, ctx, __ATOMIC_RELEASE);
            fprintf(stderr, "Triggering deliberate crash on core %d for test...\n", core->core_id);
            int* ptr = NULL;
            *ptr = 42;
        }

        /* Call Wolf HTTP handler */
        __atomic_fetch_add(&core->requests_active, 1, __ATOMIC_RELAXED);
        if (args->http_handler) {
            /* Mark active for watchdog (Release-Acquire sync) */
            __atomic_store_n(&wolf_active_ctx, ctx, __ATOMIC_RELEASE);

            wolf_req_arena_init();
            wolf_closure_t* closure = (wolf_closure_t*)args->http_handler;
            typedef void* (*wolf_closure_fn_t)(void* env, int64_t req_id, int64_t res_id);
            wolf_closure_fn_t fn = (wolf_closure_fn_t)closure->fn;

            if (!wolf_closure_valid(closure)) {
                fprintf(stderr, "[WOLF-ENGINE] CORRUPTED closure on core %d: "
                        "closure=%p magic=0x%08X fn=%p — dropping request\n",
                        core->core_id, (void*)closure,
                        closure ? closure->magic : 0,
                        (void*)(uintptr_t)fn);
                fflush(stderr);
                ctx->status_code = 500;
                __atomic_store_n(&wolf_active_ctx, NULL, __ATOMIC_RELEASE);
                goto send_and_cleanup;
            }

            ctx->oom_triggered = 0;
            if (setjmp(ctx->oom_jump) == 0) {
                fn(closure->env, ctx_id, ctx_id);
            } else {
                fprintf(stderr, "[WOLF-ENGINE] OOM exception caught on core %d, returning 500\n", core->core_id);
                ctx->status_code = 500;
                ctx->res_body = "500 Internal Server Error (OOM)";
            }

            /* Clear active marker */
            __atomic_store_n(&wolf_active_ctx, NULL, __ATOMIC_RELEASE);

            if (wolf_req_oom_check()) {
                ctx->status_code = 503;
                if (!ctx->res_body) ctx->res_body = "Service Unavailable";
                wolf_req_oom_clear();
            }
        }

send_and_cleanup:
        __atomic_fetch_sub(&core->requests_active, 1, __ATOMIC_RELAXED);

        /* Send response */
        int send_ok = wolf_engine_send_response(ctx);

        /* Stats (only increment if client didn't timeout/disconnect) */
        if (send_ok == 0) {
            __atomic_fetch_add(&core->requests_total, 1, __ATOMIC_RELAXED);
            __atomic_fetch_add(&core->bytes_in,  bytes,                              __ATOMIC_RELAXED);
            __atomic_fetch_add(&core->bytes_out, ctx->res_body ? strlen(ctx->res_body) : 0, __ATOMIC_RELAXED);
        }

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

    wolf_safe_print("\n[WOLF-CRASH] Caught SIGSEGV (Segmentation Fault)\n");

    /* Read thread-local context safely */
    WolfConnCtx* ctx = __atomic_load_n(&wolf_active_ctx, __ATOMIC_ACQUIRE);
    if (ctx) {
        wolf_safe_print("  -> Request ID:   ");
        wolf_safe_print_int((long long)ctx->request_id);
        wolf_safe_print("\n  -> Endpoint:     ");
        wolf_safe_print(ctx->path ? ctx->path : "unknown");
        wolf_safe_print("\n  -> Arena Usage:  ");
        wolf_safe_print_int((long long)ctx->arena_used);
        wolf_safe_print(" bytes (cap: ");
        wolf_safe_print_int((long long)ctx->arena_cap);
        wolf_safe_print(")\n  -> OOM Triggered: ");
        wolf_safe_print(ctx->oom_triggered ? "true\n" : "false\n");
    } else {
        wolf_safe_print("  -> Context:      No active request context\n");
    }

    wolf_safe_print("[WOLF-CRASH] Re-raising signal to supervisor...\n");

    /* Re-raise to ensure proper exit status and core dump */
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
        /* workers = 0 (auto) — use 4x physical cores for transitional oversubscription */
        core_count = max_recommended;
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
        core->sentinel   = wolf_sentinel_create(i);
        core->arena_pool = wolf_arena_pool_create(i);
        engine->cores[i] = core;
    }

    return engine;
}

int wolf_engine_start(WolfEngine* engine, wolf_http_handler_t handler, wolf_ws_handler_t ws_handler) {
    /* Install signal handlers */
    struct sigaction sa;
    memset(&sa, 0, sizeof(sa));
    sa.sa_handler = wolf_engine_signal_handler;
    sigemptyset(&sa.sa_mask);
    sigaction(SIGTERM, &sa, NULL);
    sigaction(SIGINT,  &sa, NULL);
    signal(SIGPIPE, SIG_IGN);
    
    /* Sysmon watchdog handler */
    struct sigaction sw;
    memset(&sw, 0, sizeof(sw));
    sw.sa_handler = wolf_engine_watchdog_handler;
    sigfillset(&sw.sa_mask);
    sw.sa_flags = SA_RESTART;
    sigaction(SIGURG, &sw, NULL);

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

    /* Main thread acts as sysmon, sending SIGURG preemption signals to cores every 10ms */
    while (!wolf_engine_shutdown_flag) {
        usleep(10000);
        for (int i = 0; i < engine->core_count; i++) {
            WolfCore* core = engine->cores[i];
            if (core && core->thread && __atomic_load_n(&core->ready, __ATOMIC_ACQUIRE)) {
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
    if (handler_ptr && (uintptr_t)handler_ptr >= 0x600000000000UL) {
        fprintf(stderr, "[Wolf] FATAL: handler_ptr %p looks like a heap address. "
                "Closure must not be arena-allocated.\n", handler_ptr);
        exit(1);
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

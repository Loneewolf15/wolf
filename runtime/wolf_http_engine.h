/*
 * wolf_http_engine.h — Phase 1 Thread-Per-Core HTTP Engine
 *
 * Architecture:
 *   - N threads where N = CPU cores (SO_REUSEPORT per thread)
 *   - Each thread owns its own epoll/kqueue instance
 *   - Each thread owns its own per-request arena pool
 *   - No cross-thread queue contention in the hot path
 *   - Graceful shutdown via atomic flag + drain condition
 *
 * Phase 2 placeholder: WolfScheduler (WTask/WExecutor/WThread)
 * stubs declared here, implemented in wolf_scheduler.c
 */

#ifndef WOLF_HTTP_ENGINE_H
#define WOLF_HTTP_ENGINE_H

#include <stdint.h>
#include <stddef.h>
#include <stdatomic.h>
#include "wolf_thread_compat.h"   /* pthread + Windows portability shims */
#include "wolf_config_runtime.h"
#include "wolf_queue.h"            /* Lock-free SPSC ring buffer — Wolf Engine V3 */

/* ── Platform capability detection ──────────────────────────────────────── */
#if defined(_WIN32)
#  define WOLF_HAS_IOCP 1
#endif

/* ================================================================
 * Sentinel — cross-platform I/O abstraction
 * Phase 1: epoll (Linux) / kqueue (macOS) / poll (fallback)
 * Phase 2: io_uring (Linux SQPOLL), IOCP (Windows)
 * ================================================================ */

typedef enum {
    WOLF_IO_EPOLL,    /* Linux */
    WOLF_IO_KQUEUE,   /* macOS / BSD */
    WOLF_IO_POLL,     /* fallback */
    WOLF_IO_IOURING,  /* Phase 2 */
    WOLF_IO_IOCP,     /* Phase 2 Windows */
} WolfIOBackend;

typedef void (*wolf_io_callback_t)(int fd, void* ctx, int events);

typedef struct WolfSentinel {
    WolfIOBackend backend;
    int           poll_fd;    /* epoll fd or kqueue fd */
    int           core_id;
    void*         uring;      /* WolfURing*  — io_uring (Linux)  */
    void*         iocp;       /* WolfIOCP*   — IOCP    (Windows) */
} WolfSentinel;

WolfSentinel* wolf_sentinel_create(int core_id);
int           wolf_sentinel_add(WolfSentinel* s, int fd, wolf_io_callback_t cb, void* ctx);
int           wolf_sentinel_remove(WolfSentinel* s, int fd);
int           wolf_sentinel_poll(WolfSentinel* s, int timeout_ms);
void          wolf_sentinel_destroy(WolfSentinel* s);

/* ================================================================
 * Per-Core Arena Pool
 * Pre-allocated slabs of request arenas, one pool per core.
 * Avoids malloc() in the hot path entirely for short-lived requests.
 * ================================================================ */

#ifndef WOLF_ARENA_POOL_SIZE
#  define WOLF_ARENA_POOL_SIZE 256   /* arenas pre-allocated per core */
#endif

#ifndef WOLF_ARENA_SLAB_SIZE
#  define WOLF_ARENA_SLAB_SIZE (64 * 1024)  /* 64KB per arena slab */
#endif

#ifndef WOLF_MAX_REQUEST_MEMORY
#  define WOLF_MAX_REQUEST_MEMORY (16 * 1024 * 1024) /* 16MB per request limit */
#endif

typedef struct WolfArenaBlock {
    char* slab;
    struct WolfArenaBlock* next;
} WolfArenaBlock;

typedef struct WolfArena {
    char*  active_slab;
    size_t pos;
    size_t cap;
    
    char*  base_slab;
    size_t base_cap;
    size_t total_allocated;

    int    in_use;
    int    is_overflow;  /* 1 = heap-allocated fallback arena struct, must be freed on pool destroy/reset if applicable */

    /* Linked list of geometric block-growth fallbacks */
    WolfArenaBlock* fallback_blocks;
    
    volatile int refcount; /* W1 Fix: track detached concurrent spawn tasks */
    int pool_index; /* Index within WolfArenaPool for io_uring registered buffers */
} WolfArena;

typedef struct WolfArenaPool {
    WolfArena arenas[WOLF_ARENA_POOL_SIZE];
    int       count;
    int       core_id;
} WolfArenaPool;

WolfArenaPool* wolf_arena_pool_create(int core_id);
WolfArena*     wolf_arena_acquire(WolfArenaPool* pool);
void*          wolf_arena_alloc(WolfArena* arena, size_t size);
char*          wolf_arena_strdup(WolfArena* arena, const char* s);
void           wolf_arena_ref(WolfArena* arena);     /* inc refcount */
void           wolf_arena_unref(WolfArena* arena);   /* dec refcount, resets if 0 */
void           wolf_arena_reset(WolfArena* arena);   /* O(1) — just resets pos */
void           wolf_arena_pool_destroy(WolfArenaPool* pool);

/* ================================================================
 * WCore — one per CPU core
 * Owns: server socket (SO_REUSEPORT), sentinel, arena pool,
 *       and its own independent HTTP context table.
 * ================================================================ */

#define WOLF_CORE_MAX_CONNECTIONS 2048

typedef void (*wolf_http_handler_t)(int64_t req_id, int64_t res_id);
typedef void (*wolf_ws_handler_t)(int64_t req_id, const char* message);

/* ================================================================
 * Per-core worker→poller completion: V3 SPSC Matrix
 *
 * Wolf Engine V3 replaces the MPSC complete_ring (which had a data race
 * under load — multiple workers racing to increment tail) with a
 * mathematically race-free SPSC (Single-Producer Single-Consumer) matrix.
 *
 * Matrix layout: complete_rings[core_id][worker_id]
 *   - PRODUCER: worker thread `worker_id` (only one thread writes per ring)
 *   - CONSUMER: I/O poller thread `core_id` (only one thread reads per ring)
 *
 * The old WolfCoreCompleteRing is kept for the non-io_uring fallback path
 * (epoll/kqueue) which serializes workers differently.
 * ================================================================ */

#define WOLF_CORE_COMPLETE_SIZE 512          /* must be power of 2 */
#define WOLF_CORE_COMPLETE_MASK (WOLF_CORE_COMPLETE_SIZE - 1)

typedef struct {
    void* ctx;       /* WolfConnCtx* — void* avoids circular header dep */
    char* out_buf;
    int   out_len;
} WolfCoreCompletion;

typedef struct {
    WolfCoreCompletion entries[WOLF_CORE_COMPLETE_SIZE];
    _Atomic int        head __attribute__((aligned(64)));
    _Atomic int        tail __attribute__((aligned(64)));
    int                notify_fd; /* eventfd — worker writes to wake poller */
    int                worker_id; /* ID of the worker owning this ring */
} WolfCoreCompleteRing;

/*
 * V3: Per-worker SPSC rings for io_uring path
 * Each worker has its own ring → core. Zero contention guaranteed.
 * Indexed as spsc_rings[worker_id] within each WolfCore.
 */
#ifndef WOLF_MAX_WORKERS
#  define WOLF_MAX_WORKERS 256   /* max workers for SPSC matrix */
#endif

typedef struct WolfCore {
    int             core_id;
    int             server_fd;       /* SO_REUSEPORT socket for this core */
    WolfSentinel*   sentinel;
    WolfArenaPool*  arena_pool;
    pthread_t       thread;
    volatile int    ready;           /* set to 1 when thread enters its main loop */
    void*           args;            /* pointer to original WolfCoreArgs */

    /* Per-core HTTP context table — no mutex needed, single thread owns it */
    void*           contexts;        /* wolf_http_context_t array */
    int             context_count;

    /* Stats */
    volatile int64_t requests_total;
    volatile int64_t requests_active;
    volatile int64_t bytes_in;
    volatile int64_t bytes_out;

    /* Handler pointers (set once at startup, read-only after) */
    wolf_http_handler_t http_handler;
    wolf_ws_handler_t   ws_handler;

    /* Worker → poller completion handoff ring (legacy MPSC, used on epoll/kqueue) */
    WolfCoreCompleteRing complete_ring;

    /*
     * V3: per-worker SPSC rings — io_uring fast path.
     * spsc_submit_rings[j]: Core (Producer) → Worker j (Consumer)
     * spsc_complete_rings[j]: Worker j (Producer) → Core (Consumer)
     * Zero locks, zero contention.
     */
    wolf_spsc_t spsc_submit_rings[WOLF_MAX_WORKERS];
    wolf_spsc_t spsc_complete_rings[WOLF_MAX_WORKERS];
    int         spsc_worker_count;      /* number of workers actually registered to this core */
    int         spsc_next_worker;       /* round-robin assignment counter for submit_rings */

    /* eventfd per worker: worker writes here to wake poller after spsc_push */
    int         spsc_notify_fds[WOLF_MAX_WORKERS];

    /* Capability flags — set once at startup, read-only after */
    int use_send_zc;   /* 1 = use IORING_OP_SEND_ZC (kernel >= 6.0 + liburing >= 2.3) */
    int use_spsc;      /* 1 = use V3 SPSC matrix (io_uring path), 0 = legacy ring */

    /* Event-driven epoll/kqueue fds (set in wolf_core_thread, -1 elsewhere) */
    int epoll_fd;       /* Linux epoll fd used by the event-driven loop */
    int kq_notify_rfd;  /* macOS: pipe read-end for worker→poller wakeup */
} WolfCore;

/* ================================================================
 * WolfEngine — the top-level server
 * ================================================================ */

typedef struct WolfEngine {
    WolfCore** cores;
    int        core_count;
    int        port;
    volatile int shutdown_requested;

    /* Benchmark mode */
    int bench_mode;

    /* TLS / kTLS Integration */
    int   use_ktls;
    void* ssl_ctx;
} WolfEngine;

WolfEngine* wolf_engine_create(int port, int core_count);
int         wolf_engine_enable_tls(WolfEngine* engine, const char* cert_path, const char* key_path);
int         wolf_engine_start(WolfEngine* engine, wolf_http_handler_t handler, wolf_ws_handler_t ws_handler);
void        wolf_engine_shutdown(WolfEngine* engine);
void        wolf_engine_stats(WolfEngine* engine);  /* prints per-core stats */
void        wolf_engine_destroy(WolfEngine* engine);

/* ================================================================
 * Phase 2 Placeholders — WolfScheduler (WTask/WExecutor/WThread)
 * These types are declared here but implemented in wolf_scheduler.c
 * The HTTP engine is designed to slot these in without API changes.
 * ================================================================ */

typedef enum {
    WTASK_STATE_READY,
    WTASK_STATE_RUNNING,
    WTASK_STATE_WAITING_IO,
    WTASK_STATE_COMPLETE,
} WTaskState;

typedef struct WTask {
    WTaskState   state;
    int          core_id;       /* pinned core — no migration in HTTP mode */
    WolfArena*   arena;         /* task's arena — freed on completion */
    void*        stack;         /* arena-allocated stack (Phase 2) */
    size_t       stack_size;
    int64_t      req_id;
    int64_t      res_id;
    /* Continuation pointer for cooperative scheduling (Phase 2) */
    void*        continuation;
} WTask;

/* Phase 2 stubs — currently implemented as direct function calls */
WTask* wtask_create(WolfCore* core, int64_t req_id);
void   wtask_yield(WTask* task);    /* yield back to executor event loop */
void   wtask_complete(WTask* task); /* free arena, return slot */

int wolf_engine_socket_set_nonblocking(int fd);
int wolf_engine_socket_set_tcp_nodelay(int fd);

/* SIMD HTTP Parser */
void wolf_engine_parse_request_simd(void* ctx_ptr, char* raw, size_t len);
int wolf_header_compare_ct(const char* a, const char* b);

#endif /* WOLF_HTTP_ENGINE_H */

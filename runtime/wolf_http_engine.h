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
#include <pthread.h>
#include "wolf_config_runtime.h"

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
    void*         uring;      /* WolfURing* for io_uring (void* to avoid header leak) */
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
#  define WOLF_ARENA_POOL_SIZE 128   /* arenas pre-allocated per core */
#endif

#ifndef WOLF_ARENA_SLAB_SIZE
#  define WOLF_ARENA_SLAB_SIZE (64 * 1024)  /* 64KB per arena slab */
#endif

typedef struct WolfArena {
    char*  slab;
    size_t pos;
    size_t cap;
    int    in_use;
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

typedef struct WolfCore {
    int             core_id;
    int             server_fd;       /* SO_REUSEPORT socket for this core */
    WolfSentinel*   sentinel;
    WolfArenaPool*  arena_pool;
    pthread_t       thread;

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
} WolfEngine;

WolfEngine* wolf_engine_create(int port, int core_count);
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

#endif /* WOLF_HTTP_ENGINE_H */

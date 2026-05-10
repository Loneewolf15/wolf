/*
 * wolf_config_runtime.h
 * Add an #include of this file at the TOP of wolf_runtime.c, before any
 * other #define blocks. It sets up all the compile-time constants that
 * wolf.config bakes in via -D flags, with safe defaults for every value
 * so the runtime still compiles when built without the config system.
 *
 * Usage in wolf_runtime.c:
 *   #include "wolf_config_runtime.h"   ← first include, before wolf_runtime.h
 *   #include "wolf_runtime.h"
 *
 * Build target selection (set via wolf build → -D flag):
 *   -DWOLF_BUILD_TARGET_API    → full HTTP server, Thread-Per-Core, arena pool
 *   -DWOLF_BUILD_TARGET_SCRIPT → lightweight runtime, single arena, no HTTP engine
 *   -DWOLF_BUILD_TARGET_MCU    → no OS primitives, static heap, cooperative loop
 *
 * If none is set, WOLF_BUILD_TARGET_SCRIPT is the safe default.
 */

#ifndef WOLF_CONFIG_RUNTIME_H
#define WOLF_CONFIG_RUNTIME_H

/* ================================================================
 * Build Target — default to Script if unset (safe, lightweight)
 * ================================================================ */

#if !defined(WOLF_BUILD_TARGET_API) && \
    !defined(WOLF_BUILD_TARGET_SCRIPT) && \
    !defined(WOLF_BUILD_TARGET_MCU)
#  define WOLF_BUILD_TARGET_SCRIPT
#endif

/* ================================================================
 * Derived Capability Flags
 * These are what the rest of the runtime checks — never check the
 * WOLF_BUILD_TARGET_* defines directly; use these instead.
 * ================================================================ */

#if defined(WOLF_BUILD_TARGET_API)
   /* Full production server stack */
#  define WOLF_HAS_HTTP_ENGINE    1
#  define WOLF_HAS_THREAD_POOL    1
#  define WOLF_HAS_ARENA_POOL     1   /* per-core arena pool */
#  define WOLF_HAS_DB_POOL        1
#  define WOLF_HAS_SIGNALS        1
#  define WOLF_HAS_EPOLL          1   /* epoll/kqueue/io_uring I/O */

#elif defined(WOLF_BUILD_TARGET_SCRIPT)
   /* Lightweight runtime — single-threaded by default */
#  define WOLF_HAS_HTTP_ENGINE    0   /* HTTP engine NOT started automatically */
#  define WOLF_HAS_THREAD_POOL    0   /* explicit opt-in required via wolf_parallel() */
#  define WOLF_HAS_ARENA_POOL     1   /* single arena (not per-core) */
#  define WOLF_HAS_DB_POOL        1   /* scripts can query databases */
#  define WOLF_HAS_SIGNALS        1
#  define WOLF_HAS_EPOLL          0

#elif defined(WOLF_BUILD_TARGET_MCU)
   /* Bare-metal — no OS, no pthreads, no malloc, no epoll */
#  define WOLF_HAS_HTTP_ENGINE    0
#  define WOLF_HAS_THREAD_POOL    0
#  define WOLF_HAS_ARENA_POOL     0   /* replaced by static slab below */
#  define WOLF_HAS_DB_POOL        0
#  define WOLF_HAS_SIGNALS        0
#  define WOLF_HAS_EPOLL          0
#  define WOLF_MCU_MODE           1
#endif

/* ================================================================
 * Worker thread count (baked from wolf.config [server] workers)
 * 0 = auto-detect at runtime via sysconf / sched_getaffinity
 * ================================================================ */
#ifndef WOLF_WORKER_THREADS
#  define WOLF_WORKER_THREADS 0
#endif

/* ================================================================
 * MCU Static Allocator
 * Replaces malloc/arena on bare-metal targets.
 * The slab lives in BSS (zero-initialised by the MCU startup).
 * wolf_mcu_heap_reset() is called once per cooperative tick to
 * reclaim temporary allocations (equivalent to arena_reset).
 * ================================================================ */
#if defined(WOLF_BUILD_TARGET_MCU)

#ifndef WOLF_MCU_STATIC_HEAP_KB
#  define WOLF_MCU_STATIC_HEAP_KB 64  /* 64KB default; tune to your SRAM size */
#endif

#include <stdint.h>
#include <string.h>

static uint8_t wolf_mcu_heap[WOLF_MCU_STATIC_HEAP_KB * 1024];
static size_t  wolf_mcu_heap_pos = 0;

/* wolf_mcu_panic — bare-metal hard fault. No recovery possible.
 * On Cortex-M this triggers a HardFault via an invalid instruction. */
static inline void wolf_mcu_panic(const char* msg) {
    (void)msg;  /* msg available for debugger watchpoint */
    while (1) { __asm volatile("nop"); }
}

/* wolf_mcu_alloc — bump allocator from static slab. O(1), no fragmentation.
 * Alignment: always 8-byte so doubles and pointers are safe on all MCUs. */
static inline void* wolf_mcu_alloc(size_t size) {
    size = (size + 7) & ~(size_t)7;  /* 8-byte align */
    if (wolf_mcu_heap_pos + size > sizeof(wolf_mcu_heap)) {
        wolf_mcu_panic("wolf_mcu_alloc: OOM");
        return (void*)0;  /* unreachable, silences compiler */
    }
    void* p = wolf_mcu_heap + wolf_mcu_heap_pos;
    wolf_mcu_heap_pos += size;
    memset(p, 0, size);
    return p;
}

/* wolf_mcu_heap_reset — O(1) reclaim: just reset the position pointer.
 * Call once per cooperative event loop tick, after all work is done. */
static inline void wolf_mcu_heap_reset(void) {
    wolf_mcu_heap_pos = 0;
}

/* wolf_mcu_strdup — strdup from static slab (no system strdup on bare metal) */
static inline char* wolf_mcu_strdup(const char* s) {
    if (!s) return (char*)0;
    size_t len = 0;
    while (s[len]) len++;
    char* p = (char*)wolf_mcu_alloc(len + 1);
    if (p) { memcpy(p, s, len); p[len] = '\0'; }
    return p;
}

/* Redirect standard allocator calls to MCU slab in MCU builds.
 * This catches any accidental malloc() / strdup() calls in shared code. */
#define malloc(n)       wolf_mcu_alloc(n)
#define calloc(n, s)    wolf_mcu_alloc((n) * (s))
#define free(p)         ((void)(p))  /* no-op: reset happens at tick boundary */
#define strdup(s)       wolf_mcu_strdup(s)

#endif /* WOLF_BUILD_TARGET_MCU */

/* ================================================================
 * Database pool (used by API and Script; stripped in MCU builds)
 * ================================================================ */
#if WOLF_HAS_DB_POOL

#ifndef WOLF_DB_POOL_SIZE
#  define WOLF_DB_POOL_SIZE 10
#endif

#ifndef WOLF_DB_POOL_MIN_IDLE
#  define WOLF_DB_POOL_MIN_IDLE 2
#endif

#ifndef WOLF_DB_POOL_TIMEOUT
#  define WOLF_DB_POOL_TIMEOUT 30
#endif

#ifndef WOLF_DB_MAX_RETRIES
#  define WOLF_DB_MAX_RETRIES 3
#endif

/* ---- DB credentials (baked at compile time, never in source) ---- */
#ifndef WOLF_DB_HOST
#  define WOLF_DB_HOST "localhost"
#endif
#ifndef WOLF_DB_PORT
#  define WOLF_DB_PORT 3306
#endif
#ifndef WOLF_DB_NAME
#  define WOLF_DB_NAME ""
#endif
#ifndef WOLF_DB_USER
#  define WOLF_DB_USER ""
#endif
#ifndef WOLF_DB_PASS
#  define WOLF_DB_PASS ""
#endif

#endif /* WOLF_HAS_DB_POOL */

/* ================================================================
 * Server limits (API mode only; no-ops in Script/MCU)
 * ================================================================ */
#ifndef WOLF_MAX_CONCURRENT_REQUESTS
#  define WOLF_MAX_CONCURRENT_REQUESTS 1024
#endif
#ifndef WOLF_MAX_REQUEST_SIZE
#  define WOLF_MAX_REQUEST_SIZE 65536
#endif
#ifndef WOLF_MAX_UPLOADS
#  define WOLF_MAX_UPLOADS 8
#endif

/* ================================================================
 * Mail Configuration
 * ================================================================ */
#ifndef WOLF_MAIL_FROM_EMAIL
#  define WOLF_MAIL_FROM_EMAIL ""
#endif
#ifndef WOLF_MAIL_HOST
#  define WOLF_MAIL_HOST ""
#endif

/* ================================================================
 * App environment
 * ================================================================ */
#ifndef WOLF_APP_ENV
#  define WOLF_APP_ENV "development"
#endif
#ifndef WOLF_APP_DEBUG
#  define WOLF_APP_DEBUG 0
#endif

/* ================================================================
 * Derived helpers
 * ================================================================ */

/* wolf_is_production() — 1 in production builds. Gate expensive debug logging. */
#define wolf_is_production() (strcmp(WOLF_APP_ENV, "production") == 0)

/* wolf_is_debug() — 1 when APP_DEBUG=true */
#define wolf_is_debug() (WOLF_APP_DEBUG == 1)

/* wolf_is_mcu() — 1 when compiling for bare-metal */
#if defined(WOLF_MCU_MODE)
#  define wolf_is_mcu() 1
#else
#  define wolf_is_mcu() 0
#endif

#endif /* WOLF_CONFIG_RUNTIME_H */
/*
 * wolf_timewheel.h — 3-Level Hierarchical Time-Wheel
 *
 * Provides O(1) connection timeout tracking and Slowloris eviction for the
 * Wolf HTTP Engine. Each WolfCore owns one time-wheel instance; the wheel
 * is ticked by the I/O poller thread after each epoll_wait/kevent cycle.
 *
 * Wheel layout:
 *   Level 0: 256 slots × 1ms  =  256ms range
 *   Level 1:  64 slots × 256ms = 16.384s range
 *   Level 2:  64 slots × 16.384s ≈ 17.5 min range
 *
 * Thread safety: NOT thread-safe. All operations must be called from the
 * owning I/O poller thread. No locking required (correct by design).
 */

#ifndef WOLF_TIMEWHEEL_H
#define WOLF_TIMEWHEEL_H

#include <stdint.h>
#include <stddef.h>

/* ── Wheel dimensions ─────────────────────────────────────────────────────── */
#define WOLF_TW_L0_SLOTS   256   /* Level 0: 256 × 1ms   = 256ms             */
#define WOLF_TW_L1_SLOTS    64   /* Level 1: 64 × 256ms  = 16.384s           */
#define WOLF_TW_L2_SLOTS    64   /* Level 2: 64 × 16.384s ≈ 17.5 min        */

/* Default connection idle timeout: 10 seconds (from accept to first request) */
#ifndef WOLF_TIMEWHEEL_TIMEOUT_MS
#  define WOLF_TIMEWHEEL_TIMEOUT_MS 10000
#endif

/* Max FD value tracked. Must be >= WOLF_MAX_FD from wolf_http_engine.c */
#ifndef WOLF_TW_MAX_FD
#  define WOLF_TW_MAX_FD 65536
#endif

/* ── Per-entry node stored in wheel slot linked list ─────────────────────── */
typedef struct wolf_tw_node_t {
    int                   fd;           /* the connection's file descriptor   */
    uint64_t              deadline_ms;  /* monotonic ms when this conn expires */
    struct wolf_tw_node_t *next;        /* intrusive linked-list per slot      */
    uint8_t               in_wheel;     /* 1 if node is currently linked in wheel */
} wolf_tw_node_t;

/* ── The time-wheel itself ────────────────────────────────────────────────── */
typedef struct wolf_timewheel_t {
    /* Slot heads — intrusive singly-linked lists */
    wolf_tw_node_t *l0[WOLF_TW_L0_SLOTS];
    wolf_tw_node_t *l1[WOLF_TW_L1_SLOTS];
    wolf_tw_node_t *l2[WOLF_TW_L2_SLOTS];

    /* Current position pointers */
    uint32_t        l0_idx;   /* 0..255  */
    uint32_t        l1_idx;   /* 0..63   */
    uint32_t        l2_idx;   /* 0..63   */

    /* Monotonic timestamp when the wheel was last ticked (ms) */
    uint64_t        last_tick_ms;

    /* Node pool: pre-allocated to avoid malloc in accept hot-path            */
    wolf_tw_node_t  node_pool[WOLF_TW_MAX_FD];

    /* Registered bitmap: fd_registered[fd] = 1 if fd is in the wheel.
     * Prevents double-removal and stale-fd close when the connection closes
     * normally before the wheel evicts it. */
    uint8_t         fd_registered[WOLF_TW_MAX_FD];
} wolf_timewheel_t;

/* ── Monotonic clock helper (ms) ─────────────────────────────────────────── */
static inline uint64_t wolf_monotonic_ms(void) {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (uint64_t)ts.tv_sec * 1000ULL + (uint64_t)(ts.tv_nsec / 1000000ULL);
}

/* ── Public API ───────────────────────────────────────────────────────────── */

/* Allocate and initialize a new time-wheel. Returns NULL on OOM. */
wolf_timewheel_t* wolf_timewheel_create(void);

/* Destroy the time-wheel. Does NOT close any open FDs — caller responsible. */
void wolf_timewheel_destroy(wolf_timewheel_t *tw);

/*
 * Register a new connection with the wheel.
 * deadline_ms: absolute monotonic millisecond when connection should be evicted.
 * Typically: wolf_monotonic_ms() + WOLF_TIMEWHEEL_TIMEOUT_MS
 * Safe to call multiple times for the same fd (re-arms the timer).
 */
void wolf_timewheel_add(wolf_timewheel_t *tw, int fd, uint64_t deadline_ms);

/*
 * Remove a connection from the wheel without evicting it.
 * Call this whenever a connection closes normally (before the wheel fires).
 * Safe to call on an fd that was never added (no-op).
 */
void wolf_timewheel_remove(wolf_timewheel_t *tw, int fd);

/*
 * Advance the wheel by real elapsed time. Evicts all timed-out connections.
 *
 * core_ptr: opaque pointer to the WolfCore that owns this wheel.
 *           Used to call epoll_ctl(DEL) and wolf_core_free_ctx on evicted FDs.
 * evict_fn: called for each evicted fd with (fd, core_ptr).
 *
 * Returns: number of connections evicted this tick.
 */
int wolf_timewheel_tick(wolf_timewheel_t *tw,
                        void *core_ptr,
                        void (*evict_fn)(int fd, void *core_ptr));

#endif /* WOLF_TIMEWHEEL_H */

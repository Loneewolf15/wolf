/*
 * wolf_queue.h — Lock-Free SPSC Ring Buffer for Wolf Engine V3
 *
 * Architecture:
 *   Single-Producer, Single-Consumer (SPSC) ring buffer.
 *   Zero mutex, zero spinlock. Only memory barriers (acquire/release).
 *   Head and Tail on separate 64-byte cache lines to eliminate false sharing.
 *
 * Safety Guarantee:
 *   Mathematically race-free when one thread is the producer and exactly
 *   one different thread is the consumer. For M×N engine matrix:
 *     - Core `i` is the ONLY producer for complete_ring[i][j]
 *     - Worker `j` is the ONLY consumer for complete_ring[i][j]
 *     - Worker `j` is the ONLY producer for submit_ring[i][j]
 *     - Core `i` is the ONLY consumer for submit_ring[i][j]
 *
 * References:
 *   - "Disruptor" (LMAX Exchange): cache-line padded head/tail
 *   - Cloudflare Pingora: hot/cold pool design
 *   - "Fast Concurrent Queues for x86 Processors" (Morrison & Afek, 2013)
 */

#ifndef WOLF_QUEUE_H
#define WOLF_QUEUE_H

#include <stdatomic.h>
#include <stddef.h>
#include <string.h>

/* ================================================================
 * SPSC Ring Buffer
 * Capacity MUST be a power of 2.
 * ================================================================ */

#define WOLF_SPSC_CAPACITY 1024
#define WOLF_SPSC_MASK     (WOLF_SPSC_CAPACITY - 1)

/* Opaque element stored in each ring slot */
typedef struct {
    int   type;      /* 0=empty, 1=HTTP, 2=WS_EVENT, 3=ENGINE_HTTP, 4=COMPLETION */
    int64_t id;      /* req_id or context_id */
    void* ctx;       /* WolfConnCtx* (or payload) */
    char* out_buf;   /* For completions */
    int   out_len;   /* For completions */
    int   worker_id; /* which worker produced/consumed this */
    void (*engine_fn)(void* ctx, int worker_id); /* For ENGINE_HTTP */
} wolf_spsc_entry_t;

/*
 * wolf_spsc_t
 *
 * Layout:
 *   [head | 60 bytes padding] [tail | 60 bytes padding] [data...]
 *
 * Both head and tail are on separate 64-byte cache lines so the
 * producer (writing tail) and consumer (reading head) never bounce
 * the same cache line between cores.
 */
typedef struct {
    /* Consumer reads head, Producer reads head to check fullness */
    _Atomic(unsigned int) head __attribute__((aligned(64)));
    char _pad0[64 - sizeof(_Atomic(unsigned int))];

    /* Producer writes tail, Consumer reads tail to check emptiness */
    _Atomic(unsigned int) tail __attribute__((aligned(64)));
    char _pad1[64 - sizeof(_Atomic(unsigned int))];

    wolf_spsc_entry_t entries[WOLF_SPSC_CAPACITY];
} wolf_spsc_t;

/*
 * wolf_spsc_init — zero-initialize the ring.
 * Called once per ring at engine startup.
 */
static inline void wolf_spsc_init(wolf_spsc_t* q) {
    atomic_store_explicit(&q->head, 0u, memory_order_relaxed);
    atomic_store_explicit(&q->tail, 0u, memory_order_relaxed);
    memset(q->entries, 0, sizeof(q->entries));
}

/*
 * wolf_spsc_push — called by the PRODUCER thread only.
 *
 * Returns 1 on success, 0 if ring is full.
 * Uses release store on tail so the consumer sees the entry
 * contents before seeing the updated tail.
 */
static inline int wolf_spsc_push(wolf_spsc_t* q, const wolf_spsc_entry_t* entry) {
    unsigned int t = atomic_load_explicit(&q->tail, memory_order_relaxed);
    unsigned int h = atomic_load_explicit(&q->head, memory_order_acquire);

    if ((t - h) >= WOLF_SPSC_CAPACITY) {
        return 0; /* ring full */
    }

    q->entries[t & WOLF_SPSC_MASK] = *entry;

    /* Release store: ensures entry data is visible before tail increment */
    atomic_store_explicit(&q->tail, t + 1, memory_order_release);
    return 1;
}

/*
 * wolf_spsc_pop — called by the CONSUMER thread only.
 *
 * Returns 1 on success (fills *out), 0 if ring is empty.
 * Uses acquire load on tail so we see the entry written by producer.
 */
static inline int wolf_spsc_pop(wolf_spsc_t* q, wolf_spsc_entry_t* out) {
    unsigned int h = atomic_load_explicit(&q->head, memory_order_relaxed);
    unsigned int t = atomic_load_explicit(&q->tail, memory_order_acquire);

    if (h == t) {
        return 0; /* ring empty */
    }

    *out = q->entries[h & WOLF_SPSC_MASK];

    /* Release store: allows producer to reclaim this slot */
    atomic_store_explicit(&q->head, h + 1, memory_order_release);
    return 1;
}

/*
 * wolf_spsc_is_empty — non-blocking empty check.
 * Safe to call from the consumer thread.
 */
static inline int wolf_spsc_is_empty(const wolf_spsc_t* q) {
    unsigned int h = atomic_load_explicit(&q->head, memory_order_relaxed);
    unsigned int t = atomic_load_explicit(&q->tail, memory_order_acquire);
    return (h == t);
}

/*
 * wolf_spsc_size — approximate number of entries in the ring.
 * Not atomic relative to push/pop — use for diagnostics only.
 */
static inline unsigned int wolf_spsc_size(const wolf_spsc_t* q) {
    unsigned int t = atomic_load_explicit(&q->tail, memory_order_acquire);
    unsigned int h = atomic_load_explicit(&q->head, memory_order_acquire);
    return t - h;
}

/* ================================================================
 * M×N SPSC Matrix for Wolf Engine V3
 *
 * complete_ring[core_id][worker_id]:
 *   Producer = worker_id (posts completed response)
 *   Consumer = core_id   (sends response to client)
 *
 * This eliminates the MPSC double-free race in wolf_http_engine.c
 * where multiple workers raced to write to the same ring->tail slot.
 * ================================================================ */

#define WOLF_ENGINE_MAX_CORES   64
#define WOLF_ENGINE_MAX_WORKERS 64

/* 
 * wolf_engine_matrix_t
 *
 * Statically allocated to avoid heap fragmentation.
 * Each ring is 512 * sizeof(wolf_spsc_entry_t) = ~16KB.
 * Total matrix: 64 * 64 * 16KB = 64MB.
 * In practice wolf uses 4-8 cores × 4-8 workers = 256KB–4MB.
 *
 * Rings are indexed as [core_id][worker_id].
 */
typedef struct {
    wolf_spsc_t rings[WOLF_ENGINE_MAX_CORES][WOLF_ENGINE_MAX_WORKERS];
    int         core_count;
    int         worker_count;
} wolf_engine_matrix_t;

/*
 * wolf_matrix_init — call once before any worker threads start.
 */
static inline void wolf_matrix_init(wolf_engine_matrix_t* m, int cores, int workers) {
    m->core_count   = cores;
    m->worker_count = workers;
    for (int i = 0; i < cores && i < WOLF_ENGINE_MAX_CORES; i++) {
        for (int j = 0; j < workers && j < WOLF_ENGINE_MAX_WORKERS; j++) {
            wolf_spsc_init(&m->rings[i][j]);
        }
    }
}

/*
 * wolf_matrix_push — called by worker thread `worker_id` to post result
 * destined for core `core_id`.
 *
 * Returns 1 on success, 0 if that specific ring is full.
 */
static inline int wolf_matrix_push(wolf_engine_matrix_t* m,
                                   int core_id, int worker_id,
                                   const wolf_spsc_entry_t* entry) {
    if (core_id   < 0 || core_id   >= m->core_count)   return 0;
    if (worker_id < 0 || worker_id >= m->worker_count)  return 0;
    return wolf_spsc_push(&m->rings[core_id][worker_id], entry);
}

/*
 * wolf_matrix_drain — called by poller thread of `core_id`.
 * Drains ALL N worker rings for this core in one pass.
 *
 * Callback fn is called for each drained entry.
 * Returns total number of entries consumed.
 */
typedef void (*wolf_matrix_drain_cb_t)(const wolf_spsc_entry_t* entry, void* userdata);

static inline int wolf_matrix_drain(wolf_engine_matrix_t* m,
                                    int core_id,
                                    wolf_matrix_drain_cb_t cb,
                                    void* userdata) {
    if (core_id < 0 || core_id >= m->core_count) return 0;

    int total = 0;
    wolf_spsc_entry_t entry;

    for (int j = 0; j < m->worker_count; j++) {
        while (wolf_spsc_pop(&m->rings[core_id][j], &entry)) {
            if (cb) cb(&entry, userdata);
            total++;
        }
    }
    return total;
}

#endif /* WOLF_QUEUE_H */

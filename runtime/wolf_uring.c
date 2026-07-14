#include "wolf_uring.h"

#ifdef WOLF_HAS_IO_URING

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <poll.h>      /* POLLIN — for wolf_uring_poll_fd */

WolfURing* wolf_uring_create(int queue_depth, int sqpoll) {
    WolfURing* ring = (WolfURing*)calloc(1, sizeof(WolfURing));
    
    struct io_uring_params params;
    memset(&params, 0, sizeof(params));
    
    if (sqpoll) {
        params.flags |= IORING_SETUP_SQPOLL;
        params.sq_thread_idle = 2000;
        ring->sqpoll_enabled = 1;
    }
    
    int ret = io_uring_queue_init_params(queue_depth, &ring->ring, &params);
    if (ret < 0) {
        fprintf(stderr, "[WOLF-URING] Failed to initialize io_uring: %s\n", strerror(-ret));
        free(ring);
        return NULL;
    }
    
    return ring;
}

int wolf_uring_submit_accept(WolfURing* r, int server_fd, wolf_uring_cb_t cb, void* ctx, WolfArena* arena) {
    struct io_uring_sqe *sqe = io_uring_get_sqe(&r->ring);
    if (!sqe) return -1;
    
    // Fallback to malloc if arena is NULL (e.g., dedicated global context)
    wolf_uring_req_t* req = arena ? (wolf_uring_req_t*)wolf_arena_alloc(arena, sizeof(wolf_uring_req_t)) 
                                  : (wolf_uring_req_t*)malloc(sizeof(wolf_uring_req_t));
    if (!req) return -1;
    req->fd = server_fd;
    req->cb = cb;
    req->ctx = ctx;
    
    io_uring_prep_multishot_accept(sqe, server_fd, NULL, NULL, 0);
    io_uring_sqe_set_data(sqe, req);
    
    // io_uring_submit removed; caller must batch via wolf_uring_flush()
    return 0;
}

int wolf_uring_submit_recv(WolfURing* r, int client_fd, void* buf, size_t len, wolf_uring_cb_t cb, void* ctx, WolfArena* arena) {
    struct io_uring_sqe *sqe = io_uring_get_sqe(&r->ring);
    if (!sqe) return -1;
    
    wolf_uring_req_t* req = (wolf_uring_req_t*)wolf_arena_alloc(arena, sizeof(wolf_uring_req_t));
    if (!req) return -1;
    req->fd = client_fd;
    req->cb = cb;
    req->ctx = ctx;
    
    io_uring_prep_recv(sqe, client_fd, buf, len, 0);
    io_uring_sqe_set_data(sqe, req);
    
    return 0;
}

int wolf_uring_submit_send(WolfURing* r, int client_fd, const void* buf, size_t len, wolf_uring_cb_t cb, void* ctx, WolfArena* arena) {
    struct io_uring_sqe *sqe = io_uring_get_sqe(&r->ring);
    if (!sqe) return -1;
    
    wolf_uring_req_t* req = (wolf_uring_req_t*)wolf_arena_alloc(arena, sizeof(wolf_uring_req_t));
    if (!req) return -1;
    req->fd = client_fd;
    req->cb = cb;
    req->ctx = ctx;
    
    io_uring_prep_send(sqe, client_fd, buf, len, MSG_WAITALL);
    io_uring_sqe_set_data(sqe, req);
    
    return 0;
}

int wolf_uring_submit_send_zc(WolfURing* r, int client_fd, const void* buf, size_t len,
                              wolf_uring_cb_t cb, void* ctx, WolfArena* arena) {
#ifdef IORING_OP_SEND_ZC
    struct io_uring_sqe *sqe = io_uring_get_sqe(&r->ring);
    if (!sqe) return -1;

    wolf_uring_req_t* req = (wolf_uring_req_t*)wolf_arena_alloc(arena, sizeof(wolf_uring_req_t));
    if (!req) return -1;
    req->fd         = client_fd;
    req->cb         = cb;
    req->ctx        = ctx;
    req->is_send_zc = 1;

    io_uring_prep_send_zc(sqe, client_fd, buf, len, MSG_WAITALL, 0);
    io_uring_sqe_set_data(sqe, req);
    return 0;
#else
    /* Kernel/liburing too old — fall back to regular send */
    return wolf_uring_submit_send(r, client_fd, buf, len, cb, ctx, arena);
#endif
}

/* Probe whether the running kernel supports IORING_OP_SEND_ZC.
 * Called once at engine startup; result cached in WolfCore.use_send_zc. */
int wolf_uring_has_send_zc(WolfURing* r) {
#ifdef IORING_OP_SEND_ZC
    struct io_uring_probe* probe = io_uring_get_probe_ring(&r->ring);
    if (!probe) return 0;
    int supported = io_uring_opcode_supported(probe, IORING_OP_SEND_ZC);
    free(probe);
    return supported;
#else
    (void)r;
    return 0;
#endif
}

int wolf_uring_flush(WolfURing* r) {
    if (!r) return 0;
    return io_uring_submit(&r->ring);
}

/* Register an fd for persistent POLLIN monitoring (multishot poll_add).
 * Each time the fd becomes readable, the callback fires and the poll stays
 * armed — no re-arm needed. Used for the per-core completion notify_fd. */
int wolf_uring_poll_fd(WolfURing* r, int fd, wolf_uring_cb_t cb, void* ctx) {
    if (!r) return -1;
    struct io_uring_sqe* sqe = io_uring_get_sqe(&r->ring);
    if (!sqe) return -1;

    wolf_uring_req_t* req = (wolf_uring_req_t*)malloc(sizeof(wolf_uring_req_t));
    if (!req) return -1;
    req->fd  = fd;
    req->cb  = cb;
    req->ctx = ctx;

    io_uring_prep_poll_add(sqe, fd, POLLIN);
#ifdef IORING_POLL_ADD_MULTI
    io_uring_sqe_set_flags(sqe, IORING_POLL_ADD_MULTI);
#endif
    io_uring_sqe_set_data(sqe, req);
    return 0;
}

/*
 * WOLF_URING_DRAIN_PASSES — maximum CQE drain passes per poll call.
 *
 * After each pass we flush pending SQEs (io_uring_submit). With SQPOLL, the
 * kernel can process those SQEs and inject new CQEs in nanoseconds on loopback.
 * If we loop immediately, we handle the full RECV→SEND→RECV chain in one
 * poll call instead of re-entering the wait on every hop. This cuts round-trip
 * latency without adding syscalls (SQPOLL fast path avoids io_uring_enter when
 * the ring has no pending SQEs to submit).
 *
 * Bounded at 4 passes to prevent starvation of the shutdown-flag check.
 */
#define WOLF_URING_DRAIN_PASSES 4

int wolf_uring_poll(WolfURing* r, int timeout_ms) {
    if (!r) return 0;

    struct io_uring_cqe *cqe;
    int ret;

    /*
     * Wait for at least one CQE. liburing's internal implementation of
     * io_uring_wait_cqe_timeout calls io_uring_peek_cqe first: if a CQE is
     * already in the ring (common under load with SQPOLL), it returns without
     * making a syscall. The timeout only fires when the ring is truly empty.
     */
    if (timeout_ms >= 0) {
        struct __kernel_timespec ts;
        ts.tv_sec  = timeout_ms / 1000;
        ts.tv_nsec = (timeout_ms % 1000) * 1000000LL;
        ret = io_uring_wait_cqe_timeout(&r->ring, &cqe, &ts);
    } else {
        ret = io_uring_wait_cqe(&r->ring, &cqe);
    }

    if (ret < 0) return 0; /* timeout or error */

    /*
     * Drain the ring in snapshot passes.
     *
     * SQPOLL batch-bleed safety: each pass snapshots khead/ktail once.
     * CQEs injected by the SQPOLL kernel thread mid-pass (e.g. a SEND CQE
     * arriving while we process a RECV CQE for the same connection) appear
     * AFTER snap_tail and are therefore NOT counted in this pass's `count`.
     * They will be picked up in the next pass — by which point the arena
     * has already been reset in on_send_complete and a fresh recv req has
     * been allocated. This prevents the double-arena-reset / SIGSEGV that
     * the original io_uring_for_each_cqe loop suffered.
     *
     * Callbacks must NOT set wolf_active_ctx from inside this loop:
     *   - on_accept_complete receives WolfCore* as ctx (not WolfConnCtx*)
     *   - wolf_arena_alloc() type-punches wolf_active_ctx as WolfConnCtx*
     *   - Setting it to a WolfCore* writes arena_used ~1000 bytes past the
     *     end of WolfCore → heap corruption → free() SIGSEGV on every accept
     * Each callback manages wolf_active_ctx itself (on_recv_complete sets it
     * around the Wolf handler call; all other callbacks leave it NULL).
     */
    int total = 0;

    for (int pass = 0; pass < WOLF_URING_DRAIN_PASSES; pass++) {
        unsigned snap_head = io_uring_smp_load_acquire(r->ring.cq.khead);
        unsigned snap_tail = io_uring_smp_load_acquire(r->ring.cq.ktail);
        unsigned mask      = r->ring.cq.ring_mask;
        int count          = (int)(snap_tail - snap_head);

        if (count <= 0) break;

        for (int i = 0; i < count; i++) {
            cqe = &r->ring.cq.cqes[(snap_head + i) & mask];
            wolf_uring_req_t* req = (wolf_uring_req_t*)io_uring_cqe_get_data(cqe);
            if (!req) continue;

            /* SEND_ZC two-phase completion:
             *   Phase 1 — IORING_CQE_F_MORE set: kernel has taken ownership of
             *             the buffer but hasn't finished sending yet.  The arena
             *             must NOT be reset until phase 2 fires.  Skip callback.
             *   Phase 2 — IORING_CQE_F_MORE clear: kernel has released the
             *             buffer.  Now safe to call on_send_complete which resets
             *             the arena (or closes the connection). */
#ifdef IORING_CQE_F_MORE
            if (req->is_send_zc && (cqe->flags & IORING_CQE_F_MORE)) {
                continue; /* phase 1 — buffer still pinned in kernel */
            }
#endif
            if (req->cb) {
                req->cb(req->fd, req->ctx, cqe->res);
            }
            /* No free(req): arena-owned or malloc'd once (accept multishot) */
        }

        io_uring_cq_advance(&r->ring, count);
        total += count;

        /*
         * Commit any SQEs queued during this pass to the kernel's view of
         * the SQ ring (updates the kernel-visible ktail with a store-release).
         * With SQPOLL awake: the kernel thread sees the new tail immediately,
         * no syscall needed. With SQPOLL idle: io_uring_submit sends the
         * wakeup signal (one syscall, infrequent). Either way this is cheap.
         */
        io_uring_submit(&r->ring);

        /* If SQPOLL already processed some of those SQEs and injected CQEs,
         * the next snapshot will catch them and we drain them immediately
         * without returning to the caller. */
    }

    return total;
}

void wolf_uring_destroy(WolfURing* r) {
    if (r) {
        io_uring_queue_exit(&r->ring);
        free(r);
    }
}

#endif /* WOLF_HAS_IO_URING */

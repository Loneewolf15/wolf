#include "wolf_uring.h"

#ifdef WOLF_HAS_IO_URING

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

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

int wolf_uring_submit_accept(WolfURing* r, int server_fd, wolf_uring_cb_t cb, void* ctx) {
    struct io_uring_sqe *sqe = io_uring_get_sqe(&r->ring);
    if (!sqe) return -1;
    
    wolf_uring_req_t* req = (wolf_uring_req_t*)malloc(sizeof(wolf_uring_req_t));
    req->fd = server_fd;
    req->cb = cb;
    req->ctx = ctx;
    
    io_uring_prep_multishot_accept(sqe, server_fd, NULL, NULL, 0);
    io_uring_sqe_set_data(sqe, req);
    
    io_uring_submit(&r->ring);
    return 0;
}

int wolf_uring_submit_recv(WolfURing* r, int client_fd, void* buf, size_t len, wolf_uring_cb_t cb, void* ctx) {
    struct io_uring_sqe *sqe = io_uring_get_sqe(&r->ring);
    if (!sqe) return -1;
    
    wolf_uring_req_t* req = (wolf_uring_req_t*)malloc(sizeof(wolf_uring_req_t));
    req->fd = client_fd;
    req->cb = cb;
    req->ctx = ctx;
    
    io_uring_prep_recv(sqe, client_fd, buf, len, 0);
    io_uring_sqe_set_data(sqe, req);
    
    io_uring_submit(&r->ring);
    return 0;
}

int wolf_uring_submit_send(WolfURing* r, int client_fd, const void* buf, size_t len, wolf_uring_cb_t cb, void* ctx) {
    struct io_uring_sqe *sqe = io_uring_get_sqe(&r->ring);
    if (!sqe) return -1;
    
    wolf_uring_req_t* req = (wolf_uring_req_t*)malloc(sizeof(wolf_uring_req_t));
    req->fd = client_fd;
    req->cb = cb;
    req->ctx = ctx;
    
    io_uring_prep_send(sqe, client_fd, buf, len, MSG_WAITALL);
    io_uring_sqe_set_data(sqe, req);
    
    io_uring_submit(&r->ring);
    return 0;
}

int wolf_uring_poll(WolfURing* r, int timeout_ms) {
    struct __kernel_timespec ts;
    ts.tv_sec = timeout_ms / 1000;
    ts.tv_nsec = (timeout_ms % 1000) * 1000000;
    
    struct io_uring_cqe *cqe;
    int ret;
    
    if (timeout_ms >= 0) {
        ret = io_uring_wait_cqe_timeout(&r->ring, &cqe, &ts);
    } else {
        ret = io_uring_wait_cqe(&r->ring, &cqe);
    }
    
    if (ret < 0) return 0; // Timeout or error
    
    int count = 0;
    unsigned head;
    io_uring_for_each_cqe(&r->ring, head, cqe) {
        wolf_uring_req_t* req = (wolf_uring_req_t*)io_uring_cqe_get_data(cqe);
        if (req && req->cb) {
            req->cb(req->fd, req->ctx, cqe->res);
        }
        
        // Multi-shot accept requests emit multiple completions, don't free them immediately.
        // For recv/send we free them.
        if (req && !(cqe->flags & IORING_CQE_F_MORE)) {
            free(req);
        }
        
        count++;
    }
    
    if (count > 0) {
        io_uring_cq_advance(&r->ring, count);
    }
    return count;
}

void wolf_uring_destroy(WolfURing* r) {
    if (r) {
        io_uring_queue_exit(&r->ring);
        free(r);
    }
}

#endif /* WOLF_HAS_IO_URING */

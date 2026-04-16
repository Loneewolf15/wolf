#ifndef WOLF_URING_H
#define WOLF_URING_H

#include <stddef.h>

#if defined(__linux__)
#if __has_include(<liburing.h>)
#define WOLF_HAS_IO_URING 1
#endif
#endif

#ifdef WOLF_HAS_IO_URING

#include <liburing.h>

typedef void (*wolf_uring_cb_t)(int fd, void* ctx, int res);

typedef struct {
    int fd;
    wolf_uring_cb_t cb;
    void* ctx;
} wolf_uring_req_t;

typedef struct {
    struct io_uring ring;
    int sqpoll_enabled;
} WolfURing;

WolfURing* wolf_uring_create(int queue_depth, int sqpoll);
int        wolf_uring_submit_accept(WolfURing* ring, int server_fd, wolf_uring_cb_t cb, void* ctx);
int        wolf_uring_submit_recv(WolfURing* ring, int client_fd, void* buf, size_t len, wolf_uring_cb_t cb, void* ctx);
int        wolf_uring_submit_send(WolfURing* ring, int client_fd, const void* buf, size_t len, wolf_uring_cb_t cb, void* ctx);
int        wolf_uring_poll(WolfURing* ring, int timeout_ms);
void       wolf_uring_destroy(WolfURing* ring);

#endif /* WOLF_HAS_IO_URING */

#endif /* WOLF_URING_H */

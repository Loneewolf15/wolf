#ifndef WOLF_URING_H
#define WOLF_URING_H

#include <stddef.h>

#if defined(__linux__)
#define WOLF_HAS_IO_URING 1
#endif

#ifdef WOLF_HAS_IO_URING

#include <liburing.h>

typedef void (*wolf_uring_cb_t)(int fd, void* ctx, int res);

typedef struct {
    int fd;
    wolf_uring_cb_t cb;
    void* ctx;
    int   is_send_zc;  /* 1 = SEND_ZC op — needs two CQEs before callback fires */
} wolf_uring_req_t;

typedef struct {
    struct io_uring ring;
    int sqpoll_enabled;
} WolfURing;

#include "wolf_http_engine.h" // For WolfArena

WolfURing* wolf_uring_create(int queue_depth, int sqpoll);
int        wolf_uring_submit_accept(WolfURing* ring, int server_fd, wolf_uring_cb_t cb, void* ctx, WolfArena* arena);
int        wolf_uring_submit_recv(WolfURing* ring, int client_fd, void* buf, size_t len, wolf_uring_cb_t cb, void* ctx, WolfArena* arena);
int        wolf_uring_submit_send(WolfURing* ring, int client_fd, const void* buf, size_t len, wolf_uring_cb_t cb, void* ctx, WolfArena* arena);
int        wolf_uring_submit_send_zc(WolfURing* ring, int client_fd, const void* buf, size_t len, wolf_uring_cb_t cb, void* ctx, WolfArena* arena);
int        wolf_uring_has_send_zc(WolfURing* ring);
int        wolf_uring_poll_fd(WolfURing* ring, int fd, wolf_uring_cb_t cb, void* ctx);
int        wolf_uring_flush(WolfURing* ring);
int        wolf_uring_poll(WolfURing* ring, int timeout_ms);
void       wolf_uring_destroy(WolfURing* ring);

#endif /* WOLF_HAS_IO_URING */

#endif /* WOLF_URING_H */

/* wolf_iocp.h — Windows I/O Completion Port (IOCP) async I/O backend
 *
 * Architecture mirrors wolf_uring.h:
 *   wolf_iocp_create()        → io_uring_queue_init()
 *   wolf_iocp_post_accept()   → wolf_uring_submit_accept()
 *   wolf_iocp_post_recv()     → wolf_uring_submit_recv()
 *   wolf_iocp_post_send()     → wolf_uring_submit_send()
 *   wolf_iocp_poll()          → wolf_uring_poll()
 *   wolf_iocp_notify()        → write(notify_fd, 1) / eventfd
 *
 * Thread model:
 *   • One IOCP port per engine (all cores associate their sockets to it).
 *   • AcceptEx is posted multiple times in advance to keep the accept backlog
 *     saturated.  Each accepted connection gets its socket associated with the
 *     IOCP port and an immediate WSARecv posted.
 *   • Workers call wolf_iocp_notify() via PostQueuedCompletionStatus with
 *     completion key WOLF_IOCP_KEY_NOTIFY to wake the polling thread.
 *   • The polling thread calls GetQueuedCompletionStatusEx (batched) and
 *     dispatches to per-op callbacks identical to the io_uring path.
 *
 * Windows does not support SO_REUSEPORT: a single listening socket is shared
 * across all cores.  AcceptEx is thread-safe so multiple cores can post
 * AcceptEx concurrently on the same socket.
 */
#pragma once

#ifdef _WIN32

#include "wolf_thread_compat.h"

/* ── IOCP completion key values ────────────────────────────────────────── */
#define WOLF_IOCP_KEY_IO      ((ULONG_PTR)0)   /* normal I/O completion  */
#define WOLF_IOCP_KEY_NOTIFY  ((ULONG_PTR)1)   /* worker→poller wakeup   */
#define WOLF_IOCP_KEY_STOP    ((ULONG_PTR)2)   /* graceful shutdown       */

/* ── Per-operation types ─────────────────────────────────────────────── */
#define WOLF_IOCP_OP_ACCEPT   1
#define WOLF_IOCP_OP_RECV     2
#define WOLF_IOCP_OP_SEND     3

/* ── Overlapped request — MUST start with OVERLAPPED (IOCP ABI) ───────── */
typedef void (*wolf_iocp_cb_t)(SOCKET fd, void* ctx, int bytes);

typedef struct WolfIOCPReq {
    OVERLAPPED      ovl;       /* MUST be first — GetQueuedCompletionStatus returns this */
    int             op;        /* WOLF_IOCP_OP_* */
    wolf_iocp_cb_t  cb;
    void*           ctx;
    SOCKET          fd;        /* client fd (or accept socket for ACCEPT ops) */
    WSABUF          wsabuf;    /* points into buf[] below */
    /* AcceptEx output buffer: room for two SOCKADDR_IN + 16-byte padding each */
    char            accept_buf[2 * (sizeof(struct sockaddr_in) + 16)];
} WolfIOCPReq;

/* ── Per-engine IOCP context ────────────────────────────────────────────── */
#define WOLF_IOCP_PENDING_ACCEPTS  16   /* simultaneous AcceptEx operations */

typedef struct WolfIOCP {
    HANDLE   iocp;          /* I/O Completion Port handle                 */
    SOCKET   listen_fd;     /* single shared listen socket (no REUSEPORT) */
    /* Cached AcceptEx function pointer (obtained once via WSAIoctl) */
    LPFN_ACCEPTEX            fn_AcceptEx;
    LPFN_GETACCEPTEXSOCKADDRS fn_GetAcceptExSockaddrs;
} WolfIOCP;

/* ── Lifecycle ──────────────────────────────────────────────────────────── */
WolfIOCP* wolf_iocp_create(int port);
void      wolf_iocp_destroy(WolfIOCP* r);

/* ── Submit async operations ────────────────────────────────────────────── */
/* Post one AcceptEx operation.  cb fires with the new client SOCKET as fd. */
int wolf_iocp_post_accept(WolfIOCP* r, wolf_iocp_cb_t cb, void* ctx);

/* Post async WSARecv on client_fd.  buf must remain valid until cb fires. */
int wolf_iocp_post_recv(WolfIOCP* r, SOCKET client_fd,
                        char* buf, int len,
                        wolf_iocp_cb_t cb, void* ctx);

/* Post async WSASend on client_fd.  buf must remain valid until cb fires. */
int wolf_iocp_post_send(WolfIOCP* r, SOCKET client_fd,
                        const char* buf, int len,
                        wolf_iocp_cb_t cb, void* ctx);

/* Associate a newly accepted SOCKET with the IOCP port. */
int wolf_iocp_associate(WolfIOCP* r, SOCKET fd);

/* Wake the polling thread from a worker thread. */
void wolf_iocp_notify(WolfIOCP* r);

/* Drain up to WOLF_IOCP_BATCH completions.  Returns number processed.
 * timeout_ms: milliseconds to wait (0 = non-blocking, -1 = infinite). */
#define WOLF_IOCP_BATCH 64
int wolf_iocp_poll(WolfIOCP* r, int timeout_ms);

/* Re-post WOLF_IOCP_PENDING_ACCEPTS AcceptEx operations to keep the backlog
 * full.  Called at startup and after each accepted connection. */
int wolf_iocp_rearm_accepts(WolfIOCP* r, wolf_iocp_cb_t cb, void* ctx);

#endif /* _WIN32 */

/* wolf_iocp.c — Windows IOCP async I/O backend implementation */

#include "wolf_iocp.h"

#ifdef _WIN32

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

/* ── Internal helpers ────────────────────────────────────────────────────── */

static WolfIOCPReq* wolf_iocp_req_alloc(int op, SOCKET fd,
                                         wolf_iocp_cb_t cb, void* ctx) {
    WolfIOCPReq* req = (WolfIOCPReq*)calloc(1, sizeof(WolfIOCPReq));
    if (!req) return NULL;
    req->op  = op;
    req->fd  = fd;
    req->cb  = cb;
    req->ctx = ctx;
    return req;
}

/* Obtain the AcceptEx and GetAcceptExSockaddrs function pointers via WSAIoctl.
 * They live in the Winsock extension DLL and must be loaded this way. */
static int wolf_iocp_load_extensions(WolfIOCP* r) {
    GUID guid_acceptex  = WSAID_ACCEPTEX;
    GUID guid_getaddrs  = WSAID_GETACCEPTEXSOCKADDRS;
    DWORD bytes = 0;

    if (WSAIoctl(r->listen_fd, SIO_GET_EXTENSION_FUNCTION_POINTER,
                 &guid_acceptex, sizeof(guid_acceptex),
                 &r->fn_AcceptEx, sizeof(r->fn_AcceptEx),
                 &bytes, NULL, NULL) != 0) {
        fprintf(stderr, "[WOLF-IOCP] Failed to load AcceptEx: %d\n", WSAGetLastError());
        return -1;
    }
    if (WSAIoctl(r->listen_fd, SIO_GET_EXTENSION_FUNCTION_POINTER,
                 &guid_getaddrs, sizeof(guid_getaddrs),
                 &r->fn_GetAcceptExSockaddrs, sizeof(r->fn_GetAcceptExSockaddrs),
                 &bytes, NULL, NULL) != 0) {
        fprintf(stderr, "[WOLF-IOCP] Failed to load GetAcceptExSockaddrs: %d\n", WSAGetLastError());
        return -1;
    }
    return 0;
}

/* ── Lifecycle ───────────────────────────────────────────────────────────── */

WolfIOCP* wolf_iocp_create(int port) {
    WolfIOCP* r = (WolfIOCP*)calloc(1, sizeof(WolfIOCP));
    if (!r) return NULL;

    /* Create listen socket with overlapped I/O flag */
    r->listen_fd = WSASocket(AF_INET, SOCK_STREAM, IPPROTO_TCP,
                             NULL, 0, WSA_FLAG_OVERLAPPED);
    if (r->listen_fd == INVALID_SOCKET) {
        fprintf(stderr, "[WOLF-IOCP] WSASocket failed: %d\n", WSAGetLastError());
        free(r);
        return NULL;
    }

    /* Tune socket */
    BOOL reuse = TRUE;
    setsockopt(r->listen_fd, SOL_SOCKET, SO_REUSEADDR,
               (const char*)&reuse, sizeof(reuse));
    BOOL nodelay = TRUE;
    setsockopt(r->listen_fd, IPPROTO_TCP, TCP_NODELAY,
               (const char*)&nodelay, sizeof(nodelay));

    struct sockaddr_in addr;
    memset(&addr, 0, sizeof(addr));
    addr.sin_family      = AF_INET;
    addr.sin_port        = htons((u_short)port);
    addr.sin_addr.s_addr = INADDR_ANY;

    if (bind(r->listen_fd, (struct sockaddr*)&addr, sizeof(addr)) != 0) {
        fprintf(stderr, "[WOLF-IOCP] bind failed: %d\n", WSAGetLastError());
        closesocket(r->listen_fd);
        free(r);
        return NULL;
    }
    if (listen(r->listen_fd, SOMAXCONN) != 0) {
        fprintf(stderr, "[WOLF-IOCP] listen failed: %d\n", WSAGetLastError());
        closesocket(r->listen_fd);
        free(r);
        return NULL;
    }

    /* Create IOCP port and associate listen socket */
    r->iocp = CreateIoCompletionPort(INVALID_HANDLE_VALUE, NULL, 0, 0);
    if (!r->iocp) {
        fprintf(stderr, "[WOLF-IOCP] CreateIoCompletionPort failed: %lu\n", GetLastError());
        closesocket(r->listen_fd);
        free(r);
        return NULL;
    }
    if (!CreateIoCompletionPort((HANDLE)r->listen_fd, r->iocp,
                                 WOLF_IOCP_KEY_IO, 0)) {
        fprintf(stderr, "[WOLF-IOCP] Associate listen socket failed: %lu\n", GetLastError());
        CloseHandle(r->iocp);
        closesocket(r->listen_fd);
        free(r);
        return NULL;
    }

    /* Load AcceptEx etc. */
    if (wolf_iocp_load_extensions(r) != 0) {
        CloseHandle(r->iocp);
        closesocket(r->listen_fd);
        free(r);
        return NULL;
    }

    fprintf(stderr, "[WOLF-IOCP] Listening on port %d (IOCP)\n", port);
    return r;
}

void wolf_iocp_destroy(WolfIOCP* r) {
    if (!r) return;
    if (r->listen_fd != INVALID_SOCKET) closesocket(r->listen_fd);
    if (r->iocp) CloseHandle(r->iocp);
    free(r);
}

/* ── Associate a client socket with the IOCP port ─────────────────────── */

int wolf_iocp_associate(WolfIOCP* r, SOCKET fd) {
    return CreateIoCompletionPort((HANDLE)fd, r->iocp, WOLF_IOCP_KEY_IO, 0)
           ? 0 : -1;
}

/* ── AcceptEx — async accept ─────────────────────────────────────────────── */

int wolf_iocp_post_accept(WolfIOCP* r, wolf_iocp_cb_t cb, void* ctx) {
    /* Create a socket for the incoming connection */
    SOCKET accept_sock = WSASocket(AF_INET, SOCK_STREAM, IPPROTO_TCP,
                                   NULL, 0, WSA_FLAG_OVERLAPPED);
    if (accept_sock == INVALID_SOCKET) return -1;

    WolfIOCPReq* req = wolf_iocp_req_alloc(WOLF_IOCP_OP_ACCEPT, accept_sock, cb, ctx);
    if (!req) { closesocket(accept_sock); return -1; }

    /* AcceptEx output buffer: two sockaddr_in entries + 16-byte padding each */
    DWORD addr_size = sizeof(struct sockaddr_in) + 16;
    DWORD bytes_received = 0;

    BOOL ok = r->fn_AcceptEx(r->listen_fd, accept_sock,
                             req->accept_buf,
                             0,                 /* receive 0 bytes of data inline */
                             addr_size,         /* local address size */
                             addr_size,         /* remote address size */
                             &bytes_received,
                             &req->ovl);
    if (!ok && WSAGetLastError() != ERROR_IO_PENDING) {
        free(req);
        closesocket(accept_sock);
        return -1;
    }
    return 0;
}

int wolf_iocp_rearm_accepts(WolfIOCP* r, wolf_iocp_cb_t cb, void* ctx) {
    int posted = 0;
    for (int i = 0; i < WOLF_IOCP_PENDING_ACCEPTS; i++) {
        if (wolf_iocp_post_accept(r, cb, ctx) == 0) posted++;
    }
    return posted;
}

/* ── WSARecv — async recv ───────────────────────────────────────────────── */

int wolf_iocp_post_recv(WolfIOCP* r, SOCKET client_fd,
                        char* buf, int len,
                        wolf_iocp_cb_t cb, void* ctx) {
    (void)r;
    WolfIOCPReq* req = wolf_iocp_req_alloc(WOLF_IOCP_OP_RECV, client_fd, cb, ctx);
    if (!req) return -1;

    req->wsabuf.buf = buf;
    req->wsabuf.len = (ULONG)len;

    DWORD flags = 0, bytes = 0;
    int rc = WSARecv(client_fd, &req->wsabuf, 1, &bytes, &flags, &req->ovl, NULL);
    if (rc == SOCKET_ERROR && WSAGetLastError() != WSA_IO_PENDING) {
        free(req);
        return -1;
    }
    return 0;
}

/* ── WSASend — async send ───────────────────────────────────────────────── */

int wolf_iocp_post_send(WolfIOCP* r, SOCKET client_fd,
                        const char* buf, int len,
                        wolf_iocp_cb_t cb, void* ctx) {
    (void)r;
    WolfIOCPReq* req = wolf_iocp_req_alloc(WOLF_IOCP_OP_SEND, client_fd, cb, ctx);
    if (!req) return -1;

    req->wsabuf.buf = (char*)buf;  /* const cast: WSABUF is writable but we won't modify */
    req->wsabuf.len = (ULONG)len;

    DWORD bytes = 0;
    int rc = WSASend(client_fd, &req->wsabuf, 1, &bytes, 0, &req->ovl, NULL);
    if (rc == SOCKET_ERROR && WSAGetLastError() != WSA_IO_PENDING) {
        free(req);
        return -1;
    }
    return 0;
}

/* ── Worker → poller notification ───────────────────────────────────────── */

void wolf_iocp_notify(WolfIOCP* r) {
    /* PostQueuedCompletionStatus with WOLF_IOCP_KEY_NOTIFY wakes the polling
     * thread without allocating an overlapped structure. */
    PostQueuedCompletionStatus(r->iocp, 0, WOLF_IOCP_KEY_NOTIFY, NULL);
}

/* ── Completion polling ──────────────────────────────────────────────────── */

int wolf_iocp_poll(WolfIOCP* r, int timeout_ms) {
    OVERLAPPED_ENTRY entries[WOLF_IOCP_BATCH];
    ULONG count = 0;
    DWORD ms = (timeout_ms < 0) ? INFINITE : (DWORD)timeout_ms;

    BOOL ok = GetQueuedCompletionStatusEx(r->iocp, entries, WOLF_IOCP_BATCH,
                                          &count, ms, FALSE);
    if (!ok && GetLastError() == WAIT_TIMEOUT) return 0;
    if (!ok) return 0;

    for (ULONG i = 0; i < count; i++) {
        ULONG_PTR key = entries[i].lpCompletionKey;

        /* Worker notification — no overlapped request */
        if (key == WOLF_IOCP_KEY_NOTIFY || key == WOLF_IOCP_KEY_STOP) {
            /* Caller (wolf_core_thread) checks completion ring after wolf_iocp_poll */
            continue;
        }

        WolfIOCPReq* req = (WolfIOCPReq*)entries[i].lpOverlapped;
        if (!req) continue;

        int bytes = (int)entries[i].dwNumberOfBytesTransferred;
        int error = 0;
        if (!entries[i].lpOverlapped) { free(req); continue; }

        /* For ACCEPT: promote accept_sock, associate with IOCP, fire callback */
        if (req->op == WOLF_IOCP_OP_ACCEPT) {
            SOCKET client_fd = req->fd; /* the accept socket we pre-created */

            /* Inherit listen socket options (required by AcceptEx docs) */
            setsockopt(client_fd, SOL_SOCKET, SO_UPDATE_ACCEPT_CONTEXT,
                       (char*)&r->listen_fd, sizeof(r->listen_fd));

            /* Associate with IOCP so subsequent WSARecv/WSASend go through it */
            wolf_iocp_associate(r, client_fd);

            /* TCP_NODELAY */
            BOOL nodelay = TRUE;
            setsockopt(client_fd, IPPROTO_TCP, TCP_NODELAY,
                       (const char*)&nodelay, sizeof(nodelay));

            /* Save cb/ctx before freeing req */
            wolf_iocp_cb_t saved_cb  = req->cb;
            void*          saved_ctx = req->ctx;

            /* Fire callback: fd = new client socket */
            if (saved_cb) saved_cb(client_fd, saved_ctx, bytes);
            free(req);

            /* Re-post one AcceptEx to keep the accept backlog saturated */
            wolf_iocp_post_accept(r, saved_cb, saved_ctx);
            continue;
        }

        /* RECV or SEND — fire callback with byte count (or -1 on error) */
        if (bytes == 0) error = -1; /* connection closed */
        if (req->cb) req->cb(req->fd, req->ctx, error ? -1 : bytes);
        free(req);
    }

    return (int)count;
}

#endif /* _WIN32 */

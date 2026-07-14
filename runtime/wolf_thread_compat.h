/* wolf_thread_compat.h — portable threading and socket primitives
 *
 * On POSIX (Linux / macOS / BSD): thin passthrough to pthreads + standard sockets.
 * On Windows (MinGW-w64): winpthreads provides the full pthread API, so threading
 * code compiles unchanged.  This header adds portability shims for the handful of
 * POSIX socket extensions that don't exist on Windows:
 *   • SO_REUSEPORT  → SO_REUSEADDR
 *   • accept4()     → accept() + ioctlsocket(FIONBIO)
 *   • close() on sockets → closesocket()
 *   • O_NONBLOCK via fcntl() → ioctlsocket(FIONBIO)
 *   • MSG_DONTWAIT  → 0 (WinSock send/recv are always non-blocking when the socket
 *                        is in non-blocking mode)
 *   • SIGPIPE / SIG_IGN → no-op on Windows (WinSock never raises SIGPIPE)
 */
#pragma once

#ifdef _WIN32

/* ── WinSock + Windows headers ──────────────────────────────────────────── */
#ifndef WIN32_LEAN_AND_MEAN
#  define WIN32_LEAN_AND_MEAN
#endif
#include <winsock2.h>
#include <ws2tcpip.h>
#include <mswsock.h>    /* AcceptEx, TransmitFile */
#include <windows.h>

/* ── pthreads-w64 (winpthreads, bundled with MinGW-w64) ─────────────────── */
#include <pthread.h>

/* ── Socket fd type ─────────────────────────────────────────────────────── */
typedef SOCKET wolf_socket_t;
#define WOLF_INVALID_SOCKET  INVALID_SOCKET

/* ── Socket close ───────────────────────────────────────────────────────── */
#define wolf_close_socket(fd)   closesocket((SOCKET)(fd))

/* ── Non-blocking mode ──────────────────────────────────────────────────── */
static inline int wolf_set_nonblocking(SOCKET fd) {
    u_long mode = 1;
    return ioctlsocket(fd, FIONBIO, &mode) == 0 ? 0 : -1;
}

/* ── SO_REUSEPORT does not exist on Windows; map to SO_REUSEADDR ─────────── */
#ifndef SO_REUSEPORT
#  define SO_REUSEPORT  SO_REUSEADDR
#endif

/* ── accept4 emulation ──────────────────────────────────────────────────── */
#ifndef SOCK_NONBLOCK
#  define SOCK_NONBLOCK 0x4000   /* custom flag — intercepted by our macro */
#endif
static inline SOCKET wolf_accept4(SOCKET s, struct sockaddr* addr, int* addrlen, int flags) {
    SOCKET fd = accept(s, addr, addrlen);
    if (fd == INVALID_SOCKET) return INVALID_SOCKET;
    if (flags & SOCK_NONBLOCK) wolf_set_nonblocking(fd);
    return fd;
}
#define accept4(s, addr, addrlen, flags) \
    wolf_accept4((SOCKET)(s), (struct sockaddr*)(addr), (int*)(addrlen), (flags))

/* ── MSG_DONTWAIT — WinSock sockets in non-blocking mode are already async ─ */
#ifndef MSG_DONTWAIT
#  define MSG_DONTWAIT  0
#endif

/* ── MSG_WAITALL ────────────────────────────────────────────────────────── */
#ifndef MSG_WAITALL
#  define MSG_WAITALL  0x8
#endif

/* ── SIGPIPE does not exist on Windows; SIG_IGN is a no-op ─────────────── */
#ifndef SIGPIPE
#  define SIGPIPE  13
#endif
/* signal(SIGPIPE, SIG_IGN) → no-op on Windows */
#ifndef wolf_ignore_sigpipe
#  define wolf_ignore_sigpipe()  ((void)0)
#endif

/* ── ssize_t ────────────────────────────────────────────────────────────── */
#ifndef _SSIZE_T_DEFINED
#  define _SSIZE_T_DEFINED
typedef SSIZE_T ssize_t;
#endif

/* ── read/write on sockets — use recv/send on Windows ───────────────────── */
/* Note: wolf runtime always uses recv()/send() for socket I/O, so no macro
 * aliasing needed.  close() on file descriptors still works via MSVC/MinGW. */

/* ── WinSock initialisation — call once at process start ─────────────────── */
static inline int wolf_winsock_init(void) {
    WSADATA wsa;
    return WSAStartup(MAKEWORD(2, 2), &wsa) == 0 ? 0 : -1;
}
static inline void wolf_winsock_cleanup(void) { WSACleanup(); }

/* ── Thread affinity ────────────────────────────────────────────────────── */
static inline int wolf_pin_thread_to_core_win(HANDLE thread, int core_id) {
    DWORD_PTR mask = (DWORD_PTR)1 << (core_id % 64);
    return SetThreadAffinityMask(thread, mask) ? 0 : -1;
}

/* ── Pipe (for notify_fd on kqueue-equivalent path) ─────────────────────── */
/* Windows has _pipe() in <io.h>; we provide a socket-pair alternative since
 * PostQueuedCompletionStatus is preferred for IOCP notification. */

#else /* !_WIN32 ─────────────────────────────────────────────────────────── */

#include <pthread.h>
#include <unistd.h>
#include <fcntl.h>
#include <signal.h>

typedef int wolf_socket_t;
#define WOLF_INVALID_SOCKET  (-1)

#define wolf_close_socket(fd)   close(fd)

static inline int wolf_set_nonblocking(int fd) {
    int flags = fcntl(fd, F_GETFL, 0);
    return fcntl(fd, F_SETFL, flags | O_NONBLOCK);
}

#ifndef wolf_ignore_sigpipe
#  define wolf_ignore_sigpipe()  signal(SIGPIPE, SIG_IGN)
#endif

static inline int  wolf_winsock_init(void)    { return 0; }
static inline void wolf_winsock_cleanup(void) {}

#endif /* _WIN32 */

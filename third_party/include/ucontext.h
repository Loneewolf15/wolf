#ifndef UCONTEXT_H
#define UCONTEXT_H

#include <libucontext/libucontext.h>

#define ucontext_t libucontext_ucontext_t
#define mcontext_t libucontext_mcontext_t
#define getcontext libucontext_getcontext
#define setcontext libucontext_setcontext
#define swapcontext libucontext_swapcontext
#define makecontext libucontext_makecontext

#endif

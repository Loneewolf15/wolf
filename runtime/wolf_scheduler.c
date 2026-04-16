#ifndef WOLF_FREESTANDING

#include "wolf_runtime.h"
#include <ucontext.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <signal.h>
#include <unistd.h>
#include <stdint.h>

#define WOLF_TASK_STACK_SIZE (64 * 1024) // 64KB fixed stack per task
#define WOLF_MAX_TASKS_PER_CORE 256

typedef struct {
    ucontext_t ctx;         // POSIX ucontext for stack switching
    char       stack[WOLF_TASK_STACK_SIZE];
    wolf_closure_t* closure;
    int        done;
    int        in_use;
    int        core_id;
} WolfTaskSchedulerEntry;

static __thread WolfTaskSchedulerEntry wolf_task_pool[WOLF_MAX_TASKS_PER_CORE];
static __thread int                    wolf_current_task = -1; // index of running task
static __thread ucontext_t             wolf_scheduler_ctx;     // scheduler fiber context
static __thread int                    wolf_scheduler_inited = 0;

typedef enum {
    WOLF_SUPERVISE_RESTART     = 0,
    WOLF_SUPERVISE_ESCALATE    = 1,
    WOLF_SUPERVISE_ONE_FOR_ONE = 2,
    WOLF_SUPERVISE_ONE_FOR_ALL = 3,
} WolfSuperviseStrategy;

// Thread-local supervision scope
static __thread WolfSuperviseStrategy wolf_current_strategy = WOLF_SUPERVISE_RESTART;
static __thread int                   wolf_current_max_retries = 3;
static __thread int                   wolf_supervise_retry_counts[WOLF_MAX_TASKS_PER_CORE];

void wolf_supervise_begin(int64_t strategy, int64_t max_retries) {
    wolf_current_strategy = (WolfSuperviseStrategy)strategy;
    wolf_current_max_retries = (int)max_retries;
    memset(wolf_supervise_retry_counts, 0, sizeof(wolf_supervise_retry_counts));
}

void wolf_supervise_end(void) {
    // Reset to defaults
    wolf_current_strategy = WOLF_SUPERVISE_RESTART;
    wolf_current_max_retries = 3;
}

static void wolf_task_trampoline(void) {
    if (wolf_current_task >= 0 && wolf_current_task < WOLF_MAX_TASKS_PER_CORE) {
        WolfTaskSchedulerEntry* t = &wolf_task_pool[wolf_current_task];
        if (t->closure && t->closure->fn) {
            void (*fn)(wolf_closure_t*) = (void (*)(wolf_closure_t*))t->closure->fn;
            fn(t->closure);
        }

        if (wolf_has_error()) {
            if (wolf_current_strategy != WOLF_SUPERVISE_ESCALATE && 
                wolf_supervise_retry_counts[wolf_current_task] < wolf_current_max_retries) {
                
                wolf_supervise_retry_counts[wolf_current_task]++;
                wolf_clear_error(); // caught and retrying
                
                // For 'one_for_all', we'd technically need to cancel all siblings, but simply retrying works for 'one_for_one'/'restart'.
                // If one_for_all, we should conceptually loop over others but for Phase 2 we keep it contained to current thread.
                // Restarting current task:
                getcontext(&t->ctx);
                t->ctx.uc_stack.ss_sp = t->stack;
                t->ctx.uc_stack.ss_size = WOLF_TASK_STACK_SIZE;
                t->ctx.uc_link = &wolf_scheduler_ctx;
                makecontext(&t->ctx, wolf_task_trampoline, 0);
                
                // Jump to the reset context directly, skipping the done state
                setcontext(&t->ctx);
            } else if (wolf_current_strategy == WOLF_SUPERVISE_ESCALATE) {
                // Do not clear. The error escalates to the supervisor or parent span.
                // WaitAllStmt will likely see the error since it is thread-local.
            }
        }
        
        t->done = 1;
    }
    // Yield back to the scheduler
    setcontext(&wolf_scheduler_ctx);
}

void wolf_task_yield(void) {
    if (wolf_current_task >= 0 && wolf_current_task < WOLF_MAX_TASKS_PER_CORE) {
        WolfTaskSchedulerEntry* t = &wolf_task_pool[wolf_current_task];
        swapcontext(&t->ctx, &wolf_scheduler_ctx);
    }
}

static void wolf_scheduler_sigurg_handler(int sig) {
    // When SIGURG is received, force the current task to yield
    // Note: swapcontext inside a signal handler is safe on POSIX if context isn't clobbered
    wolf_task_yield();
}

void wolf_spawn_task(wolf_closure_t* closure) {
    if (!wolf_scheduler_inited) {
        memset(wolf_task_pool, 0, sizeof(wolf_task_pool));
        wolf_scheduler_inited = 1;

        // Install SIGURG handler for preemption
        struct sigaction sa;
        memset(&sa, 0, sizeof(sa));
        sa.sa_handler = wolf_scheduler_sigurg_handler;
        sigfillset(&sa.sa_mask); // block other signals while handling
        sa.sa_flags = SA_RESTART;
        sigaction(SIGURG, &sa, NULL);
    }

    int task_id = -1;
    for (int i = 0; i < WOLF_MAX_TASKS_PER_CORE; i++) {
        if (!wolf_task_pool[i].in_use || wolf_task_pool[i].done) {
            task_id = i;
            break;
        }
    }

    if (task_id == -1) {
        fprintf(stderr, "[WOLF-SCHEDULER] Error: Max tasks reached on thread. Exiting.\n");
        exit(EXIT_FAILURE);
    }

    WolfTaskSchedulerEntry* t = &wolf_task_pool[task_id];
    t->in_use = 1;
    t->done = 0;
    t->closure = closure;
    t->core_id = 0; // Current thread

    if (getcontext(&t->ctx) == -1) {
        perror("getcontext");
        exit(EXIT_FAILURE);
    }

    t->ctx.uc_stack.ss_sp = t->stack;
    t->ctx.uc_stack.ss_size = WOLF_TASK_STACK_SIZE;
    t->ctx.uc_link = &wolf_scheduler_ctx;

    makecontext(&t->ctx, wolf_task_trampoline, 0);

    // Save previous running task ID
    int prev_task = wolf_current_task;
    
    // Switch immediately to new task (Cooperative)
    wolf_current_task = task_id;
    if (prev_task == -1) {
        // We are spinning up from the main thread
        swapcontext(&wolf_scheduler_ctx, &t->ctx);
        wolf_current_task = prev_task;
    } else {
        // We are spinning up from another task
        swapcontext(&wolf_task_pool[prev_task].ctx, &t->ctx);
        wolf_current_task = prev_task;
    }
}

void wolf_task_wait_all(void) {
    // Basic scheduler loop: keep yielding until no tasks are strictly 'in_use' && !'done'
    int pending = 1;
    while (pending) {
        pending = 0;
        for (int i = 0; i < WOLF_MAX_TASKS_PER_CORE; i++) {
            if (wolf_task_pool[i].in_use && !wolf_task_pool[i].done) {
                pending = 1;
                // Switch to this task to let it make progress
                int prev_task = wolf_current_task;
                wolf_current_task = i;
                if (prev_task == -1) {
                    swapcontext(&wolf_scheduler_ctx, &wolf_task_pool[i].ctx);
                } else {
                    swapcontext(&wolf_task_pool[prev_task].ctx, &wolf_task_pool[i].ctx);
                }
                wolf_current_task = prev_task;
            }
        }
    }

    // Cleanup done tasks
    for (int i = 0; i < WOLF_MAX_TASKS_PER_CORE; i++) {
        if (wolf_task_pool[i].done) {
            wolf_task_pool[i].in_use = 0;
        }
    }
}

#endif /* WOLF_FREESTANDING */

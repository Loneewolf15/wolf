/*
 * wolf_timewheel.c — 3-Level Hierarchical Time-Wheel Implementation
 *
 * Thread-safety: NOT thread-safe by design.
 * All operations run on the owning I/O poller thread. This is enforced by
 * the architecture: one wheel per WolfCore, ticked only from wolf_core_thread.
 *
 * Cascade algorithm:
 *   When L0 wraps around (256 ticks = 256ms elapsed), cascade L1.
 *   When L1 wraps around (64 L1 ticks = 16.384s elapsed), cascade L2.
 *   Cascading re-inserts entries from the next outer level into L0.
 *
 * FD-reuse safety:
 *   fd_registered[] bitmap prevents close() on an FD that was already closed
 *   by normal request completion before the wheel fired.
 */

#define _GNU_SOURCE
#include "wolf_timewheel.h"

#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <unistd.h>

/* ── Internal helpers ─────────────────────────────────────────────────────── */

/* Place a node into the correct slot across all three levels. */
static void tw_insert(wolf_timewheel_t *tw, wolf_tw_node_t *node) {
    uint64_t now_ms    = tw->last_tick_ms;
    uint64_t deadline  = node->deadline_ms;
    uint64_t delta     = (deadline > now_ms) ? (deadline - now_ms) : 0;

    if (delta < (uint64_t)WOLF_TW_L0_SLOTS) {
        /* Expires within 256ms → L0 */
        uint32_t slot = (tw->l0_idx + (uint32_t)delta) & (WOLF_TW_L0_SLOTS - 1);
        node->next = tw->l0[slot];
        tw->l0[slot] = node;
    } else if (delta < (uint64_t)(WOLF_TW_L0_SLOTS * WOLF_TW_L1_SLOTS)) {
        /* Expires within 16.384s → L1 */
        uint32_t slot = (tw->l1_idx + (uint32_t)(delta / WOLF_TW_L0_SLOTS))
                        & (WOLF_TW_L1_SLOTS - 1);
        node->next = tw->l1[slot];
        tw->l1[slot] = node;
    } else {
        /* Expires beyond 16.384s (or far future) → L2, capped at max range */
        uint64_t l2_ticks = delta / ((uint64_t)WOLF_TW_L0_SLOTS * WOLF_TW_L1_SLOTS);
        if (l2_ticks >= WOLF_TW_L2_SLOTS) l2_ticks = WOLF_TW_L2_SLOTS - 1;
        uint32_t slot = (tw->l2_idx + (uint32_t)l2_ticks)
                        & (WOLF_TW_L2_SLOTS - 1);
        node->next = tw->l2[slot];
        tw->l2[slot] = node;
    }
}

/* Cascade L1 → L0: move all nodes from L1[l1_idx] back into L0. */
static void tw_cascade_l1(wolf_timewheel_t *tw) {
    wolf_tw_node_t *list = tw->l1[tw->l1_idx];
    tw->l1[tw->l1_idx] = NULL;
    while (list) {
        wolf_tw_node_t *next = list->next;
        list->next = NULL;
        tw_insert(tw, list);
        list = next;
    }
}

/* Cascade L2 → L1/L0: move all nodes from L2[l2_idx]. */
static void tw_cascade_l2(wolf_timewheel_t *tw) {
    wolf_tw_node_t *list = tw->l2[tw->l2_idx];
    tw->l2[tw->l2_idx] = NULL;
    while (list) {
        wolf_tw_node_t *next = list->next;
        list->next = NULL;
        tw_insert(tw, list);
        list = next;
    }
}

/* ── Public API ───────────────────────────────────────────────────────────── */

wolf_timewheel_t* wolf_timewheel_create(void) {
    wolf_timewheel_t *tw = (wolf_timewheel_t*)calloc(1, sizeof(wolf_timewheel_t));
    if (!tw) return NULL;

    /* All slot heads initialize to NULL (via calloc). */
    /* All fd_registered[] bytes initialize to 0 (via calloc). */

    tw->l0_idx       = 0;
    tw->l1_idx       = 0;
    tw->l2_idx       = 0;
    tw->last_tick_ms = wolf_monotonic_ms();

    fprintf(stderr, "[WOLF-TIMEWHEEL] Initialized. Timeout=%dms, Slots=L0:%d/L1:%d/L2:%d\n",
            WOLF_TIMEWHEEL_TIMEOUT_MS,
            WOLF_TW_L0_SLOTS, WOLF_TW_L1_SLOTS, WOLF_TW_L2_SLOTS);

    return tw;
}

void wolf_timewheel_destroy(wolf_timewheel_t *tw) {
    if (!tw) return;
    /* All nodes are from node_pool[], embedded in the struct — no heap to free. */
    free(tw);
}

void wolf_timewheel_add(wolf_timewheel_t *tw, int fd, uint64_t deadline_ms) {
    if (!tw || fd < 0 || fd >= WOLF_TW_MAX_FD) return;
    wolf_tw_node_t *node = &tw->node_pool[fd];

    if (node->in_wheel) {
        /* Already in the wheel (either active or lazily deleted). 
         * Just update the deadline. The tick loop will re-insert it based on 
         * the new deadline when it encounters it, preventing list corruption. */
        node->deadline_ms = deadline_ms;
        tw->fd_registered[fd] = 1;
        return;
    }

    node->fd          = fd;
    node->deadline_ms = deadline_ms;
    node->next        = NULL;
    node->in_wheel    = 1;
    tw->fd_registered[fd] = 1;

    tw_insert(tw, node);
}

void wolf_timewheel_remove(wolf_timewheel_t *tw, int fd) {
    if (!tw || fd < 0 || fd >= WOLF_TW_MAX_FD) return;
    if (!tw->fd_registered[fd]) return;

    /*
     * We do NOT search the slot lists to unlink the node (that would be O(n)).
     * Instead, we mark it unregistered. When the tick drains the slot and
     * encounters this node, fd_registered[fd] == 0, so it skips eviction.
     * This is the standard "lazy removal" pattern for time-wheels — correct
     * and O(1).
     */
    tw->fd_registered[fd] = 0;
}

int wolf_timewheel_tick(wolf_timewheel_t *tw,
                        void *core_ptr,
                        void (*evict_fn)(int fd, void *core_ptr)) {
    if (!tw) return 0;

    uint64_t now_ms  = wolf_monotonic_ms();
    uint64_t elapsed = now_ms - tw->last_tick_ms;
    if (elapsed == 0) return 0;

    /* Cap elapsed to prevent runaway cascade on resume from suspend */
    if (elapsed > (uint64_t)(WOLF_TW_L0_SLOTS * WOLF_TW_L1_SLOTS)) {
        elapsed = (uint64_t)(WOLF_TW_L0_SLOTS * WOLF_TW_L1_SLOTS);
    }

    int evicted = 0;

    for (uint64_t t = 0; t < elapsed; t++) {
        tw->last_tick_ms++;

        /* --- L0 tick --- */
        wolf_tw_node_t *list = tw->l0[tw->l0_idx];
        tw->l0[tw->l0_idx] = NULL;

        while (list) {
            wolf_tw_node_t *next = list->next;
            list->next = NULL;
            int fd = list->fd;

            if (tw->fd_registered[fd]) {
                if (list->deadline_ms > now_ms) {
                    /* It was re-armed! Re-insert it into the wheel based on new deadline! */
                    tw_insert(tw, list);
                } else {
                    /* EVICT! */
                    tw->fd_registered[fd] = 0;
                    list->in_wheel = 0;
                    evict_fn(fd, core_ptr);
                    evicted++;
                }
            } else {
                /* Not registered (closed or lazy deleted). Drop from wheel. */
                list->in_wheel = 0;
            }

            list = next;
        }

        /* --- Advance L0, cascade if wrapped --- */
        tw->l0_idx = (tw->l0_idx + 1) & (WOLF_TW_L0_SLOTS - 1);
        if (tw->l0_idx == 0) {
            /* L0 wrapped → advance L1 and cascade its current slot into L0 */
            tw_cascade_l1(tw);
            tw->l1_idx = (tw->l1_idx + 1) & (WOLF_TW_L1_SLOTS - 1);
            if (tw->l1_idx == 0) {
                /* L1 wrapped → advance L2 and cascade its current slot */
                tw_cascade_l2(tw);
                tw->l2_idx = (tw->l2_idx + 1) & (WOLF_TW_L2_SLOTS - 1);
            }
        }
    }

    return evicted;
}

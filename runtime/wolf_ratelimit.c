#include "wolf_ratelimit.h"
#include <stdlib.h>
#include <time.h>

#if __STDC_VERSION__ >= 201112L && !defined(__STDC_NO_ATOMICS__)
#include <stdatomic.h>
#else
#error "C11 atomics required for wolf_ratelimit"
#endif

typedef struct wolf_rl_bucket_t {
    /* 
     * We pack both tokens and timestamp into a single 64-bit atomic integer 
     * to allow lock-free CAS without double-width CAS (which isn't always available).
     * High 32 bits: tokens
     * Low 32 bits: last_refill_ms (wraps every 49 days, handled correctly by unsigned diff)
     */
    _Atomic uint64_t state;
} wolf_rl_bucket_t;

struct wolf_ratelimit_t {
    uint32_t         rps;
    uint32_t         burst;
    wolf_rl_bucket_t buckets[WOLF_RL_NUM_BUCKETS];
};

/* FNV-1a hash algorithm for fast, high-dispersion hashing */
static inline uint32_t fnv1a_hash(const uint8_t *data, int len) {
    uint32_t hash = 2166136261u;
    for (int i = 0; i < len; i++) {
        hash ^= data[i];
        hash *= 16777619u;
    }
    return hash;
}

/* Helper to get current monotonic time in milliseconds (truncated to 32 bits) */
static inline uint32_t wolf_monotonic_ms_32(void) {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    uint64_t ms = (uint64_t)ts.tv_sec * 1000ULL + (uint64_t)(ts.tv_nsec / 1000000ULL);
    return (uint32_t)ms;
}

wolf_ratelimit_t* wolf_ratelimit_create(uint32_t rps, uint32_t burst) {
    wolf_ratelimit_t *rl = calloc(1, sizeof(wolf_ratelimit_t));
    if (!rl) return NULL;
    
    rl->rps   = rps;
    rl->burst = burst;
    
    uint32_t now = wolf_monotonic_ms_32();
    uint64_t initial_state = ((uint64_t)burst << 32) | (uint64_t)now;
    
    for (int i = 0; i < WOLF_RL_NUM_BUCKETS; i++) {
        atomic_init(&rl->buckets[i].state, initial_state);
    }
    
    return rl;
}

void wolf_ratelimit_destroy(wolf_ratelimit_t *rl) {
    if (rl) {
        free(rl);
    }
}

bool wolf_ratelimit_allow(wolf_ratelimit_t *rl, const uint8_t *ip_bytes, int ip_len) {
    if (!rl || rl->rps == 0 || rl->burst == 0) return true; // Disabled
    
    uint32_t hash = fnv1a_hash(ip_bytes, ip_len);
    uint32_t idx  = hash & (WOLF_RL_NUM_BUCKETS - 1);
    
    wolf_rl_bucket_t *bucket = &rl->buckets[idx];
    
    uint32_t now_ms = wolf_monotonic_ms_32();
    
    uint64_t expected = atomic_load_explicit(&bucket->state, memory_order_relaxed);
    
    while (true) {
        uint32_t last_refill = (uint32_t)(expected & 0xFFFFFFFF);
        uint32_t tokens      = (uint32_t)(expected >> 32);
        
        // Use unsigned arithmetic to naturally handle the 49-day 32-bit wrap-around
        uint32_t elapsed_ms = now_ms - last_refill;
        
        uint32_t new_tokens = tokens;
        uint32_t new_refill = last_refill;
        
        if (elapsed_ms > 0) {
            // tokens_to_add = (elapsed_ms * rps) / 1000
            // Be careful of overflow when elapsed_ms * rps. Use 64-bit math.
            uint64_t add = ((uint64_t)elapsed_ms * (uint64_t)rl->rps) / 1000ULL;
            if (add > 0) {
                new_tokens += (uint32_t)add;
                if (new_tokens > rl->burst) {
                    new_tokens = rl->burst;
                }
                // Update refill time to account for exactly the tokens added
                // time_for_added_tokens = (add * 1000) / rps
                uint32_t time_adv = (uint32_t)(((uint64_t)add * 1000ULL) / (uint64_t)rl->rps);
                new_refill += time_adv;
            }
        }
        
        if (new_tokens == 0) {
            return false; // Rate limited
        }
        
        // Consume 1 token
        new_tokens -= 1;
        
        uint64_t new_state = ((uint64_t)new_tokens << 32) | (uint64_t)new_refill;
        
        if (atomic_compare_exchange_weak_explicit(&bucket->state, &expected, new_state, 
                                                  memory_order_release, memory_order_relaxed)) {
            return true;
        }
        // CAS failed, expected is updated to the current state, try again
    }
}

#ifndef WOLF_RATELIMIT_H
#define WOLF_RATELIMIT_H

#include <stdint.h>
#include <stdbool.h>

/* Number of buckets: must be a power of 2 for fast masking */
#define WOLF_RL_NUM_BUCKETS 65536

#ifndef WOLF_RATE_RPS
#define WOLF_RATE_RPS 100
#endif

#ifndef WOLF_RATE_BURST
#define WOLF_RATE_BURST 200
#endif

typedef struct wolf_ratelimit_t wolf_ratelimit_t;

/* Allocate and initialize a new rate limiter (shared across cores, or per-core)
 * We allocate one per WolfCore to avoid cross-core false sharing if possible,
 * or one global if we want global limits.
 * The implementation plan says "shared across cores, so it uses C11 atomics"
 */
wolf_ratelimit_t* wolf_ratelimit_create(uint32_t rps, uint32_t burst);
void wolf_ratelimit_destroy(wolf_ratelimit_t *rl);

/* Try to consume 1 token for the given IP address byte array.
 * ip_bytes points to the raw IP address (4 bytes for IPv4, 16 for IPv6).
 * Returns true if allowed, false if rate limited.
 */
bool wolf_ratelimit_allow(wolf_ratelimit_t *rl, const uint8_t *ip_bytes, int ip_len);

#endif

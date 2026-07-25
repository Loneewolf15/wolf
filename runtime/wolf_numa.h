#ifndef WOLF_NUMA_H
#define WOLF_NUMA_H

#include <stddef.h>

/* 
 * Initializes the NUMA subsystem. Dynamically attempts to load libnuma.so.
 * Returns 1 if NUMA is available and supported, 0 otherwise.
 */
int wolf_numa_init(void);

/*
 * Returns 1 if NUMA functions are successfully loaded and available.
 */
int wolf_numa_available(void);

/*
 * Returns the NUMA node of the specified CPU core.
 * Falls back to 0 if libnuma is unavailable.
 */
int wolf_numa_node_of_cpu(int cpu);

/*
 * Allocates `size` bytes on the specified NUMA `node`.
 * Falls back to standard malloc() if libnuma is unavailable.
 */
void* wolf_numa_alloc_onnode(size_t size, int node);

/*
 * Frees memory allocated by wolf_numa_alloc_onnode.
 */
void wolf_numa_free(void* ptr, size_t size);

#endif /* WOLF_NUMA_H */

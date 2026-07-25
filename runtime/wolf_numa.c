#include "wolf_numa.h"

#include <stdio.h>
#include <stdlib.h>

#ifdef __linux__
#include <dlfcn.h>
#endif

/* Function pointers for libnuma */
static int (*p_numa_available)(void) = NULL;
static int (*p_numa_node_of_cpu)(int cpu) = NULL;
static void* (*p_numa_alloc_onnode)(size_t size, int node) = NULL;
static void (*p_numa_free)(void* ptr, size_t size) = NULL;

static int wolf_numa_is_initialized = 0;
static int wolf_numa_is_available = 0;

int wolf_numa_init(void) {
    if (wolf_numa_is_initialized) {
        return wolf_numa_is_available;
    }
    wolf_numa_is_initialized = 1;

#ifdef __linux__
    /* Try to load libnuma dynamically */
    void* handle = dlopen("libnuma.so.1", RTLD_NOW | RTLD_LOCAL);
    if (!handle) {
        handle = dlopen("libnuma.so", RTLD_NOW | RTLD_LOCAL);
    }
    
    if (handle) {
        p_numa_available = (int (*)(void)) dlsym(handle, "numa_available");
        p_numa_node_of_cpu = (int (*)(int)) dlsym(handle, "numa_node_of_cpu");
        p_numa_alloc_onnode = (void* (*)(size_t, int)) dlsym(handle, "numa_alloc_onnode");
        p_numa_free = (void (*)(void*, size_t)) dlsym(handle, "numa_free");
        
        if (p_numa_available && p_numa_node_of_cpu && p_numa_alloc_onnode && p_numa_free) {
            if (p_numa_available() != -1) {
                wolf_numa_is_available = 1;
                fprintf(stderr, "[WOLF-NUMA] libnuma loaded successfully. NUMA bindings active.\n");
                return 1;
            }
        }
        
        fprintf(stderr, "[WOLF-NUMA] libnuma loaded, but NUMA is not available on this system.\n");
    } else {
        fprintf(stderr, "[WOLF-NUMA] libnuma not found. Falling back to standard malloc/free.\n");
    }
#else
    fprintf(stderr, "[WOLF-NUMA] OS is not Linux. NUMA bindings disabled.\n");
#endif

    return 0;
}

int wolf_numa_available(void) {
    if (!wolf_numa_is_initialized) {
        wolf_numa_init();
    }
    return wolf_numa_is_available;
}

int wolf_numa_node_of_cpu(int cpu) {
    if (wolf_numa_available() && p_numa_node_of_cpu) {
        return p_numa_node_of_cpu(cpu);
    }
    return 0; /* Fallback to node 0 */
}

void* wolf_numa_alloc_onnode(size_t size, int node) {
    if (wolf_numa_available() && p_numa_alloc_onnode) {
        return p_numa_alloc_onnode(size, node);
    }
    /* Fallback to standard allocation */
    return malloc(size);
}

void wolf_numa_free(void* ptr, size_t size) {
    if (wolf_numa_available() && p_numa_free) {
        p_numa_free(ptr, size);
    } else {
        free(ptr);
    }
}

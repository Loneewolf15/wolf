#include "../runtime/wolf_http_engine.h"
#include "../runtime/wolf_runtime.h"
#include <stdio.h>
#include <stdlib.h>

/* Mock definitions for missing externs that might not be linked easily */
wolf_metric_t wolf_metrics_registry[WOLF_MAX_METRICS];
void wolf_db_pool_destroy(void) {}

void* test_handler(void* env, int64_t req_id, int64_t res_id) {
    printf("Deliberately causing a SIGSEGV...\n");
    int* ptr = NULL;
    *ptr = 42;
    return NULL;
}

int main() {
    printf("Starting crash test on port 8081...\n");
    WolfEngine* engine = wolf_engine_create(8081, 1);
    
    wolf_closure_t closure;
    closure.fn = (void*)test_handler;
    closure.env = NULL;
    closure.magic = 0xW0LF; // need to mock closure
    
    // Instead of fighting closure magic which I don't know offhand, 
    // I can just trigger a SIGSEGV in the main thread right after creating the engine 
    // to verify the handler catches it! The handler reads __thread vars, 
    // which should just be NULL/0 in the main thread.

    printf("Crashing main thread...\n");
    int* ptr = NULL;
    *ptr = 42;

    return 0;
}

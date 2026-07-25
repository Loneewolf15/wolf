#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <assert.h>

typedef struct WolfFDEntryNode {
    struct WolfFDEntryNode* prev;
    struct WolfFDEntryNode* next;
} WolfFDEntryNode;

typedef struct WolfFDEntry {
    int fd;
    uint8_t in_wheel;
    WolfFDEntryNode node;
} WolfFDEntry;

typedef struct WolfTimeBucket {
    WolfFDEntryNode* head;
    WolfFDEntryNode* tail;
} WolfTimeBucket;

WolfTimeBucket bucket5 = {NULL, NULL};
WolfTimeBucket bucket15 = {NULL, NULL};

void add_buggy(WolfTimeBucket* bucket, WolfFDEntry* entry) {
    if (entry->node.next != NULL || entry->node.prev != NULL) {
        return;
    }
    if (bucket->tail) {
        bucket->tail->next = &entry->node;
        entry->node.prev = bucket->tail;
        bucket->tail = &entry->node;
    } else {
        bucket->head = bucket->tail = &entry->node;
        entry->node.prev = NULL;
    }
    entry->node.next = NULL;
}

int main(int argc, char** argv) {
    WolfFDEntry* e = calloc(1, sizeof(WolfFDEntry));
    e->fd = 10;
    
    // Add to bucket 5
    add_buggy(&bucket5, e);
    
    // Refresh connection -> calculates new timeout in bucket 15
    // Because it's the only element in bucket5, its prev and next are NULL.
    // add_buggy will NOT reject it!
    add_buggy(&bucket15, e);
    
    // Now check both buckets
    printf("Bucket 5 head = %p (expected e=%p)\n", (void*)bucket5.head, (void*)&e->node);
    printf("Bucket 15 head = %p (expected e=%p)\n", (void*)bucket15.head, (void*)&e->node);
    
    if (bucket5.head == &e->node && bucket15.head == &e->node) {
        printf("CRITICAL BUG: 'e' is simultaneously the head of Bucket 5 AND Bucket 15!\n");
        printf("When Bucket 5 ticks, it will close the FD. When Bucket 15 ticks, it will double-close or close a reused FD!\n");
    }
    
    return 0;
}

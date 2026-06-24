import re

with open("runtime/wolf_runtime.c", "r") as f:
    content = f.read()

old_delete = """void wolf_map_delete(void* map_ptr, const char* key) {
    if (!map_ptr || !key) return;
    wolf_map_t* m = (wolf_map_t*)map_ptr;
    
    for (int64_t i = 0; i < m->size; i++) {
        if (strcmp(m->keys[i], key) == 0) {
            // Shift left to maintain insertion order
            for (int64_t j = i; j < m->size - 1; j++) {
                m->keys[j] = m->keys[j + 1];
                m->values[j] = m->values[j + 1];
            }
            m->size--;
            
            // Rebuild buckets
            for (int64_t k = 0; k < m->capacity; k++) m->buckets[k] = -1;
            for (int64_t k = 0; k < m->size; k++) {
                uint32_t h = wolf_hash_str(m->keys[k]);
                int32_t b = h % m->capacity;
                m->next[k] = m->buckets[b];
                m->buckets[b] = k;
            }
            return;
        }
    }
}"""

new_delete = """void wolf_map_delete(void* map_ptr, const char* key) {
    if (!map_ptr || !key) return;
    wolf_map_t* m = (wolf_map_t*)map_ptr;
    
    uint32_t hash = wolf_hash_str(key);
    int32_t bucket = hash % m->capacity;
    int32_t curr = m->buckets[bucket];
    int32_t prev = -1;
    
    while (curr != -1) {
        if (strcmp(m->keys[curr], key) == 0) {
            // Unlink from current bucket chain
            if (prev == -1) {
                m->buckets[bucket] = m->next[curr];
            } else {
                m->next[prev] = m->next[curr];
            }
            
            // Swap with the last element for O(1) removal
            int64_t last_idx = m->size - 1;
            if (curr != last_idx) {
                m->keys[curr] = m->keys[last_idx];
                m->values[curr] = m->values[last_idx];
                
                // We moved the last element to `curr`. 
                // We must update its bucket chain pointer to point to `curr` instead of `last_idx`.
                uint32_t last_hash = wolf_hash_str(m->keys[curr]);
                int32_t last_bucket = last_hash % m->capacity;
                int32_t c = m->buckets[last_bucket];
                int32_t p = -1;
                while (c != -1) {
                    if (c == last_idx) {
                        if (p == -1) m->buckets[last_bucket] = curr;
                        else m->next[p] = curr;
                        m->next[curr] = m->next[last_idx];
                        break;
                    }
                    p = c;
                    c = m->next[c];
                }
            }
            m->size--;
            return;
        }
        prev = curr;
        curr = m->next[curr];
    }
}"""

content = content.replace(old_delete, new_delete)
with open("runtime/wolf_runtime.c", "w") as f:
    f.write(content)
print("delete fixed")

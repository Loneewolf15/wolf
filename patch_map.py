import re

with open("runtime/wolf_runtime.c", "r") as f:
    content = f.read()

# 1. Update struct
old_struct = """typedef struct {
    char**  keys;
    void**  values;
    int64_t size;
    int64_t capacity;
} wolf_map_t;"""
new_struct = """typedef struct {
    char**  keys;
    void**  values;
    int32_t* buckets;
    int32_t* next;
    int64_t size;
    int64_t capacity;
} wolf_map_t;"""
content = content.replace(old_struct, new_struct)

# 2. wolf_map_create
old_create = """void* wolf_map_create() {
    wolf_map_t* m = (wolf_map_t*)wolf_req_alloc(sizeof(wolf_map_t));
    if (!m) return NULL;
    m->size     = 0;
    m->capacity = 16;
    m->keys     = (char**)wolf_req_alloc(sizeof(char*) * m->capacity);
    m->values   = (void**)wolf_req_alloc(sizeof(void*) * m->capacity);
    return m;
}"""
new_create = """static uint32_t wolf_hash_str(const char* str) {
    uint32_t hash = 2166136261u;
    while (*str) {
        hash ^= (uint8_t)(*str++);
        hash *= 16777619;
    }
    return hash;
}

void* wolf_map_create() {
    wolf_map_t* m = (wolf_map_t*)wolf_req_alloc(sizeof(wolf_map_t));
    if (!m) return NULL;
    m->size     = 0;
    m->capacity = 16;
    m->keys     = (char**)wolf_req_alloc(sizeof(char*) * m->capacity);
    m->values   = (void**)wolf_req_alloc(sizeof(void*) * m->capacity);
    m->buckets  = (int32_t*)wolf_req_alloc(sizeof(int32_t) * m->capacity);
    m->next     = (int32_t*)wolf_req_alloc(sizeof(int32_t) * m->capacity);
    for (int64_t i = 0; i < m->capacity; i++) m->buckets[i] = -1;
    return m;
}"""
content = content.replace(old_create, new_create)

# 3. wolf_map_set
old_set = """void wolf_map_set(void* map_ptr, const char* key, void* v) {
    if (!map_ptr || !key) return;
    wolf_map_t* m = (wolf_map_t*)map_ptr;
    for (int64_t i = 0; i < m->size; i++) if (strcmp(m->keys[i], key)==0) { m->values[i]=v; return; }
    if (m->size >= m->capacity) {
        int64_t old_k = m->capacity * sizeof(char*);
        int64_t old_v = m->capacity * sizeof(void*);
        m->capacity *= 2;
        m->keys=(char**)wolf_req_realloc(m->keys, old_k, sizeof(char*)*m->capacity);
        m->values=(void**)wolf_req_realloc(m->values, old_v, sizeof(void*)*m->capacity);
    }
    m->keys[m->size] = wolf_req_strdup(key); m->values[m->size]=v; m->size++;
}"""
new_set = """void wolf_map_set(void* map_ptr, const char* key, void* v) {
    if (!map_ptr || !key) return;
    wolf_map_t* m = (wolf_map_t*)map_ptr;
    
    uint32_t hash = wolf_hash_str(key);
    int32_t bucket = hash % m->capacity;
    
    int32_t curr = m->buckets[bucket];
    while (curr != -1) {
        if (strcmp(m->keys[curr], key) == 0) {
            m->values[curr] = v;
            return;
        }
        curr = m->next[curr];
    }
    
    if (m->size >= m->capacity) {
        int64_t old_cap = m->capacity;
        m->capacity *= 2;
        m->keys = (char**)wolf_req_realloc(m->keys, old_cap * sizeof(char*), sizeof(char*) * m->capacity);
        m->values = (void**)wolf_req_realloc(m->values, old_cap * sizeof(void*), sizeof(void*) * m->capacity);
        m->buckets = (int32_t*)wolf_req_realloc(m->buckets, old_cap * sizeof(int32_t), sizeof(int32_t) * m->capacity);
        m->next = (int32_t*)wolf_req_realloc(m->next, old_cap * sizeof(int32_t), sizeof(int32_t) * m->capacity);
        
        for (int64_t i = 0; i < m->capacity; i++) m->buckets[i] = -1;
        for (int64_t i = 0; i < m->size; i++) {
            uint32_t h = wolf_hash_str(m->keys[i]);
            int32_t b = h % m->capacity;
            m->next[i] = m->buckets[b];
            m->buckets[b] = i;
        }
        
        bucket = hash % m->capacity;
    }
    
    m->keys[m->size] = wolf_req_strdup(key);
    m->values[m->size] = v;
    m->next[m->size] = m->buckets[bucket];
    m->buckets[bucket] = m->size;
    m->size++;
}"""
content = content.replace(old_set, new_set)

# 4. wolf_map_get
old_get = """void* wolf_map_get(void* map_ptr, const char* key) {
    if (!map_ptr || !key) return NULL;
    wolf_map_t* m = (wolf_map_t*)map_ptr;
    for (int64_t i = 0; i < m->size; i++) if (strcmp(m->keys[i], key)==0) return m->values[i];
    return NULL;
}"""
new_get = """void* wolf_map_get(void* map_ptr, const char* key) {
    if (!map_ptr || !key) return NULL;
    wolf_map_t* m = (wolf_map_t*)map_ptr;
    
    uint32_t hash = wolf_hash_str(key);
    int32_t bucket = hash % m->capacity;
    int32_t curr = m->buckets[bucket];
    
    while (curr != -1) {
        if (strcmp(m->keys[curr], key) == 0) return m->values[curr];
        curr = m->next[curr];
    }
    return NULL;
}"""
content = content.replace(old_get, new_get)

# 5. wolf_map_has
old_has = """int wolf_map_has(void* map_ptr, const char* key) {
    if (!map_ptr || !key) return 0;
    wolf_map_t* m = (wolf_map_t*)map_ptr;
    for (int64_t i = 0; i < m->size; i++) if (strcmp(m->keys[i], key)==0) return 1;
    return 0;
}"""
new_has = """int wolf_map_has(void* map_ptr, const char* key) {
    if (!map_ptr || !key) return 0;
    wolf_map_t* m = (wolf_map_t*)map_ptr;
    
    uint32_t hash = wolf_hash_str(key);
    int32_t bucket = hash % m->capacity;
    int32_t curr = m->buckets[bucket];
    
    while (curr != -1) {
        if (strcmp(m->keys[curr], key) == 0) return 1;
        curr = m->next[curr];
    }
    return 0;
}"""
content = content.replace(old_has, new_has)

# 6. Delete
old_delete = """void wolf_map_delete(void* map_ptr, const char* key) {
    if (!map_ptr || !key) return;
    wolf_map_t* m = (wolf_map_t*)map_ptr;
    for (int64_t i = 0; i < m->size; i++) {
        if (strcmp(m->keys[i], key) == 0) {
            // Shift left
            for (int64_t j = i; j < m->size - 1; j++) {
                m->keys[j] = m->keys[j + 1];
                m->values[j] = m->values[j + 1];
            }
            m->size--;
            return;
        }
    }
}"""
new_delete = """void wolf_map_delete(void* map_ptr, const char* key) {
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
content = content.replace(old_delete, new_delete)

with open("runtime/wolf_runtime.c", "w") as f:
    f.write(content)
print("map refactored")

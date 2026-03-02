/*
 * Wolf Runtime Library
 * Provides core functions that compiled Wolf programs link against.
 * Compiled with: clang -c wolf_runtime.c -o wolf_runtime.o
 */

#include "wolf_runtime.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <ctype.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <unistd.h>
#include <pthread.h>
#include <strings.h>

// ========== Print ==========

void wolf_print_str(const char* s) {
    if (s) {
        printf("%s", s);
    }
}

void wolf_print_int(int64_t n) {
    printf("%lld", (long long)n);
}

void wolf_print_float(double f) {
    // Remove trailing zeros for clean output
    char buf[64];
    snprintf(buf, sizeof(buf), "%g", f);
    printf("%s", buf);
}

void wolf_print_bool(int b) {
    printf("%s", b ? "true" : "false");
}

void wolf_print_nil(void) {
    printf("nil");
}

void wolf_println(void) {
    printf("\n");
}

// ========== String Operations ==========

const char* wolf_string_concat(const char* a, const char* b) {
    if (!a) a = "";
    if (!b) b = "";
    size_t la = strlen(a);
    size_t lb = strlen(b);
    char* result = (char*)malloc(la + lb + 1);
    memcpy(result, a, la);
    memcpy(result + la, b, lb);
    result[la + lb] = '\0';
    return result;
}

int64_t wolf_string_length(const char* s) {
    if (!s) return 0;
    return (int64_t)strlen(s);
}

const char* wolf_string_upper(const char* s) {
    if (!s) return "";
    size_t len = strlen(s);
    char* result = (char*)malloc(len + 1);
    for (size_t i = 0; i < len; i++) {
        result[i] = (char)toupper((unsigned char)s[i]);
    }
    result[len] = '\0';
    return result;
}

const char* wolf_string_lower(const char* s) {
    if (!s) return "";
    size_t len = strlen(s);
    char* result = (char*)malloc(len + 1);
    for (size_t i = 0; i < len; i++) {
        result[i] = (char)tolower((unsigned char)s[i]);
    }
    result[len] = '\0';
    return result;
}

const char* wolf_string_trim(const char* s) {
    if (!s) return "";
    // Skip leading whitespace
    while (*s && isspace((unsigned char)*s)) s++;
    size_t len = strlen(s);
    // Skip trailing whitespace
    while (len > 0 && isspace((unsigned char)s[len - 1])) len--;
    char* result = (char*)malloc(len + 1);
    memcpy(result, s, len);
    result[len] = '\0';
    return result;
}

// ========== Conversions ==========

const char* wolf_int_to_string(int64_t n) {
    char* buf = (char*)malloc(32);
    snprintf(buf, 32, "%lld", (long long)n);
    return buf;
}

const char* wolf_float_to_string(double f) {
    char* buf = (char*)malloc(64);
    snprintf(buf, 64, "%g", f);
    return buf;
}

const char* wolf_bool_to_string(int b) {
    return b ? "true" : "false";
}

// ========== Memory ==========

void* wolf_alloc(int64_t size) {
    return malloc((size_t)size);
}

void wolf_free(void* ptr) {
    free(ptr);
}

// ========== HTTP Server ==========

#define MAX_CONCURRENT_REQUESTS 1024

typedef struct {
    int active;
    int client_fd;
    char* method;
    char* path;
    char* query;
    char* body;
    char* header_keys[32];
    char* header_vals[32];
    int header_count;
    
    int status_code;
    char* res_header_keys[32];
    char* res_header_vals[32];
    int res_header_count;
    char* res_body;
} wolf_http_context_t;

static wolf_http_context_t http_contexts[MAX_CONCURRENT_REQUESTS];
static pthread_mutex_t http_mutex = PTHREAD_MUTEX_INITIALIZER;
static wolf_http_handler_t global_wolf_handler = NULL;

static int alloc_http_context(int client_fd) {
    pthread_mutex_lock(&http_mutex);
    for (int i = 0; i < MAX_CONCURRENT_REQUESTS; i++) {
        if (!http_contexts[i].active) {
            memset(&http_contexts[i], 0, sizeof(wolf_http_context_t));
            http_contexts[i].active = 1;
            http_contexts[i].client_fd = client_fd;
            http_contexts[i].status_code = 200;
            pthread_mutex_unlock(&http_mutex);
            return i;
        }
    }
    pthread_mutex_unlock(&http_mutex);
    return -1;
}

static void free_http_context(int id) {
    if (id < 0 || id >= MAX_CONCURRENT_REQUESTS) return;
    pthread_mutex_lock(&http_mutex);
    
    wolf_http_context_t* ctx = &http_contexts[id];
    if (ctx->method) free(ctx->method);
    if (ctx->path) free(ctx->path);
    if (ctx->query) free(ctx->query);
    if (ctx->body) free(ctx->body);
    for (int i = 0; i < ctx->header_count; i++) {
        free(ctx->header_keys[i]);
        free(ctx->header_vals[i]);
    }
    for (int i = 0; i < ctx->res_header_count; i++) {
        free(ctx->res_header_keys[i]);
        free(ctx->res_header_vals[i]);
    }
    if (ctx->res_body) free(ctx->res_body);
    
    ctx->active = 0;
    pthread_mutex_unlock(&http_mutex);
}

static void parse_http_request(int id, char* raw_req, size_t len) {
    wolf_http_context_t* ctx = &http_contexts[id];
    
    // Find body
    char* body_start = strstr(raw_req, "\r\n\r\n");
    if (body_start) {
        *body_start = '\0';
        body_start += 4;
        ctx->body = strdup(body_start);
    } else {
        ctx->body = strdup("");
    }
    
    char* saveptr;
    char* line = strtok_r(raw_req, "\r\n", &saveptr);
    if (!line) return;
    
    char* l_save;
    char* method = strtok_r(line, " ", &l_save);
    char* full_path = strtok_r(NULL, " ", &l_save);
    
    if (method) ctx->method = strdup(method);
    if (full_path) {
        char* q_mark = strchr(full_path, '?');
        if (q_mark) {
            *q_mark = '\0';
            ctx->path = strdup(full_path);
            ctx->query = strdup(q_mark + 1);
        } else {
            ctx->path = strdup(full_path);
            ctx->query = strdup("");
        }
    }
    
    while ((line = strtok_r(NULL, "\r\n", &saveptr))) {
        char* colon = strchr(line, ':');
        if (colon && ctx->header_count < 32) {
            *colon = '\0';
            char* val = colon + 1;
            while (*val == ' ') val++;
            ctx->header_keys[ctx->header_count] = strdup(line);
            ctx->header_vals[ctx->header_count] = strdup(val);
            ctx->header_count++;
        }
    }
}

static void* http_worker(void* arg) {
    int id = (int)(intptr_t)arg;
    wolf_http_context_t* ctx = &http_contexts[id];
    
    char buffer[8192];
    memset(buffer, 0, sizeof(buffer));
    
    ssize_t bytes_read = read(ctx->client_fd, buffer, sizeof(buffer) - 1);
    if (bytes_read > 0) {
        parse_http_request(id, buffer, bytes_read);
        
        if (global_wolf_handler) {
            global_wolf_handler((int64_t)id, (int64_t)id);
        }
        
        char res_header[2048];
        snprintf(res_header, sizeof(res_header), "HTTP/1.1 %d OK\r\n", ctx->status_code);
        write(ctx->client_fd, res_header, strlen(res_header));
        
        for (int i = 0; i < ctx->res_header_count; i++) {
            snprintf(res_header, sizeof(res_header), "%s: %s\r\n", ctx->res_header_keys[i], ctx->res_header_vals[i]);
            write(ctx->client_fd, res_header, strlen(res_header));
        }
        
        int body_len = ctx->res_body ? strlen(ctx->res_body) : 0;
        snprintf(res_header, sizeof(res_header), "Content-Length: %d\r\n\r\n", body_len);
        write(ctx->client_fd, res_header, strlen(res_header));
        
        if (body_len > 0) {
            write(ctx->client_fd, ctx->res_body, body_len);
        }
    }
    
    close(ctx->client_fd);
    free_http_context(id);
    return NULL;
}

void wolf_http_serve(int64_t port, void* handler_ptr) {
    global_wolf_handler = (wolf_http_handler_t)handler_ptr;
    
    int server_fd;
    struct sockaddr_in address;
    int opt = 1;
    
    if ((server_fd = socket(AF_INET, SOCK_STREAM, 0)) == 0) {
        perror("wolf_http: socket failed");
        exit(EXIT_FAILURE);
    }
    
    if (setsockopt(server_fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt))) {
        perror("wolf_http: setsockopt");
        exit(EXIT_FAILURE);
    }
    
    address.sin_family = AF_INET;
    address.sin_addr.s_addr = INADDR_ANY;
    address.sin_port = htons(port);
    
    if (bind(server_fd, (struct sockaddr *)&address, sizeof(address)) < 0) {
        perror("wolf_http: bind failed");
        exit(EXIT_FAILURE);
    }
    
    if (listen(server_fd, 1024) < 0) {
        perror("wolf_http: listen failed");
        exit(EXIT_FAILURE);
    }
    
    printf("🐺 Wolf HTTP Server running on port %d...\n", (int)port);
    
    while (1) {
        struct sockaddr_in client_addr;
        socklen_t client_len = sizeof(client_addr);
        int client_fd = accept(server_fd, (struct sockaddr *)&client_addr, &client_len);
        if (client_fd < 0) continue;
        
        int id = alloc_http_context(client_fd);
        if (id >= 0) {
            pthread_t thread;
            pthread_create(&thread, NULL, http_worker, (void*)(intptr_t)id);
            pthread_detach(thread);
        } else {
            char* busy = "HTTP/1.1 503 Service Unavailable\r\n\r\n";
            write(client_fd, busy, strlen(busy));
            close(client_fd);
        }
    }
}

// --- Request API ---

const char* wolf_http_req_method(int64_t req_id) {
    if (req_id < 0 || req_id >= MAX_CONCURRENT_REQUESTS) return "";
    return http_contexts[req_id].method ? http_contexts[req_id].method : "";
}

const char* wolf_http_req_path(int64_t req_id) {
    if (req_id < 0 || req_id >= MAX_CONCURRENT_REQUESTS) return "";
    return http_contexts[req_id].path ? http_contexts[req_id].path : "";
}

const char* wolf_http_req_query(int64_t req_id, const char* key) {
    if (req_id < 0 || req_id >= MAX_CONCURRENT_REQUESTS || !key) return "";
    char* query = http_contexts[req_id].query;
    if (!query) return "";
    
    char* q_copy = strdup(query);
    char* saveptr;
    char* pair = strtok_r(q_copy, "&", &saveptr);
    
    while (pair) {
        char* eq = strchr(pair, '=');
        if (eq) {
            *eq = '\0';
            if (strcmp(pair, key) == 0) {
                char* result = strdup(eq + 1);
                free(q_copy);
                return result;
            }
        }
        pair = strtok_r(NULL, "&", &saveptr);
    }
    free(q_copy);
    return "";
}

const char* wolf_http_req_header(int64_t req_id, const char* key) {
    if (req_id < 0 || req_id >= MAX_CONCURRENT_REQUESTS || !key) return "";
    wolf_http_context_t* ctx = &http_contexts[req_id];
    for (int i = 0; i < ctx->header_count; i++) {
        if (strcasecmp(ctx->header_keys[i], key) == 0) {
            return ctx->header_vals[i];
        }
    }
    return "";
}

const char* wolf_http_req_body(int64_t req_id) {
    if (req_id < 0 || req_id >= MAX_CONCURRENT_REQUESTS) return "";
    return http_contexts[req_id].body ? http_contexts[req_id].body : "";
}

// --- Response API ---

void wolf_http_res_header(int64_t res_id, const char* key, const char* value) {
    if (res_id < 0 || res_id >= MAX_CONCURRENT_REQUESTS || !key || !value) return;
    wolf_http_context_t* ctx = &http_contexts[res_id];
    if (ctx->res_header_count < 32) {
        ctx->res_header_keys[ctx->res_header_count] = strdup(key);
        ctx->res_header_vals[ctx->res_header_count] = strdup(value);
        ctx->res_header_count++;
    }
}

void wolf_http_res_status(int64_t res_id, int64_t status_code) {
    if (res_id < 0 || res_id >= MAX_CONCURRENT_REQUESTS) return;
    http_contexts[res_id].status_code = (int)status_code;
}

void wolf_http_res_write(int64_t res_id, const char* body) {
    if (res_id < 0 || res_id >= MAX_CONCURRENT_REQUESTS || !body) return;
    wolf_http_context_t* ctx = &http_contexts[res_id];
    if (ctx->res_body) {
        size_t old_len = strlen(ctx->res_body);
        size_t new_len = strlen(body);
        char* new_body = malloc(old_len + new_len + 1);
        strcpy(new_body, ctx->res_body);
        strcat(new_body, body);
        free(ctx->res_body);
        ctx->res_body = new_body;
    } else {
        ctx->res_body = strdup(body);
    }
}

#include "wolf_http_engine.h"
#include <string.h>
#include <ctype.h>

#if defined(__x86_64__) && defined(__AVX2__)
#include <immintrin.h>
#endif

/* Constant-time string comparison for security-sensitive headers (e.g. Authorization) */
int wolf_header_compare_ct(const char* a, const char* b) {
    if (!a || !b) return a == b ? 0 : 1;
    size_t len_a = strlen(a);
    size_t len_b = strlen(b);
    if (len_a != len_b) return 1;
    
    unsigned char result = 0;
    for (size_t i = 0; i < len_a; i++) {
        result |= (a[i] ^ b[i]);
    }
    return result;
}

/* HTTP Smuggling Defense: Reject ambiguous requests */
static int validate_crlf_and_smuggling(WolfConnCtx* ctx) {
    int has_cl = 0;
    int has_te = 0;
    for (int i = 0; i < ctx->header_count; i++) {
        if (strcasecmp(ctx->header_keys[i], "content-length") == 0) has_cl = 1;
        if (strcasecmp(ctx->header_keys[i], "transfer-encoding") == 0) has_te = 1;
        if (strcasecmp(ctx->header_keys[i], "upgrade") == 0 && strcasecmp(ctx->header_vals[i], "h2c") == 0) {
            /* Block cleartext HTTP/2 tunneling */
            return -1;
        }
    }
    if (has_cl && has_te) {
        /* RFC 7230: If both are present, it's a smuggling attempt. Reject. */
        return -1;
    }
    return 0;
}

void wolf_engine_parse_request_simd(void* ctx_ptr, char* raw, size_t len) {
    WolfConnCtx* ctx = (WolfConnCtx*)ctx_ptr;
    WolfArena* a = ctx->arena;
    char* body_start = NULL;

    size_t j = 0;

#if defined(__x86_64__) && defined(__AVX2__)
    if (len >= 32) {
        __m256i cr = _mm256_set1_epi8('\r');

        /* 
         * Scan 32 bytes at a time for \r. 
         * To handle \r\n\r\n spanning chunks, we stop 3 bytes short of the end in AVX loop.
         */
        for (; j + 31 < len; j += 32) {
            __m256i chunk = _mm256_loadu_si256((const __m256i*)(raw + j));
            __m256i cmp_cr = _mm256_cmpeq_epi8(chunk, cr);
            uint32_t mask = (uint32_t)_mm256_movemask_epi8(cmp_cr);

            while (mask != 0) {
                int bit = __builtin_ctz(mask);
                size_t idx = j + bit;
                /* Since we only scan up to len - 32, idx + 3 is strictly < len */
                if (raw[idx+1] == '\n' && raw[idx+2] == '\r' && raw[idx+3] == '\n') {
                    raw[idx] = '\0';
                    body_start = raw + idx + 4;
                    goto found;
                }
                mask &= mask - 1; /* clear lowest set bit */
            }
        }
    }
#endif

    /* Scalar fallback / continuation for the remainder */
    for (; j + 3 < len; j++) {
        if (raw[j]=='\r' && raw[j+1]=='\n' && raw[j+2]=='\r' && raw[j+3]=='\n') {
            raw[j] = '\0';
            body_start = raw + j + 4;
            break;
        }
    }
    
found:

    ctx->body = wolf_arena_strdup(a, body_start ? body_start : "");

    /* Parse request line */
    char* saveptr;
    char* line = strtok_r(raw, "\r\n", &saveptr);
    if (!line) return;

    char* l_save;
    char* method    = strtok_r(line, " ", &l_save);
    char* full_path = strtok_r(NULL, " ", &l_save);

    if (method)    ctx->method = wolf_arena_strdup(a, method);
    if (full_path) {
        char* q = strchr(full_path, '?');
        if (q) {
            *q = '\0';
            ctx->path  = wolf_arena_strdup(a, full_path);
            ctx->query = wolf_arena_strdup(a, q + 1);
        } else {
            ctx->path  = wolf_arena_strdup(a, full_path);
            ctx->query = wolf_arena_strdup(a, "");
        }
    }

    /* Parse headers */
    const char* upgrade_val      = NULL;
    const char* ws_key_val       = NULL;
    const char* content_type_val = NULL;

    while ((line = strtok_r(NULL, "\r\n", &saveptr))) {
        char* colon = strchr(line, ':');
        if (colon && ctx->header_count < 32) {
            *colon = '\0';
            char* val = colon + 1;
            while (*val == ' ') val++;
            
            /* Lowercase header name in-place for fast case-insensitive access */
            for (char* p = line; *p; p++) {
                *p = tolower((unsigned char)*p);
            }

            ctx->header_keys[ctx->header_count] = wolf_arena_strdup(a, line);
            ctx->header_vals[ctx->header_count] = wolf_arena_strdup(a, val);
            
            if (strcmp(line, "upgrade") == 0)          upgrade_val      = ctx->header_vals[ctx->header_count];
            if (strcmp(line, "sec-websocket-key") == 0) ws_key_val       = ctx->header_vals[ctx->header_count];
            if (strcmp(line, "content-type") == 0)      content_type_val = ctx->header_vals[ctx->header_count];
            
            wolf_header_htab_insert(ctx->header_htab, ctx->header_count, line);
            ctx->header_count++;
        }
    }

    /* Smuggling defense validation */
    if (validate_crlf_and_smuggling(ctx) < 0) {
        ctx->status_code = 400; /* Bad Request */
        ctx->is_websocket = 0;
        return;
    }

    if (upgrade_val && strcasecmp(upgrade_val, "websocket") == 0 && ws_key_val) {
        ctx->is_websocket = 1;
        ctx->ws_key = wolf_arena_strdup(a, ws_key_val);
    }

    if (body_start && content_type_val && strstr(content_type_val, "multipart/form-data")) {
        size_t body_len = len - (body_start - raw);
        if (body_len > 0) {
            extern void wolf_engine_parse_multipart(WolfConnCtx* ctx, const char* content_type_val, char* body_start, size_t body_len);
            wolf_engine_parse_multipart(ctx, content_type_val, body_start, body_len);
        }
    }
}

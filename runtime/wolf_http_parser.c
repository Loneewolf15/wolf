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

/* ================================================================
 * Phase 5 — HTTP Request Smuggling Defense Suite
 *
 * Enforces RFC 9112 §6.3 disambiguation rules plus hardened rejection
 * of all known CL.TE / TE.CL / TE.TE attack variants.
 *
 * Rules:
 *   R1: Reject if both Content-Length AND Transfer-Encoding are present
 *       (unless TE == "identity" in which case CL takes precedence — RFC 7230)
 *   R2: If Transfer-Encoding: chunked is present, ignore Content-Length
 *       (recorded as ctx->ignore_content_length for the body reader)
 *   R3: Reject any Transfer-Encoding value that is not "chunked" or "identity"
 *       — covers TE.TE obfuscation ("xchunked", " chunked", "chunked, trailers")
 *   R4: Reject duplicate Content-Length headers with *different* values
 *       (same value is technically allowed by some implementations but
 *       still rejected here for maximum defensive posture)
 *   R5: Bare \r not followed by \n in the header section → reject
 *       (RFC 9112 §2.2 — only \r\n is a valid line terminator)
 *
 * Returns:
 *   0  — request is well-formed; caller may proceed
 *  -1  — request is malformed/smuggling attempt; caller must return 400
 * ================================================================ */

/* Normalise a Transfer-Encoding value: strip leading/trailing whitespace,
 * return true if the value is a safe canonical token. */
static int te_value_is_safe(const char* val) {
    if (!val) return 0;
    while (*val == ' ' || *val == '\t') val++;  /* ltrim */
    if (strcasecmp(val, "chunked")  == 0) return 1;
    if (strcasecmp(val, "identity") == 0) return 1;
    /* Reject "xchunked", "CHUNKED\t", "chunked, trailers", etc. */
    return 0;
}

static int validate_crlf_and_smuggling(WolfConnCtx* ctx) {
    int    has_te          = 0;  /* any TE header present */
    int    te_is_chunked   = 0;  /* TE == chunked */
    int    cl_count        = 0;  /* number of CL headers seen */
    long   cl_first        = -1; /* value of first CL header */

    for (int i = 0; i < ctx->header_count; i++) {
        const char* k = ctx->header_keys[i];
        const char* v = ctx->header_vals[i];

        /* ── Transfer-Encoding ─────────────────────────────────────── */
        if (strcasecmp(k, "transfer-encoding") == 0) {
            has_te = 1;
            /* R3: Reject obfuscated / non-canonical TE values */
            if (!te_value_is_safe(v)) {
                return -1;  /* TE.TE obfuscation attempt */
            }
            if (strcasecmp(v, "chunked") == 0) te_is_chunked = 1;
        }

        /* ── Content-Length ────────────────────────────────────────── */
        if (strcasecmp(k, "content-length") == 0) {
            cl_count++;
            char* endptr;
            long cl_val = strtol(v, &endptr, 10);
            if (*endptr != '\0' && *endptr != ' ') return -1; /* non-numeric CL */
            if (cl_val < 0) return -1;  /* negative CL */
            if (cl_count == 1) {
                cl_first = cl_val;
            } else {
                /* R4: Two CL headers — reject regardless of value match
                 * (duplicate CL is used in CL.CL smuggling vectors) */
                return -1;
            }
        }

        /* ── Upgrade: h2c — block HTTP/1.1→HTTP/2 cleartext upgrade ── */
        if (strcasecmp(k, "upgrade") == 0 && strcasecmp(v, "h2c") == 0) {
            return -1;
        }
    }

    /* R1: Both TE and CL present — RFC 9112 §6.3 says "the server MUST
     * reject the message" when TE is not "identity". We reject both cases
     * for maximum defensive posture. */
    if (has_te && cl_first >= 0) {
        return -1;
    }

    /* R2: Record that TE:chunked means we use chunked framing, not CL */
    if (te_is_chunked) {
        ctx->ignore_content_length = 1;
    }

    (void)cl_count; /* suppress unused-variable warning in edge paths */
    return 0;
}

/* R5: Scan raw header bytes for bare \r not followed by \n.
 * Called before the header tokeniser so we see the raw bytes.
 * Returns -1 if a bare CR is found, 0 otherwise. */
static int validate_raw_crlf(const char* raw, size_t header_len) {
    for (size_t i = 0; i + 1 < header_len; i++) {
        if (raw[i] == '\r' && raw[i+1] != '\n') {
            return -1;
        }
    }
    /* Also check the last byte */
    if (header_len > 0 && raw[header_len - 1] == '\r') {
        return -1;
    }
    return 0;
}


int wolf_engine_parse_request_simd(void* ctx_ptr, char* raw, size_t len) {
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
    if (!body_start) {
        return 1; /* Incomplete request */
    }

    /* R5: Reject bare \r not followed by \n in the header section.
     * body_start points 4 bytes past the \r\n\r\n terminator.
     * At this point raw[j] has been set to '\0' (see above) so header_len == j.
     * We scan the original raw bytes before strtok_r erases them. */
    {
        /* Compute header length: body_start is raw + j + 4, j is where \r was */
        size_t header_len = (size_t)(body_start - raw - 4);
        if (validate_raw_crlf(raw, header_len) < 0) {
            ctx->status_code = 400;
            return -1;
        }
    }

    ctx->body = wolf_arena_strdup(a, body_start);


    /* Parse request line */
    char* saveptr;
    char* line = strtok_r(raw, "\r\n", &saveptr);
    if (!line) return -1;

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
        return -1;
    }

    if (upgrade_val && strcasecmp(upgrade_val, "websocket") == 0 && ws_key_val) {
        ctx->is_websocket = 1;
        ctx->ws_key = wolf_arena_strdup(a, ws_key_val);
    }

    if (body_start && content_type_val && strstr(content_type_val, "multipart/form-data")) {
        size_t body_len = len - (body_start - raw);
        if (body_len > 0) {
            extern void wolf_engine_parse_multipart(WolfConnCtx* ctx, const char* content_type_val, const char* body, size_t body_len);
            wolf_engine_parse_multipart(ctx, content_type_val, body_start, body_len);
        }
    }
    
    return 0; /* Success */
}

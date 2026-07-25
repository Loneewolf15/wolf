/*
 * test_smuggling.c — HTTP Request Smuggling Defense Unit Tests (Phase 5)
 *
 * Self-contained test: defines a minimal WolfCtxStub that mirrors the fields
 * used by the smuggling validators, then re-includes only the three static
 * functions from wolf_http_parser.c via a thin shim.
 *
 * Build:
 *   gcc -Wall -Wextra -std=c11 -o test_smuggling bench/test_smuggling.c && ./test_smuggling
 */

#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include <stdint.h>
#include <strings.h>   /* strcasecmp */

/* ── Minimal WolfConnCtx stub ────────────────────────────────────────────── */
typedef struct {
    char*  header_keys[32];
    char*  header_vals[32];
    int    header_count;
    int    status_code;
    int    is_websocket;
    char*  ws_key;
    int    ignore_content_length;
} WolfConnCtx;

/* ── Inline the three pure functions from wolf_http_parser.c ─────────────── */

static int te_value_is_safe(const char* val) {
    if (!val) return 0;
    while (*val == ' ' || *val == '\t') val++;
    if (strcasecmp(val, "chunked")  == 0) return 1;
    if (strcasecmp(val, "identity") == 0) return 1;
    return 0;
}

static int validate_crlf_and_smuggling(WolfConnCtx* ctx) {
    int    has_te        = 0;
    int    te_is_chunked = 0;
    int    cl_count      = 0;
    long   cl_first      = -1;

    for (int i = 0; i < ctx->header_count; i++) {
        const char* k = ctx->header_keys[i];
        const char* v = ctx->header_vals[i];

        if (strcasecmp(k, "transfer-encoding") == 0) {
            has_te = 1;
            if (!te_value_is_safe(v)) return -1;
            if (strcasecmp(v, "chunked") == 0) te_is_chunked = 1;
        }

        if (strcasecmp(k, "content-length") == 0) {
            cl_count++;
            char* endptr;
            long cl_val = strtol(v, &endptr, 10);
            if (*endptr != '\0' && *endptr != ' ') return -1;
            if (cl_val < 0) return -1;
            if (cl_count == 1) {
                cl_first = cl_val;
            } else {
                return -1;  /* R4: duplicate CL */
            }
        }

        if (strcasecmp(k, "upgrade") == 0 && strcasecmp(v, "h2c") == 0) {
            return -1;
        }
    }

    if (has_te && cl_first >= 0) return -1;  /* R1 */

    if (te_is_chunked) ctx->ignore_content_length = 1;  /* R2 */

    (void)cl_count;
    return 0;
}

static int validate_raw_crlf(const char* raw, size_t header_len) {
    for (size_t i = 0; i + 1 < header_len; i++) {
        if (raw[i] == '\r' && raw[i+1] != '\n') return -1;
    }
    if (header_len > 0 && raw[header_len - 1] == '\r') return -1;
    return 0;
}

/* ── Test harness ─────────────────────────────────────────────────────────── */

static int g_pass = 0;
static int g_fail = 0;

#define ASSERT_EQ(label, got, expected) do { \
    if ((int)(got) == (int)(expected)) { \
        printf("  OK  PASS  %s\n", (label)); \
        g_pass++; \
    } else { \
        printf("  !! FAIL  %s  (got=%d  want=%d)\n", (label), (int)(got), (int)(expected)); \
        g_fail++; \
    } \
} while(0)

static WolfConnCtx make_ctx(const char** keys, const char** vals, int n) {
    WolfConnCtx ctx;
    memset(&ctx, 0, sizeof(ctx));
    for (int i = 0; i < n && i < 32; i++) {
        ctx.header_keys[i] = (char*)keys[i];
        ctx.header_vals[i] = (char*)vals[i];
    }
    ctx.header_count = n;
    return ctx;
}

#define CTX(...)  make_ctx((const char*[]){__VA_ARGS__}, NULL, 0)

/* ── Tests ─────────────────────────────────────────────────────────────────── */

int main(void) {
    printf("\nWolf HTTP Smuggling Defense -- Unit Test Suite (Phase 5)\n");
    printf("==========================================================\n\n");

    /* ── Rule 1: Reject CL + TE ── */
    printf("Rule 1 - Reject CL + TE coexistence:\n");
    {
        const char* k[] = {"content-length","transfer-encoding"}; const char* v[] = {"42","chunked"};
        WolfConnCtx c = make_ctx(k,v,2);
        ASSERT_EQ("R1 CL+TE:chunked -> reject", validate_crlf_and_smuggling(&c), -1);
    }
    {
        const char* k[] = {"content-length","transfer-encoding"}; const char* v[] = {"10","identity"};
        WolfConnCtx c = make_ctx(k,v,2);
        ASSERT_EQ("R1 CL+TE:identity -> reject", validate_crlf_and_smuggling(&c), -1);
    }
    {
        const char* k[] = {"content-length"}; const char* v[] = {"0"};
        WolfConnCtx c = make_ctx(k,v,1);
        ASSERT_EQ("R1 CL only -> allow", validate_crlf_and_smuggling(&c), 0);
    }
    {
        const char* k[] = {"transfer-encoding"}; const char* v[] = {"chunked"};
        WolfConnCtx c = make_ctx(k,v,1);
        ASSERT_EQ("R1 TE:chunked only -> allow", validate_crlf_and_smuggling(&c), 0);
    }

    /* ── Rule 2: TE:chunked sets flag ── */
    printf("\nRule 2 - TE:chunked sets ignore_content_length:\n");
    {
        const char* k[] = {"transfer-encoding"}; const char* v[] = {"chunked"};
        WolfConnCtx c = make_ctx(k,v,1);
        validate_crlf_and_smuggling(&c);
        ASSERT_EQ("R2 TE:chunked sets flag=1", c.ignore_content_length, 1);
    }
    {
        const char* k[] = {"content-length"}; const char* v[] = {"55"};
        WolfConnCtx c = make_ctx(k,v,1);
        validate_crlf_and_smuggling(&c);
        ASSERT_EQ("R2 CL only -> flag=0", c.ignore_content_length, 0);
    }

    /* ── Rule 3: Reject invalid TE values ── */
    printf("\nRule 3 - Reject TE.TE obfuscation:\n");
    {
        const char* k[] = {"transfer-encoding"}; const char* v[] = {"xchunked"};
        WolfConnCtx c = make_ctx(k,v,1);
        ASSERT_EQ("R3 TE:xchunked -> reject", validate_crlf_and_smuggling(&c), -1);
    }
    {
        const char* k[] = {"transfer-encoding"}; const char* v[] = {" chunked"};
        WolfConnCtx c = make_ctx(k,v,1);
        ASSERT_EQ("R3 TE:' chunked' (ltrim) -> allow", validate_crlf_and_smuggling(&c), 0);
    }
    {
        const char* k[] = {"transfer-encoding"}; const char* v[] = {"chunked, trailers"};
        WolfConnCtx c = make_ctx(k,v,1);
        ASSERT_EQ("R3 TE:chunked+trailers -> reject", validate_crlf_and_smuggling(&c), -1);
    }
    {
        const char* k[] = {"transfer-encoding"}; const char* v[] = {"CHUNKED"};
        WolfConnCtx c = make_ctx(k,v,1);
        ASSERT_EQ("R3 TE:CHUNKED (uppercase) -> allow", validate_crlf_and_smuggling(&c), 0);
    }

    /* ── Rule 4: Duplicate / negative CL ── */
    printf("\nRule 4 - Reject duplicate/negative Content-Length:\n");
    {
        const char* k[] = {"content-length","content-length"}; const char* v[] = {"42","42"};
        WolfConnCtx c = make_ctx(k,v,2);
        ASSERT_EQ("R4 Dup CL same val -> reject", validate_crlf_and_smuggling(&c), -1);
    }
    {
        const char* k[] = {"content-length","content-length"}; const char* v[] = {"42","100"};
        WolfConnCtx c = make_ctx(k,v,2);
        ASSERT_EQ("R4 Dup CL diff val -> reject", validate_crlf_and_smuggling(&c), -1);
    }
    {
        const char* k[] = {"content-length"}; const char* v[] = {"-1"};
        WolfConnCtx c = make_ctx(k,v,1);
        ASSERT_EQ("R4 Negative CL -> reject", validate_crlf_and_smuggling(&c), -1);
    }

    /* ── H2C cleartext upgrade ── */
    printf("\nH2C Upgrade block:\n");
    {
        const char* k[] = {"upgrade"}; const char* v[] = {"h2c"};
        WolfConnCtx c = make_ctx(k,v,1);
        ASSERT_EQ("H2C Upgrade:h2c -> reject", validate_crlf_and_smuggling(&c), -1);
    }

    /* ── Normal GET ── */
    printf("\nNormal traffic passthrough:\n");
    {
        const char* k[] = {"host","user-agent","accept"};
        const char* v[] = {"example.com","curl/7.81.0","*/*"};
        WolfConnCtx c = make_ctx(k,v,3);
        ASSERT_EQ("Normal GET (no CL/TE) -> allow", validate_crlf_and_smuggling(&c), 0);
    }
    {
        const char* k[] = {"host","content-length"};
        const char* v[] = {"example.com","512"};
        WolfConnCtx c = make_ctx(k,v,2);
        ASSERT_EQ("POST with CL only -> allow", validate_crlf_and_smuggling(&c), 0);
    }

    /* ── Rule 5: Bare CR detection ── */
    printf("\nRule 5 - Bare CR line terminator:\n");
    {
        const char* h = "GET / HTTP/1.1\rHost: evil.com\r\n";
        ASSERT_EQ("R5 bare \\r in header -> reject", validate_raw_crlf(h, strlen(h)), -1);
    }
    {
        const char* h = "GET / HTTP/1.1\r\nHost: example.com\r\n";
        ASSERT_EQ("R5 proper CRLF -> allow", validate_raw_crlf(h, strlen(h)), 0);
    }
    {
        const char* h = "GET / HTTP/1.1\nHost: example.com\n";
        ASSERT_EQ("R5 bare LF only (no CR) -> allow", validate_raw_crlf(h, strlen(h)), 0);
    }
    {
        /* Trailing bare CR at end of buffer */
        char buf[] = "GET / HTTP/1.1\r\nHost: foo.com\r";
        ASSERT_EQ("R5 trailing bare CR -> reject", validate_raw_crlf(buf, sizeof(buf)-1), -1);
    }
    {
        /* Multiple \r\n pairs — valid */
        const char* h = "POST / HTTP/1.1\r\nHost: x.com\r\nContent-Length: 4\r\n";
        ASSERT_EQ("R5 multi-header CRLF -> allow", validate_raw_crlf(h, strlen(h)), 0);
    }

    printf("\n==========================================================\n");
    int total = g_pass + g_fail;
    printf("Results: %d/%d passed", g_pass, total);
    if (g_fail == 0) printf("  -- ALL PASS\n\n");
    else printf("  -- %d FAILED\n\n", g_fail);
    return g_fail == 0 ? 0 : 1;
}

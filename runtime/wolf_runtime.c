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

#ifndef WOLF_RUNTIME_H
#define WOLF_RUNTIME_H

#include <stdint.h>

// --- Print ---
void wolf_print_str(const char* s);
void wolf_print_int(int64_t n);
void wolf_print_float(double f);
void wolf_print_bool(int b);
void wolf_print_nil(void);
void wolf_println(void);

// --- String Operations ---
const char* wolf_string_concat(const char* a, const char* b);
int64_t wolf_string_length(const char* s);
const char* wolf_string_upper(const char* s);
const char* wolf_string_lower(const char* s);
const char* wolf_string_trim(const char* s);

// --- Conversions ---
const char* wolf_int_to_string(int64_t n);
const char* wolf_float_to_string(double f);
const char* wolf_bool_to_string(int b);

// --- Memory ---
void* wolf_alloc(int64_t size);
void wolf_free(void* ptr);

#endif // WOLF_RUNTIME_H

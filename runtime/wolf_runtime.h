#ifndef WOLF_RUNTIME_H
#define WOLF_RUNTIME_H

#include <stdint.h>

// --- Output & Display (Wolf Way) ---
void wolf_say(const char* s);
void wolf_show(void* variable);
void wolf_inspect(void* variable);

// --- Print (Legacy/Internal) ---
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

// --- Data Structures ---
void* wolf_array_create();
void wolf_array_push(void* arr, void* item);
void* wolf_array_get(void* arr, int64_t index);
int64_t wolf_array_length(void* arr);

void* wolf_map_create();
void wolf_map_set(void* map, const char* key, void* value);
void* wolf_map_get(void* map, const char* key);
int wolf_env_has(const char* key);

void* wolf_class_create(const char* name);

// --- Time & Date ---
int64_t wolf_time_now();
const char* wolf_time_date(const char* format, int64_t timestamp);
int64_t wolf_time_strtotime(const char* datetime_string);

// --- System Utilities ---
void wolf_system_sleep(int64_t seconds);
void wolf_system_exit(int64_t code);
void wolf_system_die(const char* message);
const char* wolf_env_get(const char* key, const char* default_val);

// --- Sessions ---
void wolf_session_begin();
void wolf_session_set(const char* key, const char* value);
const char* wolf_session_get(const char* key);
void wolf_session_end();

// --- Math ---
double wolf_math_abs(double v);
double wolf_math_ceil(double v);
double wolf_math_floor(double v);
double wolf_math_max(double a, double b);
double wolf_math_min(double a, double b);

// --- Stdlib ---
int wolf_strings_contains(const char* s, const char* substr);
const char* wolf_strings_upper(const char* s);
const char* wolf_strings_split(const char* s, const char* sep);
const char* wolf_strings_join(const char* arr, const char* sep);
const char* wolf_json_encode(void* obj);

// --- Conversions ---
const char* wolf_int_to_string(int64_t n);
const char* wolf_float_to_string(double f);
const char* wolf_bool_to_string(int b);

// --- Memory ---
void* wolf_alloc(int64_t size);
void wolf_free(void* ptr);

// --- HTTP Server ---
typedef void (*wolf_http_handler_t)(int64_t req_id, int64_t res_id);

void wolf_http_serve(int64_t port, void* handler_ptr);

const char* wolf_http_req_method(int64_t req_id);
const char* wolf_http_req_path(int64_t req_id);
const char* wolf_http_req_query(int64_t req_id, const char* key);
const char* wolf_http_req_header(int64_t req_id, const char* key);
const char* wolf_http_req_body(int64_t req_id);

void wolf_http_res_header(int64_t res_id, const char* key, const char* value);
void wolf_http_res_status(int64_t res_id, int64_t status_code);
void wolf_http_res_write(int64_t res_id, const char* body);

#endif // WOLF_RUNTIME_H

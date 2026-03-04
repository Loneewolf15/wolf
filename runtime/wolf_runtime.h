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

// --- Define System (PHP-style constants) ---
void wolf_define(const char* key, const char* value);
int wolf_defined(const char* key);
const char* wolf_define_get(const char* key);

// --- Redis (In-Memory Mock / hiredis swap) ---
void* wolf_redis_connect(const char* host, int64_t port, const char* pass);
void wolf_redis_set(void* handle, const char* key, const char* value, int64_t ttl);
const char* wolf_redis_get(void* handle, const char* key);
int64_t wolf_redis_del(void* handle, const char* key);
int wolf_redis_exists(void* handle, const char* key);
void wolf_redis_hset(void* handle, const char* key, const char* field, const char* value);
const char* wolf_redis_hget(void* handle, const char* key, const char* field);
void wolf_redis_close(void* handle);

// --- Math ---
double wolf_math_abs(double v);
double wolf_math_ceil(double v);
double wolf_math_floor(double v);
double wolf_math_max(double a, double b);
double wolf_math_min(double a, double b);

// --- Trig ---
double wolf_math_sin(double v);
double wolf_math_cos(double v);
double wolf_math_tan(double v);
double wolf_math_asin(double v);
double wolf_math_acos(double v);
double wolf_math_atan(double v);
double wolf_math_atan2(double y, double x);

// --- Power / Root / Log ---
double wolf_math_sqrt(double v);
double wolf_math_pow(double base, double exp_val);
double wolf_math_log(double v);
double wolf_math_log10(double v);
double wolf_math_exp(double v);

// --- Rounding ---
double wolf_math_round(double v);
double wolf_math_fmod(double a, double b);

// --- Constants ---
double wolf_math_pi();

// --- Number Formatting ---
const char* wolf_number_format(double number, int64_t decimals, const char* dec_point, const char* thousands_sep);

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

// --- Phase 1 Stdlib: Strings ---
const char* wolf_strtoupper(const char* s);
const char* wolf_strtolower(const char* s);
const char* wolf_ucfirst(const char* s);
const char* wolf_ucwords(const char* s);
const char* wolf_lcfirst(const char* s);
const char* wolf_trim(const char* s);
const char* wolf_ltrim(const char* s);
const char* wolf_rtrim(const char* s);
int wolf_str_contains(const char* s, const char* sub);
int wolf_str_starts_with(const char* s, const char* prefix);
int wolf_str_ends_with(const char* s, const char* suffix);
const char* wolf_str_replace(const char* find, const char* rep, const char* s);
const char* wolf_str_repeat(const char* s, int64_t times);
const char* wolf_str_pad(const char* s, int64_t len, const char* pad);
const char* wolf_explode(const char* sep, const char* s);
const char* wolf_implode(const char* sep, const char* arr);
const char* wolf_substr(const char* s, int64_t start, int64_t len);
int64_t wolf_strpos(const char* s, const char* sub);
int64_t wolf_strrpos(const char* s, const char* sub);
int64_t wolf_str_word_count(const char* s);
int64_t wolf_strcmp(const char* a, const char* b);
const char* wolf_nl2br(const char* s);
const char* wolf_strip_tags(const char* s);
const char* wolf_htmlspecialchars(const char* s);
const char* wolf_addslashes(const char* s);
const char* wolf_stripslashes(const char* s);
const char* wolf_sprintf(const char* fmt, const char* arg1);

// --- Math Extras ---
double wolf_deg2rad(double deg);
double wolf_rad2deg(double rad);
double wolf_clamp(double n, double mn, double mx);

// --- Type Casting ---
int64_t wolf_intval(const char* s);
double wolf_floatval(const char* s);
const char* wolf_strval(int64_t n);
int wolf_boolval(const char* s);
int64_t wolf_intdiv(int64_t a, int64_t b);
const char* wolf_gettype(const char* val);
int wolf_is_numeric(const char* s);

// --- Encoding ---
const char* wolf_base64_encode(const char* s);
const char* wolf_base64_decode(const char* s);
const char* wolf_url_encode(const char* s);
const char* wolf_url_decode(const char* s);
const char* wolf_json_pretty(const char* json);
const char* wolf_json_decode(const char* json);

// --- Security ---
const char* wolf_md5(const char* s);
const char* wolf_sha256(const char* s);
const char* wolf_password_hash(const char* password);
int wolf_password_verify(const char* password, const char* hash);
const char* wolf_uuid_v4();
const char* wolf_rand_hex(int64_t length);

// --- Output ---
void wolf_dump(const char* val);
void wolf_dd(const char* val);
void wolf_log_info(const char* msg);
void wolf_log_warning(const char* msg);
void wolf_log_error(const char* msg);

// --- Phase 2 Stdlib: Arrays ---
int64_t wolf_count(void* a);
int wolf_in_array(const char* val, void* a);
int64_t wolf_array_search(const char* val, void* a);
void* wolf_array_pop(void* a);
void* wolf_array_shift(void* a);
void wolf_array_unshift(void* a, void* item);
void* wolf_array_reverse(void* a);
void* wolf_array_unique(void* a);
void* wolf_array_merge(void* a, void* b);
void* wolf_array_slice(void* a, int64_t offset, int64_t len);
void wolf_sort(void* a);
void wolf_rsort(void* a);
double wolf_array_sum(void* a);
void* wolf_array_keys(void* m);
void* wolf_array_values(void* m);
void* wolf_array_diff(void* a, void* b);
void* wolf_array_intersect(void* a, void* b);
void* wolf_array_flip(void* a);
void* wolf_range(int64_t start, int64_t end);

// --- Phase 3: Date/Time ---
int64_t wolf_time_ms();
int64_t wolf_time_ns();
int64_t wolf_mktime(int64_t hour, int64_t min, int64_t sec, int64_t mon, int64_t day, int64_t year);
int64_t wolf_date_diff(int64_t ts1, int64_t ts2);
const char* wolf_date_format(int64_t timestamp, const char* format);
int64_t wolf_day_of_week(int64_t timestamp);
int64_t wolf_days_in_month(int64_t month, int64_t year);
int wolf_is_leap_year(int64_t year);
int64_t wolf_strtotime(const char* str);

// --- Phase 3: Validation ---
int wolf_is_email(const char* s);
int wolf_is_url(const char* s);
int wolf_is_phone(const char* s);
int wolf_is_uuid(const char* s);
int wolf_is_json(const char* s);
int wolf_is_ip(const char* s);
int wolf_is_alpha(const char* s);
int wolf_is_alpha_num(const char* s);

// --- Phase 3: File System ---
int wolf_file_exists(const char* path);
const char* wolf_file_read(const char* path);
int wolf_file_write(const char* path, const char* data);
int wolf_file_append(const char* path, const char* data);
int wolf_file_delete(const char* path);
int64_t wolf_file_size(const char* path);
const char* wolf_file_extension(const char* path);
const char* wolf_file_basename(const char* path);
const char* wolf_file_dirname(const char* path);
int wolf_dir_exists(const char* path);

// --- Phase 3: Utilities ---
const char* wolf_slug(const char* s);
const char* wolf_truncate(const char* s, int64_t len, const char* suffix);

#endif // WOLF_RUNTIME_H

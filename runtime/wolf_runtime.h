#ifndef WOLF_RUNTIME_H
#define WOLF_RUNTIME_H

#include <stdint.h>
#include <stddef.h>  /* size_t */

/* ================================================================ *
 * wolf_runtime.h — Public API for the Wolf runtime library         *
 *                                                                  *
 * Compile with:                                                    *
 *   clang -c wolf_runtime.c -o wolf_runtime.o                     *
 *                                                                  *
 * Optional feature flags (pass as -D flags to clang):             *
 *   -DWOLF_REDIS_ENABLED   link against hiredis                   *
 *   -DWOLF_DB_POSTGRES     link against libpq instead of MySQL    *
 *   -DWOLF_DB_MSSQL        use MSSQL mock driver                  *
 *   -DWOLF_DEBUG           enable verbose pool/arena logging       *
 * ================================================================ */

/* --- Typed Value System --- */
#define WOLF_TYPE_STRING  0
#define WOLF_TYPE_INT     1
#define WOLF_TYPE_FLOAT   2
#define WOLF_TYPE_BOOL    3
#define WOLF_TYPE_NULL    4
#define WOLF_TYPE_MAP     5
#define WOLF_TYPE_ARRAY   6

#define WOLF_VALUE_MAGIC 0x574F4C46 /* "WOLF" */

typedef struct {
    uint32_t magic;
    int type;
    union {
        char*   s;
        int64_t i;
        double  f;
        int     b;
        void*   ptr;
    } val;
} wolf_value_t;

int wolf_is_tagged_value(void* ptr);

/* --- Per-Request Memory Arena --- */
void  wolf_req_arena_init(void);
void* wolf_req_alloc_register(void* ptr);
void* wolf_req_alloc(size_t sz);
char* wolf_req_strdup(const char* s);
void  wolf_req_arena_flush(void);

/* --- Output & Display (Wolf Way) --- */
void wolf_say(const char* s);
void wolf_show(void* variable);
void wolf_inspect(void* variable);

/* --- Print (Legacy/Internal) --- */
void wolf_print_str(const char* s);
void wolf_print_int(int64_t n);
void wolf_print_float(double f);
void wolf_print_bool(int b);
void wolf_print_nil(void);
void wolf_println(void);

/* --- String Operations --- */
const char* wolf_string_concat(const char* a, const char* b);
int64_t     wolf_string_length(const char* s);
const char* wolf_string_upper(const char* s);
const char* wolf_string_lower(const char* s);
const char* wolf_string_trim(const char* s);

/* --- Data Structures --- */
void*   wolf_array_create(void);
void    wolf_array_push(void* arr, void* item);
void*   wolf_array_get(void* arr, int64_t index);
int64_t wolf_array_length(void* arr);

void*   wolf_map_create(void);
void    wolf_map_set(void* map, const char* key, void* value);
void    wolf_map_set_int(void* map, const char* key, int64_t value);
void    wolf_map_set_float(void* map, const char* key, double value);
void    wolf_map_set_bool(void* map, const char* key, int value);
void*   wolf_map_get(void* map, const char* key);

/* --- Typed Value Constructors --- */
wolf_value_t* wolf_val_int(int64_t i);
wolf_value_t* wolf_val_float(double f);
wolf_value_t* wolf_val_bool(int b);
wolf_value_t* wolf_val_array(void* arr);

void* wolf_class_create(const char* name);

/* --- Time & Date --- */
int64_t     wolf_time_now(void);
const char* wolf_time_date(const char* format, int64_t timestamp);

/* --- System Utilities --- */
int64_t     wolf_argc(void);
const char* wolf_argv(int64_t index);
void        wolf_init_args(int argc, char** argv);
void        wolf_system_sleep(int64_t seconds);
void        wolf_system_exit(int64_t code);
void        wolf_system_die(const char* message);
const char* wolf_env_get(const char* key, const char* def_val); /* single declaration */
int         wolf_env_has(const char* key);

/* --- Sessions --- */
void        wolf_session_begin(void);
void        wolf_session_set(const char* key, const char* value);
const char* wolf_session_get(const char* key);
void        wolf_session_end(void);

/* --- Extended Strings --- */
const char* wolf_strings_title(const char* s);
const char* wolf_strings_trimleft(const char* s, const char* cutset);
const char* wolf_strings_trimright(const char* s, const char* cutset);

/* --- Define System (PHP-style constants) --- */
void        wolf_define(const char* key, const char* value);
int         wolf_defined(const char* key);
const char* wolf_define_get(const char* key);

/* --- Database --- */
void*   wolf_db_connect(const char* host, const char* user, const char* pass, const char* dbname);
void*   wolf_db_prepare(void* conn, const char* sql);
void    wolf_db_bind(void* stmt, const char* param, const char* value);
int64_t wolf_db_execute(void* stmt);
void*   wolf_db_fetch_all(void* stmt);
void*   wolf_db_fetch_one(void* stmt);
int64_t wolf_db_row_count(void* stmt);
int64_t wolf_db_last_insert_id(void* conn);
void    wolf_db_close(void* conn);
void    wolf_db_pool_destroy(void);        /* ← add this */
void    wolf_db_begin_transaction(void* conn);
void    wolf_db_commit(void* conn);
void    wolf_db_rollback(void* conn);

/* --- Redis ---
 * Always declared so the LLVM emitter can reference these symbols.
 * Without -DWOLF_REDIS_ENABLED, wolf_runtime.c provides no-op stubs
 * so linking always succeeds even when hiredis is not installed.       */
void*       wolf_redis_connect(const char* host, int64_t port, const char* pass);
void        wolf_redis_set(const char* key, const char* value, int64_t ttl);
const char* wolf_redis_get(const char* key);
int64_t     wolf_redis_del(const char* key);
int         wolf_redis_exists(const char* key);
void        wolf_redis_hset(const char* key, const char* field, const char* value);
const char* wolf_redis_hget(const char* key, const char* field);
void        wolf_redis_close(void);

/* --- Math --- */
double wolf_math_abs(double v);
double wolf_math_ceil(double v);
double wolf_math_floor(double v);
double wolf_math_max(double a, double b);
double wolf_math_min(double a, double b);
int64_t wolf_math_random(int64_t min, int64_t max);

/* --- Trig --- */
double wolf_math_sin(double v);
double wolf_math_cos(double v);
double wolf_math_tan(double v);
double wolf_math_asin(double v);
double wolf_math_acos(double v);
double wolf_math_atan(double v);
double wolf_math_atan2(double y, double x);

/* --- Power / Root / Log --- */
double wolf_math_sqrt(double v);
double wolf_math_pow(double base, double exp_val);
double wolf_math_log(double v);
double wolf_math_log10(double v);
double wolf_math_exp(double v);

/* --- Rounding --- */
double wolf_math_round(double v, int64_t precision);
double wolf_math_fmod(double a, double b);

/* --- Constants --- */
double wolf_math_pi(void);

/* --- Number Formatting --- */
const char* wolf_number_format(double number, int64_t decimals,
                               const char* dec_point, const char* thousands_sep);

/* --- Stdlib Strings & JSON --- */
int         wolf_strings_contains(const char* s, const char* substr);
const char* wolf_strings_upper(const char* s);
const char* wolf_strings_split(const char* s, const char* sep);
const char* wolf_strings_join(const char* arr, const char* sep);
const char* wolf_json_encode(void* obj);

/* --- Conversions --- */
const char* wolf_int_to_string(int64_t n);
const char* wolf_float_to_string(double f);
const char* wolf_bool_to_string(int b);

/* --- Memory --- */
void* wolf_alloc(int64_t size);
void  wolf_free(void* ptr);

/* --- HTTP Server --- */
typedef void (*wolf_http_handler_t)(int64_t req_id, int64_t res_id);

void        wolf_http_serve(int64_t port, void* handler_ptr);
const char* wolf_http_req_method(int64_t req_id);
const char* wolf_http_req_path(int64_t req_id);
const char* wolf_http_req_query(int64_t req_id, const char* key);
const char* wolf_http_req_header(int64_t req_id, const char* key);
const char* wolf_http_req_body(int64_t req_id);
void        wolf_http_res_header(int64_t res_id, const char* key, const char* value);
void        wolf_http_res_status(int64_t res_id, int64_t status_code);
void        wolf_http_res_write(int64_t res_id, const char* body);
/* --- File Uploads --- */
const char* wolf_http_req_file(int64_t req_id, const char* field_name);
int64_t     wolf_http_req_file_count(int64_t req_id);

/* --- Phase 1 Stdlib: Strings --- */
const char* wolf_strtoupper(const char* s);
const char* wolf_strtolower(const char* s);
const char* wolf_ucfirst(const char* s);
const char* wolf_ucwords(const char* s);
const char* wolf_lcfirst(const char* s);
const char* wolf_trim(const char* s);
const char* wolf_ltrim(const char* s);
const char* wolf_rtrim(const char* s);
int         wolf_str_contains(const char* s, const char* sub);
int         wolf_str_starts_with(const char* s, const char* prefix);
int         wolf_str_ends_with(const char* s, const char* suffix);
const char* wolf_str_replace(const char* find, const char* rep, const char* s);
const char* wolf_str_repeat(const char* s, int64_t times);
const char* wolf_str_pad(const char* s, int64_t len, const char* pad);
void*       wolf_explode(const char* sep, const char* s);
const char* wolf_implode(const char* sep, void* arr);
const char* wolf_substr(const char* s, int64_t start, int64_t len);
int64_t     wolf_strpos(const char* s, const char* sub);
int64_t     wolf_strrpos(const char* s, const char* sub);
int64_t     wolf_str_word_count(const char* s);
int64_t     wolf_strcmp(const char* a, const char* b);
const char* wolf_nl2br(const char* s);
const char* wolf_strip_tags(const char* s);
const char* wolf_htmlspecialchars(const char* s);
const char* wolf_addslashes(const char* s);
const char* wolf_stripslashes(const char* s);
const char* wolf_sprintf(const char* fmt, ...);

/* --- STDLIB-01: Additional String Functions --- */
const char* wolf_str_ireplace(const char* find, const char* rep, const char* s);
const char* wolf_htmlspecialchars_decode(const char* s);
double      wolf_similar_text(const char* a, const char* b);
const char* wolf_wordwrap(const char* s, int64_t width, const char* brk, int cut_long);
const char* wolf_quoted_printable_encode(const char* s);

/* --- STDLIB-01: Regex (POSIX ERE) --- */
int         wolf_preg_match(const char* pattern, const char* s);
const char* wolf_preg_match_captures(const char* pattern, const char* s);
int64_t     wolf_preg_match_all(const char* pattern, const char* s);
const char* wolf_preg_replace(const char* pattern, const char* rep, const char* s);
void*       wolf_preg_split(const char* pattern, const char* s);

/* --- Math Extras --- */
double wolf_deg2rad(double deg);
double wolf_rad2deg(double rad);
double wolf_clamp(double n, double mn, double mx);

/* --- Math Additions (STDLIB-03) --- */
double  wolf_rand_float(void);
int64_t wolf_rand_secure(int64_t min, int64_t max);
double  wolf_math_inf(void);
double  wolf_math_nan(void);
int     wolf_is_nan(double v);
int     wolf_is_inf(double v);
int     wolf_is_finite(double v);
double  wolf_math_log_base(double n, double base);
double  wolf_math_hypot(double a, double b);
 
/* --- Type Casting --- */
 
int64_t     wolf_intval(const char* s);
double      wolf_floatval(const char* s);
const char* wolf_strval(int64_t n);
int         wolf_boolval(const char* s);
int64_t     wolf_intdiv(int64_t a, int64_t b);
const char* wolf_gettype(const char* val);
int         wolf_is_numeric(const char* s);

/* --- Encoding --- */
const char* wolf_base64_encode(const char* s);
const char* wolf_base64_decode(const char* s);
const char* wolf_url_encode(const char* s);
const char* wolf_url_decode(const char* s);
const char* wolf_json_pretty(const char* json);
void*       wolf_json_decode(const char* json);

/* --- Security --- */
/* --- Crypto Init --- */
void wolf_crypto_init(void);
/* --- Hashing --- */
const char* wolf_md5(const char* s);
const char* wolf_sha256(const char* s);
const char* wolf_sha512(const char* s);
const char* wolf_hash(const char* algo, const char* s);
const char* wolf_hash_hmac(const char* algo, const char* data, const char* key);
int         wolf_hash_equals(const char* known, const char* user);

/* --- Password Hashing (Argon2id) --- */
const char* wolf_password_hash(const char* password);
int         wolf_password_verify(const char* password, const char* hash);
int         wolf_password_needs_rehash(const char* hash);

/* --- Symmetric Encryption (XSalsa20-Poly1305) --- */
const char* wolf_encrypt(const char* data, const char* key);
const char* wolf_decrypt(const char* data, const char* key);

/* --- Random / Token Generation --- */
const char* wolf_rand_bytes(int64_t length);
const char* wolf_rand_hex(int64_t length);
const char* wolf_rand_token(void);

/* --- ID Generation --- */
const char* wolf_uuid_v4(void);
const char* wolf_uuid_v7(void);
const char* wolf_nanoid(int64_t size);
const char* wolf_custom_id(const char* prefix, int64_t entropy);

/* --- Encoding --- */
const char* wolf_base64_url_encode(const char* s);
const char* wolf_base64_url_decode(const char* s);
const char* wolf_hex_encode(const char* s);
const char* wolf_hex_decode(const char* s);

/* --- JWT (HMAC-SHA256) --- */
const char* wolf_jwt_encode(const char* payload, const char* secret);
const char* wolf_jwt_encode_exp(const char* payload, const char* secret, int64_t expiry_sec);
const char* wolf_jwt_decode(const char* token, const char* secret);
const char* wolf_jwt_decode_unverified(const char* token);
int         wolf_jwt_expired(const char* token);

/* --- Curve25519 ECDH (Signal/WhatsApp key exchange primitive) --- */
const char* wolf_curve25519_keypair(void);
const char* wolf_curve25519_shared(const char* my_secret_hex, const char* their_public_hex);

/* --- RSA Sign / Verify --- */
const char* wolf_sign(const char* data, const char* privkey_pem);
int         wolf_verify(const char* data, const char* sig_b64, const char* pubkey_pem);

/* --- Output / Debugging --- */
void wolf_dump(const char* val);
void wolf_dd(const char* val);
void wolf_log_info(const char* msg);
void wolf_log_warning(const char* msg);
void wolf_log_error(const char* msg);

/* --- Phase 2 Stdlib: Arrays --- */
int64_t wolf_count(void* a);
int     wolf_in_array(const char* val, void* a);
int64_t wolf_array_search(const char* val, void* a);
void*   wolf_array_pop(void* a);
void*   wolf_array_shift(void* a);
void    wolf_array_unshift(void* a, void* item);
void*   wolf_array_reverse(void* a);
void*   wolf_array_unique(void* a);
void*   wolf_array_merge(void* a, void* b);
void*   wolf_array_slice(void* a, int64_t offset, int64_t len);
void    wolf_sort(void* a);
void    wolf_rsort(void* a);
double  wolf_array_sum(void* a);
void*   wolf_array_keys(void* m);
void*   wolf_array_values(void* m);
void*   wolf_array_diff(void* a, void* b);
void*   wolf_array_intersect(void* a, void* b);
void*   wolf_array_flip(void* a);
void*   wolf_range(int64_t start, int64_t end);

/* --- Phase 2 Stdlib: Additional Array Functions --- */
void*   wolf_array_fill(int64_t start, int64_t num, const char* value);
void*   wolf_array_combine(void* keys, void* values);
void*   wolf_array_chunk(void* a, int64_t size);
void*   wolf_array_column(void* a, const char* col);
double  wolf_array_product(void* a);
void*   wolf_array_diff_key(void* a, void* b);
void*   wolf_array_splice(void* a, int64_t offset, int64_t length);
void*   wolf_array_pad(void* a, int64_t size, const char* value);
void*   wolf_array_count_values(void* a);
int64_t wolf_array_rand_one(void* a);
double  wolf_array_mean(void* a);
double  wolf_array_median(void* a);
const char* wolf_array_mode(void* a);
double  wolf_array_variance(void* a);
double  wolf_array_std_dev(void* a);
double  wolf_array_percentile(void* a, double p);
 
/* --- Phase 3: Date/Time --- */
int64_t     wolf_time_ms(void);
int64_t     wolf_time_ns(void);
int64_t     wolf_mktime(int64_t hour, int64_t min, int64_t sec,
                        int64_t mon, int64_t day, int64_t year);
int64_t     wolf_date_diff(int64_t ts1, int64_t ts2);
const char* wolf_date_format(int64_t timestamp, const char* format);
int64_t     wolf_day_of_week(int64_t timestamp);
int64_t     wolf_days_in_month(int64_t month, int64_t year);
int64_t     wolf_is_leap_year(int64_t year);
int64_t     wolf_strtotime(const char* str);

/* --- STDLIB-04: Date Object --- */
int64_t     wolf_date_create(const char* str);
int64_t     wolf_date_add_days(int64_t ts, int64_t days);
int64_t     wolf_date_add_months(int64_t ts, int64_t months);
int64_t     wolf_date_diff_days(int64_t ts1, int64_t ts2);
int         wolf_date_is_past(int64_t ts);
int         wolf_date_is_future(int64_t ts);
const char* wolf_date_to_iso(int64_t ts);

/* --- Phase 3: Validation --- */
int wolf_is_email(const char* s);
int wolf_is_url(const char* s);
int wolf_is_phone(const char* s);
int wolf_is_uuid(const char* s);
int wolf_is_json(const char* s);
int wolf_is_ip(const char* s);
int wolf_is_alpha(const char* s);
int wolf_is_alpha_num(const char* s);

/* --- Phase 3: File System --- */
int         wolf_file_exists(const char* path);
const char* wolf_file_read(const char* path);
int         wolf_file_write(const char* path, const char* data);
int         wolf_file_save(const char* path, const char* b64_data);  /* Save binary upload to disk (b64 decode + binary write) */
int         wolf_file_append(const char* path, const char* data);
int         wolf_file_delete(const char* path);
int64_t     wolf_file_size(const char* path);
const char* wolf_file_extension(const char* path);
const char* wolf_file_basename(const char* path);
const char* wolf_file_dirname(const char* path);
int         wolf_dir_exists(const char* path);

/* --- Phase 3: Utilities --- */
const char* wolf_slug(const char* s);
const char* wolf_truncate(const char* s, int64_t len, const char* suffix);

/* --- Thread-Local Request Context --- */
void        wolf_set_current_context(void* req_id, void* res_id);
const char* wolf_get_request_body(void);
const char* wolf_get_request_header(const char* key);
const char* wolf_get_request_method(void);
const char* wolf_get_request_path(void);
const char* wolf_input(const char* key);
void        wolf_http_response_code(int64_t code);
void        wolf_http_write_response(const char* body);

/* --- Sanitization --- */
const char* wolf_sanitize_string(const char* s);
const char* wolf_sanitize_email(const char* s);
const char* wolf_sanitize_url(const char* s);
const char* wolf_sanitize_int(const char* s);
const char* wolf_sanitize_float(const char* s);

/* --- JWT --- */
const char* wolf_jwt_encode(const char* payload, const char* secret);
const char* wolf_jwt_decode(const char* token, const char* secret);

/* --- WebSocket --- */
void*       wolf_ws_on_message(void* handler);
void*       wolf_ws_send(int64_t req_id, const char* message);

/* --- Validation Rules Engine (STDLIB-08) --- */
void*       wolf_validate(void* data, void* rules);
int         wolf_validator_passes(void* v);
const char* wolf_validator_errors(void* v);
void*       wolf_validator_validated(void* v);

/* --- Query Builder (DB-01) --- */
void*   wolf_qb_create(void* conn, const char* table);
void*   wolf_qb_where(void* qb, const char* col, const char* val, const char* op);
void*   wolf_qb_order_by(void* qb, const char* col, const char* dir);
void*   wolf_qb_limit(void* qb, int64_t n);
void*   wolf_qb_offset(void* qb, int64_t n);
void*   wolf_qb_get(void* qb);
void*   wolf_qb_first(void* qb);
int64_t wolf_qb_insert(void* qb, void* data);
int64_t wolf_qb_update(void* qb, void* data);
int64_t wolf_qb_delete(void* qb);
void*   wolf_qb_paginate(void* qb, int64_t page, int64_t per_page);

#endif /* WOLF_RUNTIME_H */
/*
 * Wolf Runtime Library
 * Provides core functions that compiled Wolf programs link against.
 * Compiled with: clang -c wolf_runtime.c -o wolf_runtime.o
 */

#define _GNU_SOURCE
#define _POSIX_C_SOURCE 200809L

#include "wolf_runtime.h"
#include "wolf_config_runtime.h"
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
#include <time.h>
#include <math.h>
#include <sys/stat.h>
#include <errno.h>

// ========== Configurable Runtime Limits ==========
#ifndef WOLF_MAX_CONCURRENT_REQUESTS
#define WOLF_MAX_CONCURRENT_REQUESTS 1024
#endif
#define MAX_CONCURRENT_REQUESTS WOLF_MAX_CONCURRENT_REQUESTS

#ifndef WOLF_MAX_REQUEST_SIZE
#define WOLF_MAX_REQUEST_SIZE 65536
#endif

#ifndef WOLF_DEFINE_MAX
#define WOLF_DEFINE_MAX 256
#endif

#ifndef WOLF_REDIS_MAX
#define WOLF_REDIS_MAX 512
#endif
/* Forward declarations for HTTP context */
static __thread int64_t wolf_current_res_id;
void wolf_http_res_write(int64_t res_id, const char* body);

/* ================================================================ *
 * Per-Request Memory Arena                                         *
 *                                                                  *
 * All allocations that should live exactly as long as one HTTP     *
 * request register themselves here via wolf_req_alloc() /         *
 * wolf_req_strdup(). At end of request, wolf_req_arena_flush()    *
 * frees everything in O(n) — one call, no leaks.                  *
 *                                                                  *
 * Thread-safe: each worker thread has its own arena (__thread).   *
 * ================================================================ */

#define WOLF_ARENA_CHUNK 256  /* grow by this many entries at once */

typedef struct {
    void   **ptrs;      /* pointers registered for deferred free */
    int      count;     /* number of entries currently in use    */
    int      cap;       /* allocated capacity of ptrs[]          */
    int      active;    /* 1 while handling a request            */
} WolfReqArena;

static __thread WolfReqArena wolf_req_arena = {NULL, 0, 0, 0};

/* Call at start of every HTTP request (before running Wolf handler). */
void wolf_req_arena_init(void) {
    wolf_req_arena.count  = 0;  /* reset counters — reuse existing ptrs[] */
    wolf_req_arena.active = 1;
}

/* Register ptr for deferred free at end of request.
 * Safe to call with NULL (no-op). Returns ptr unchanged for chaining. */
void* wolf_req_alloc_register(void* ptr) {
    if (!ptr || !wolf_req_arena.active) return ptr;
    if (wolf_req_arena.count >= wolf_req_arena.cap) {
        int new_cap = wolf_req_arena.cap + WOLF_ARENA_CHUNK;
        void** new_ptrs = realloc(wolf_req_arena.ptrs, new_cap * sizeof(void*));
        if (!new_ptrs) return ptr;  /* on OOM — leak rather than crash */
        wolf_req_arena.ptrs = new_ptrs;
        wolf_req_arena.cap  = new_cap;
    }
    wolf_req_arena.ptrs[wolf_req_arena.count++] = ptr;
    return ptr;
}

/* Convenience: malloc + register + zero out (like calloc). */
void* wolf_req_alloc(size_t sz) {
    void* p = malloc(sz);
    if (p) memset(p, 0, sz);
    return wolf_req_alloc_register(p);
}

/* Convenience: strdup + register. */
char* wolf_req_strdup(const char* s) {
    if (!s) return NULL;
    char* p = strdup(s);
    return (char*)wolf_req_alloc_register(p);
}

/* Call at end of every HTTP request (after writing response). *
 * Frees every registered pointer and resets the arena.        */
void wolf_req_arena_flush(void) {
    wolf_req_arena.active = 0;
    for (int i = 0; i < wolf_req_arena.count; i++) {
        free(wolf_req_arena.ptrs[i]);
        wolf_req_arena.ptrs[i] = NULL;
    }
    wolf_req_arena.count = 0;
    /* Keep ptrs[] buffer alive for next request — avoids repeated realloc */
}

// ========== Print ==========

void wolf_print_str(const char* s) {
    if (!s) return;
    if (wolf_current_res_id >= 0) {
        wolf_http_res_write(wolf_current_res_id, s);
    } else {
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
    if (wolf_current_res_id < 0) {
        printf("\n");
    }
}

// --- Output & Display (Wolf Way) ---

void wolf_say(const char* s) {
    if (!s) return;
    if (wolf_current_res_id >= 0) {
        wolf_http_res_write(wolf_current_res_id, s);
    } else {
        printf("%s", s);
    }
}

void wolf_show(void* variable) {
    if (wolf_current_res_id >= 0) {
        if (variable) {
            char buf[32];
            snprintf(buf, sizeof(buf), "%p", variable);
            wolf_http_res_write(wolf_current_res_id, buf);
        } else {
            wolf_http_res_write(wolf_current_res_id, "nil");
        }
    } else {
        if (variable) printf("%p\n", variable);
        else printf("nil\n");
    }
}

void wolf_inspect(void* variable) {
    if (wolf_current_res_id >= 0) {
        char buf[64];
        if (variable) snprintf(buf, sizeof(buf), "[ptr] %p", variable);
        else snprintf(buf, sizeof(buf), "[nil] null");
        wolf_http_res_write(wolf_current_res_id, buf);
    } else {
        if (variable) printf("[ptr] %p\n", variable);
        else printf("[nil] null\n");
    }
}

// ========== String Operations ==========

const char* wolf_string_concat(const char* a, const char* b) {
    if (!a) a = "";
    if (!b) b = "";
    size_t la = strlen(a);
    size_t lb = strlen(b);
    char* result = (char*)wolf_req_alloc(la + lb + 1);
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
    char* result = (char*)wolf_req_alloc(len + 1);
    for (size_t i = 0; i < len; i++) {
        result[i] = (char)toupper((unsigned char)s[i]);
    }
    result[len] = '\0';
    return result;
}

const char* wolf_string_lower(const char* s) {
    if (!s) return "";
    size_t len = strlen(s);
    char* result = (char*)wolf_req_alloc(len + 1);
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
    char* result = (char*)wolf_req_alloc(len + 1);
    memcpy(result, s, len);
    result[len] = '\0';
    return result;
}

// ========== Math Operations ==========

double wolf_math_abs(double v) {
	return v < 0 ? -v : v;
}
double wolf_math_ceil(double v) {
	int64_t i = (int64_t)v;
	return (v > i) ? (double)(i + 1) : (double)i;
}
double wolf_math_floor(double v) {
	int64_t i = (int64_t)v;
	return (v < i) ? (double)(i - 1) : (double)i;
}
double wolf_math_max(double a, double b) {
	return a > b ? a : b;
}
double wolf_math_min(double a, double b) {
	return a < b ? a : b;
}
int64_t wolf_math_random(int64_t min, int64_t max) {
    srand(time(NULL));
    return min + rand() % ((max + 1) - min);
}

// --- Trigonometric ---
double wolf_math_sin(double v)  { return sin(v); }
double wolf_math_cos(double v)  { return cos(v); }
double wolf_math_tan(double v)  { return tan(v); }
double wolf_math_asin(double v) { return asin(v); }
double wolf_math_acos(double v) { return acos(v); }
double wolf_math_atan(double v) { return atan(v); }
double wolf_math_atan2(double y, double x) { return atan2(y, x); }

// --- Power / Root / Log ---
double wolf_math_sqrt(double v)           { return sqrt(v); }
double wolf_math_pow(double base, double exp_val) { return pow(base, exp_val); }
double wolf_math_log(double v)            { return log(v); }
double wolf_math_log10(double v)          { return log10(v); }
double wolf_math_exp(double v)            { return exp(v); }

// --- Rounding ---
double wolf_math_round(double v) { return round(v); }
double wolf_math_fmod(double a, double b) { return fmod(a, b); }

// --- Constants ---
double wolf_math_pi() { return 3.14159265358979323846; }

// --- Number Formatting (PHP-style) ---
const char* wolf_number_format(double number, int64_t decimals, const char* dec_point, const char* thousands_sep) {
    if (!dec_point) dec_point = ".";
    if (!thousands_sep) thousands_sep = ",";

    // Format the decimal part
    char fmt[32];
    snprintf(fmt, sizeof(fmt), "%%.%lldf", (long long)decimals);
    char raw[128];
    snprintf(raw, sizeof(raw), fmt, number < 0 ? -number : number);

    // Split into integer and decimal parts
    char* dot = strchr(raw, '.');
    int int_len = dot ? (int)(dot - raw) : (int)strlen(raw);

    // Count commas needed
    int commas = (int_len - 1) / 3;
    int result_len = (number < 0 ? 1 : 0) + int_len + commas + (dot ? (int)strlen(dec_point) + (int)strlen(dot + 1) : 0) + 1;
    char* result = (char*)wolf_req_alloc(result_len + 16); // extra safety
    char* p = result;

    if (number < 0) *p++ = '-';

    // Write integer part with thousands separators
    int sep_len = (int)strlen(thousands_sep);
    for (int i = 0; i < int_len; i++) {
        *p++ = raw[i];
        int remaining = int_len - i - 1;
        if (remaining > 0 && remaining % 3 == 0) {
            for (int s = 0; s < sep_len; s++) *p++ = thousands_sep[s];
        }
    }

    // Write decimal part
    if (dot && decimals > 0) {
        int dp_len = (int)strlen(dec_point);
        for (int s = 0; s < dp_len; s++) *p++ = dec_point[s];
        char* frac = dot + 1;
        for (int i = 0; i < (int)decimals && frac[i]; i++) *p++ = frac[i];
    }
    *p = '\0';
    return result;
}

// ========== Time & System ==========

int64_t wolf_time_now() {
    return (int64_t)time(NULL);
}

// --- Environment ---
const char* wolf_env_get(const char* key, const char* def_val) {
    const char* val = getenv(key);
    return val ? val : (def_val ? def_val : "");
}

const char* wolf_time_date(const char* format, int64_t timestamp) {
    time_t rawtime = (time_t)timestamp;
    struct tm *info = localtime(&rawtime);
    char* buf = (char*)wolf_req_alloc(256);
    strftime(buf, 256, format, info);
    return buf;
}

void wolf_system_sleep(int64_t seconds) {
    sleep((unsigned int)seconds);
}

void wolf_system_exit(int64_t code) {
    exit((int)code);
}

void wolf_system_die(const char* message) {
    if (message) printf("%s\n", message);
    exit(1);
}

// ========== Sessions ==========

void wolf_session_begin() {
    // LLVM FFI Stub for Sessions
}
void wolf_session_set(const char* key, const char* value) {
    // LLVM FFI Stub
}
const char* wolf_session_get(const char* key) {
    return ""; // LLVM FFI Stub
}
void wolf_session_end() {
    // LLVM FFI Stub
}

// ========== Define System (PHP-style constants) ==========



static struct {
    char* keys[WOLF_DEFINE_MAX];
    char* values[WOLF_DEFINE_MAX];
    int count;
} wolf_defines = { .count = 0 };

void wolf_define(const char* key, const char* value) {
    if (!key || wolf_defines.count >= WOLF_DEFINE_MAX) return;
    // Check if already defined (constants are immutable)
    for (int i = 0; i < wolf_defines.count; i++) {
        if (strcmp(wolf_defines.keys[i], key) == 0) return;
    }
    wolf_defines.keys[wolf_defines.count] = wolf_req_strdup(key);
    wolf_defines.values[wolf_defines.count] = value ? wolf_req_strdup(value) : wolf_req_strdup("");
    wolf_defines.count++;
}

int wolf_defined(const char* key) {
    if (!key) return 0;
    for (int i = 0; i < wolf_defines.count; i++) {
        if (strcmp(wolf_defines.keys[i], key) == 0) return 1;
    }
    return 0;
}

const char* wolf_define_get(const char* key) {
    if (!key) return "";
    for (int i = 0; i < wolf_defines.count; i++) {
        if (strcmp(wolf_defines.keys[i], key) == 0) return wolf_defines.values[i];
    }
    return "";
}

// ========== Database — Connection Pool ==========

#if defined(WOLF_DB_POSTGRES)
#include <libpq-fe.h>
typedef PGconn WolfDBConn;
typedef PGresult WolfDBRes;
#define WOLF_DB_PING(conn) (PQstatus(conn) == CONNECTION_OK ? 0 : 1)
#define WOLF_DB_CLOSE(conn) PQfinish(conn)
#elif defined(WOLF_DB_MSSQL)
/* Mock MSSQL */
typedef void WolfDBConn;
typedef void WolfDBRes;
#define WOLF_DB_PING(conn) 0
#define WOLF_DB_CLOSE(conn) do {} while(0)
#else
#include <mysql/mysql.h>
typedef MYSQL WolfDBConn;
typedef MYSQL_RES WolfDBRes;
#define WOLF_DB_PING(conn) mysql_ping(conn)
#define WOLF_DB_CLOSE(conn) mysql_close(conn)
#endif

/* ------------------------------------------------------------------ *
 * Pool internals — all touched under wolf_pool_mutex                 *
 * ------------------------------------------------------------------ */
typedef struct {
    WolfDBConn *conn;     /* DB handle; NULL = slot empty / failed init */
    int         in_use;   /* 1 = checked out to a worker thread            */
} WolfPoolSlot;

static WolfPoolSlot    wolf_pool[WOLF_DB_POOL_SIZE];
static int             wolf_pool_inited  = 0;
static pthread_mutex_t wolf_pool_mutex   = PTHREAD_MUTEX_INITIALIZER;
static pthread_cond_t  wolf_pool_cond    = PTHREAD_COND_INITIALIZER;

/* Saved credentials for reconnect — set on first wolf_db_connect() */
static char *wolf_pool_host   = NULL;
static char *wolf_pool_user   = NULL;
static char *wolf_pool_pass   = NULL;
static char *wolf_pool_dbname = NULL;



typedef struct {
    WolfDBConn *conn;
    char       *sql;
    WolfDBRes  *last_result;
} WolfDBStmt;

/* --- Internal: open one DB connection ---- */
static WolfDBConn* wolf_pool_open_one(void) {
    const char *host   = wolf_pool_host   ? wolf_pool_host   : WOLF_DB_HOST;
    const char *user   = wolf_pool_user   ? wolf_pool_user   : WOLF_DB_USER;
    const char *pass   = wolf_pool_pass   ? wolf_pool_pass   : WOLF_DB_PASS;
    const char *dbname = wolf_pool_dbname ? wolf_pool_dbname : WOLF_DB_NAME;

#if defined(WOLF_DB_POSTGRES)
    char conninfo[512];
    snprintf(conninfo, sizeof(conninfo), "host=%s port=%d dbname=%s user=%s password=%s",
             host && *host ? host : "localhost", WOLF_DB_PORT, 
             dbname ? dbname : "", 
             user ? user : "", 
             pass ? pass : "");
    WolfDBConn *conn = PQconnectdb(conninfo);
    if (PQstatus(conn) != CONNECTION_OK) {
        fprintf(stderr, "[WOLF-POOL] PG connect failed: %s\n", PQerrorMessage(conn));
        PQfinish(conn);
        return NULL;
    }
    return conn;

#elif defined(WOLF_DB_MSSQL)
    return (WolfDBConn*)1;

#else
    MYSQL *conn = mysql_init(NULL);
    if (!conn) return NULL;

    if (host && (strcmp(host, "localhost") == 0 || strcmp(host, "") == 0)) {
        const char *sockets[] = {
            "/opt/lampp/var/mysql/mysql.sock",
            "/var/run/mysqld/mysqld.sock",
            "/tmp/mysql.sock",
            "/var/lib/mysql/mysql.sock",
            NULL
        };
        for (int i = 0; sockets[i]; i++) {
            if (access(sockets[i], F_OK) == 0) {
                if (mysql_real_connect(conn, NULL, user, pass, dbname,
                                       0, sockets[i], 0)) {
                    mysql_set_character_set(conn, "utf8mb4");
                    return conn;
                }
                mysql_close(conn);
                conn = mysql_init(NULL);
                if (!conn) return NULL;
            }
        }
        /* Socket failed — fall through to TCP 127.0.0.1 */
        if (mysql_real_connect(conn, "127.0.0.1", user, pass, dbname,
                               WOLF_DB_PORT, NULL, 0)) {
            mysql_set_character_set(conn, "utf8mb4");
            return conn;
        }
    } else {
        if (mysql_real_connect(conn, host, user, pass, dbname,
                               WOLF_DB_PORT, NULL, 0)) {
            mysql_set_character_set(conn, "utf8mb4");
            return conn;
        }
    }

    fprintf(stderr, "[WOLF-POOL] MySQL connect failed: %s\n", mysql_error(conn));
    mysql_close(conn);
    return NULL;
#endif
}

/* --- Init the pool (called once from wolf_db_connect or wolf_http_serve) */
static void wolf_pool_init_locked(void) {
    /* Already done */
    if (wolf_pool_inited) return;

    fprintf(stderr, "[WOLF-POOL] Initializing pool (size=%d) host=%s db=%s\n",
            WOLF_DB_POOL_SIZE,
            wolf_pool_host ? wolf_pool_host : WOLF_DB_HOST,
            wolf_pool_dbname ? wolf_pool_dbname : WOLF_DB_NAME);

    for (int i = 0; i < WOLF_DB_POOL_SIZE; i++) {
        wolf_pool[i].in_use = 0;
        wolf_pool[i].conn   = wolf_pool_open_one();
        if (wolf_pool[i].conn) {
            fprintf(stderr, "[WOLF-POOL] slot %d OK\n", i);
        } else {
            fprintf(stderr, "[WOLF-POOL] slot %d FAILED (will retry on acquire)\n", i);
        }
    }
    wolf_pool_inited = 1;
}

/* --- Acquire a connection from the pool (blocks up to WOLF_DB_POOL_TIMEOUT s) */
static WolfDBConn* wolf_pool_acquire(void) {
    struct timespec deadline;
    clock_gettime(CLOCK_REALTIME, &deadline);
    deadline.tv_sec += WOLF_DB_POOL_TIMEOUT;

    pthread_mutex_lock(&wolf_pool_mutex);
    wolf_pool_init_locked(); /* idempotent */

    while (1) {
        /* Look for a free, live slot */
        for (int i = 0; i < WOLF_DB_POOL_SIZE; i++) {
            if (!wolf_pool[i].in_use) {
                /* Health-check — reconnect if stale */
                if (wolf_pool[i].conn && WOLF_DB_PING(wolf_pool[i].conn) != 0) {
                    fprintf(stderr, "[WOLF-POOL] slot %d stale, reconnecting\n", i);
                    WOLF_DB_CLOSE(wolf_pool[i].conn);
                    wolf_pool[i].conn = wolf_pool_open_one();
                }
                /* If slot was NULL (failed init), try now */
                if (!wolf_pool[i].conn) {
                    wolf_pool[i].conn = wolf_pool_open_one();
                }
                if (wolf_pool[i].conn) {
                    wolf_pool[i].in_use = 1;
                    fprintf(stderr, "[WOLF-POOL] acquire slot=%d\n", i);
                    pthread_mutex_unlock(&wolf_pool_mutex);
                    return wolf_pool[i].conn;
                }
            }
        }
        /* All busy — wait for a release signal */
        int rc = pthread_cond_timedwait(&wolf_pool_cond, &wolf_pool_mutex, &deadline);
        if (rc == ETIMEDOUT) {
            fprintf(stderr, "[WOLF-POOL] timeout waiting for free slot (%ds)\n",
                    WOLF_DB_POOL_TIMEOUT);
            pthread_mutex_unlock(&wolf_pool_mutex);
            return NULL;
        }
    }
}

/* --- Release a connection back to the pool */
static void wolf_pool_release(WolfDBConn *conn) {
    if (!conn) return;
    pthread_mutex_lock(&wolf_pool_mutex);
    for (int i = 0; i < WOLF_DB_POOL_SIZE; i++) {
        if (wolf_pool[i].conn == conn) {
            wolf_pool[i].in_use = 0;
            fprintf(stderr, "[WOLF-POOL] release slot=%d\n", i);
            pthread_cond_signal(&wolf_pool_cond);
            break;
        }
    }
    pthread_mutex_unlock(&wolf_pool_mutex);
}

/* Public: wolf Wolf source calls db_connect(host,user,pass,db) —
 * first call seeds the credentials and starts the pool;
 * every subsequent call just checks out a pooled connection.     */
void* wolf_db_connect(const char* host, const char* user,
                      const char* pass, const char* dbname) {
    pthread_mutex_lock(&wolf_pool_mutex);
    /* Seed credentials on first call (from wolf source like Database.wolf) */
    if (!wolf_pool_host && host && *host) {
        wolf_pool_host   = wolf_req_strdup(host);
        wolf_pool_user   = user   && *user   ? wolf_req_strdup(user)   : NULL;
        wolf_pool_pass   = pass   && *pass   ? wolf_req_strdup(pass)   : NULL;
        wolf_pool_dbname = dbname && *dbname ? wolf_req_strdup(dbname) : NULL;
    }
    pthread_mutex_unlock(&wolf_pool_mutex);

    WolfDBConn *conn = wolf_pool_acquire();
    if (!conn) {
        fprintf(stderr, "[WOLF-POOL] failed to acquire connection\n");
    }
    return (void*)conn;
}

// Helper for simple string replace (used by wolf_db_bind)
static char* wolf_internal_str_replace(const char* orig, const char* rep,
                                        const char* with) {
    char *result, *ins, *tmp;
    int len_rep, len_with, len_front, count;

    if (!orig || !rep) return NULL;
    len_rep = strlen(rep);
    if (len_rep == 0) return NULL;
    if (!with) with = "";
    len_with = strlen(with);

    ins = (char *)orig;
    for (count = 0; (tmp = strstr(ins, rep)); ++count)
        ins = tmp + len_rep;

    tmp = result = wolf_req_alloc(strlen(orig) + (len_with - len_rep) * count + 1);
    if (!result) return NULL;

    while (count--) {
        ins = strstr(orig, rep);
        len_front = ins - orig;
        tmp = strncpy(tmp, orig, len_front) + len_front;
        tmp = strcpy(tmp, with) + len_with;
        orig += len_front + len_rep;
    }
    strcpy(tmp, orig);
    return result;
}


void* wolf_db_prepare(void* conn, const char* sql) {
    if (!conn) return NULL;
    WolfDBStmt* stmt = (WolfDBStmt*)wolf_req_alloc(sizeof(WolfDBStmt));
    stmt->conn = (WolfDBConn*)conn;
    stmt->sql = wolf_req_strdup(sql ? sql : "");
    stmt->last_result = NULL;
    return stmt;
}

void wolf_db_bind(void* stmt_ptr, const char* param, const char* value) {
    if (!stmt_ptr || !param) return;
    WolfDBStmt* stmt = (WolfDBStmt*)stmt_ptr;
    
    if (!value) value = "";
    
#if defined(WOLF_DB_POSTGRES)
    char *escaped = wolf_req_alloc(strlen(value) * 2 + 1);
    int err;
    PQescapeStringConn(stmt->conn, escaped, value, strlen(value), &err);
#elif defined(WOLF_DB_MSSQL)
    char *escaped = wolf_req_alloc(strlen(value) * 2 + 1);
    strcpy(escaped, value);
#else
    size_t val_len = strlen(value);
    char *escaped = wolf_req_alloc(val_len * 2 + 1);
    mysql_real_escape_string(stmt->conn, escaped, value, val_len);
#endif
    
    // Add quotes around the escaped value
    char *quoted = wolf_req_alloc(strlen(escaped) + 3);
    sprintf(quoted, "'%s'", escaped);
    
    char *new_sql = wolf_internal_str_replace(stmt->sql, param, quoted);
    if (new_sql) {
        // Omitting free() because pointers are managed by wolf_req_arena
        stmt->sql = new_sql;
    }
}

int64_t wolf_db_execute(void* stmt_ptr) {
    if (!stmt_ptr) return 0;
    WolfDBStmt* stmt = (WolfDBStmt*)stmt_ptr;
    
#if defined(WOLF_DB_POSTGRES)
    if (stmt->last_result) {
        PQclear(stmt->last_result);
        stmt->last_result = NULL;
    }
    WolfDBRes *res = PQexec(stmt->conn, stmt->sql);
    if (PQresultStatus(res) != PGRES_COMMAND_OK && PQresultStatus(res) != PGRES_TUPLES_OK) {
        printf("[WOLF-DB] PG Query failed: %s\n", PQerrorMessage(stmt->conn));
        PQclear(res);
        return 0;
    }
    stmt->last_result = res;
    return 1;
#elif defined(WOLF_DB_MSSQL)
    return 1;
#else
    if (stmt->last_result) {
        mysql_free_result(stmt->last_result);
        stmt->last_result = NULL;
    }
    
    if (mysql_query(stmt->conn, stmt->sql)) {
        printf("[WOLF-DB] MySQL Query failed: %s\n", mysql_error(stmt->conn));
        return 0; // Failure
    }
    
    stmt->last_result = mysql_store_result(stmt->conn);
    return 1; // Success
#endif
}

void* wolf_db_fetch_all(void* stmt_ptr) {
    void* arr = wolf_array_create();
    if (!stmt_ptr) return arr;
    WolfDBStmt* stmt = (WolfDBStmt*)stmt_ptr;
    if (!stmt->last_result) return arr;
    
#if defined(WOLF_DB_POSTGRES)
    WolfDBRes *res = stmt->last_result;
    int num_fields = PQnfields(res);
    int num_rows = PQntuples(res);
    
    for (int r = 0; r < num_rows; r++) {
        void *row = wolf_map_create();
        for (int i = 0; i < num_fields; i++) {
            if (PQgetisnull(res, r, i)) {
                wolf_map_set(row, PQfname(res, i), NULL);
            } else {
                char *val = wolf_req_strdup(PQgetvalue(res, r, i));
                wolf_map_set(row, PQfname(res, i), val);
            }
        }
        wolf_array_push(arr, row);
    }
    return arr;
    
#elif defined(WOLF_DB_MSSQL)
    return arr;
    
#else
    int num_fields = mysql_num_fields(stmt->last_result);
    MYSQL_FIELD *fields = mysql_fetch_fields(stmt->last_result);
    
    MYSQL_ROW row_data;
    while ((row_data = mysql_fetch_row(stmt->last_result))) {
        unsigned long *lengths = mysql_fetch_lengths(stmt->last_result);
        void *row = wolf_map_create();
        
        for (int i = 0; i < num_fields; i++) {
            if (row_data[i]) {
                char *val = wolf_req_alloc(lengths[i] + 1);
                memcpy(val, row_data[i], lengths[i]);
                val[lengths[i]] = '\0';
                wolf_map_set(row, fields[i].name, val);
            } else {
                wolf_map_set(row, fields[i].name, NULL);
            }
        }
        wolf_array_push(arr, row);
    }
    return arr;
#endif
}

void* wolf_db_fetch_one(void* stmt_ptr) {
    if (!stmt_ptr) return wolf_map_create();
    WolfDBStmt* stmt = (WolfDBStmt*)stmt_ptr;
    if (!stmt->last_result) return wolf_map_create();
    
#if defined(WOLF_DB_POSTGRES)
    WolfDBRes *res = stmt->last_result;
    if (PQntuples(res) > 0) {
        int num_fields = PQnfields(res);
        void *row = wolf_map_create();
        for (int i = 0; i < num_fields; i++) {
            if (PQgetisnull(res, 0, i)) {
                wolf_map_set(row, PQfname(res, i), NULL);
            } else {
                char *val = wolf_req_strdup(PQgetvalue(res, 0, i));
                wolf_map_set(row, PQfname(res, i), val);
            }
        }
        return row;
    }
    return wolf_map_create();
    
#elif defined(WOLF_DB_MSSQL)
    return wolf_map_create();
    
#else
    int num_fields = mysql_num_fields(stmt->last_result);
    MYSQL_FIELD *fields = mysql_fetch_fields(stmt->last_result);
    
    MYSQL_ROW row_data = mysql_fetch_row(stmt->last_result);
    if (row_data) {
        unsigned long *lengths = mysql_fetch_lengths(stmt->last_result);
        void *row = wolf_map_create();
        for (int i = 0; i < num_fields; i++) {
            if (row_data[i]) {
                char *val = wolf_req_alloc(lengths[i] + 1);
                memcpy(val, row_data[i], lengths[i]);
                val[lengths[i]] = '\0';
                wolf_map_set(row, fields[i].name, val);
            } else {
                wolf_map_set(row, fields[i].name, NULL);
            }
        }
        return row;
    }
    return wolf_map_create();
#endif
}

int64_t wolf_db_row_count(void* stmt_ptr) {
    if (!stmt_ptr) return 0;
    WolfDBStmt* stmt = (WolfDBStmt*)stmt_ptr;
#if defined(WOLF_DB_POSTGRES)
    if (stmt->last_result) {
        return (int64_t)PQntuples(stmt->last_result);
    }
    return 0; // PQcmdTuples might be needed for affected rows, but sticking to result sets for now
#elif defined(WOLF_DB_MSSQL)
    return 0;
#else
    if (stmt->last_result) {
        return (int64_t)mysql_num_rows(stmt->last_result);
    }
    return (int64_t)mysql_affected_rows(stmt->conn);
#endif
}

int64_t wolf_db_last_insert_id(void* conn_ptr) {
    if (!conn_ptr) return 0;
#if defined(WOLF_DB_POSTGRES)
    return 0; // Postgres uses RETURNING id instead
#elif defined(WOLF_DB_MSSQL)
    return 0;
#else
    return (int64_t)mysql_insert_id((WolfDBConn*)conn_ptr);
#endif
}

void wolf_db_close(void* conn_ptr) {
    if (!conn_ptr) return;
    wolf_pool_release((WolfDBConn*)conn_ptr);
}

void wolf_db_begin_transaction(void* conn_ptr) {
    if (!conn_ptr) return;
#if defined(WOLF_DB_POSTGRES)
    PQexec((WolfDBConn*)conn_ptr, "BEGIN");
#elif defined(WOLF_DB_MSSQL)
#else
    mysql_query((WolfDBConn*)conn_ptr, "START TRANSACTION");
#endif
}

void wolf_db_commit(void* conn_ptr) {
    if (!conn_ptr) return;
#if defined(WOLF_DB_POSTGRES)
    PQexec((WolfDBConn*)conn_ptr, "COMMIT");
#elif defined(WOLF_DB_MSSQL)
#else
    mysql_query((WolfDBConn*)conn_ptr, "COMMIT");
#endif
}

void wolf_db_rollback(void* conn_ptr) {
    if (!conn_ptr) return;
#if defined(WOLF_DB_POSTGRES)
    PQexec((WolfDBConn*)conn_ptr, "ROLLBACK");
#elif defined(WOLF_DB_MSSQL)
#else
    mysql_query((WolfDBConn*)conn_ptr, "ROLLBACK");
#endif
}

// ========== Redis (In-Memory Mock) ==========

// Production: swap with hiredis calls. Mock lets Wolf code compile & run without a live Redis.


#include <hiredis/hiredis.h>

static __thread redisContext* wolf_redis_ctx = NULL;

void* wolf_redis_connect(const char* host, int64_t port, const char* pass) {
    if (wolf_redis_ctx) {
        return wolf_redis_ctx;
    }
    
    struct timeval tv = {1, 500000}; // 1.5 seconds timeout
    wolf_redis_ctx = redisConnectWithTimeout(host, port, tv);
    
    if (wolf_redis_ctx == NULL || wolf_redis_ctx->err) {
        if (wolf_redis_ctx) {
            fprintf(stderr, "[WOLF-REDIS] Connection error: %s\n", wolf_redis_ctx->errstr);
            redisFree(wolf_redis_ctx);
            wolf_redis_ctx = NULL;
        } else {
            fprintf(stderr, "[WOLF-REDIS] Connection error: can't allocate redis context\n");
        }
        return NULL;
    }
    
    if (pass && strlen(pass) > 0) {
        redisReply *reply = redisCommand(wolf_redis_ctx, "AUTH %s", pass);
        if (reply) freeReplyObject(reply);
    }
    
    return wolf_redis_ctx;
}

void wolf_redis_set(void* handle, const char* key, const char* value, int64_t ttl) {
    redisContext* c = (redisContext*)handle;
    if (!c || !key) return;
    
    redisReply *reply;
    if (ttl > 0) {
        reply = redisCommand(c, "SET %s %s EX %lld", key, value ? value : "", (long long)ttl);
    } else {
        reply = redisCommand(c, "SET %s %s", key, value ? value : "");
    }
    if (reply) freeReplyObject(reply);
}

const char* wolf_redis_get(void* handle, const char* key) {
    redisContext* c = (redisContext*)handle;
    if (!c || !key) return "";
    
    redisReply *reply = redisCommand(c, "GET %s", key);
    if (reply && reply->type == REDIS_REPLY_STRING) {
        char *str = wolf_req_strdup(reply->str);
        freeReplyObject(reply);
        return str;
    }
    if (reply) freeReplyObject(reply);
    return "";
}

int64_t wolf_redis_del(void* handle, const char* key) {
    redisContext* c = (redisContext*)handle;
    if (!c || !key) return 0;
    
    redisReply *reply = redisCommand(c, "DEL %s", key);
    int64_t result = 0;
    if (reply && reply->type == REDIS_REPLY_INTEGER) {
        result = reply->integer;
    }
    if (reply) freeReplyObject(reply);
    return result;
}

int wolf_redis_exists(void* handle, const char* key) {
    redisContext* c = (redisContext*)handle;
    if (!c || !key) return 0;
    
    redisReply *reply = redisCommand(c, "EXISTS %s", key);
    int result = 0;
    if (reply && reply->type == REDIS_REPLY_INTEGER) {
        result = reply->integer > 0 ? 1 : 0;
    }
    if (reply) freeReplyObject(reply);
    return result;
}

void wolf_redis_hset(void* handle, const char* key, const char* field, const char* value) {
    redisContext* c = (redisContext*)handle;
    if (!c || !key || !field) return;
    
    redisReply *reply = redisCommand(c, "HSET %s %s %s", key, field, value ? value : "");
    if (reply) freeReplyObject(reply);
}

const char* wolf_redis_hget(void* handle, const char* key, const char* field) {
    redisContext* c = (redisContext*)handle;
    if (!c || !key || !field) return "";
    
    redisReply *reply = redisCommand(c, "HGET %s %s", key, field);
    if (reply && reply->type == REDIS_REPLY_STRING) {
        char *str = wolf_req_strdup(reply->str);
        freeReplyObject(reply);
        return str;
    }
    if (reply) freeReplyObject(reply);
    return "";
}

void wolf_redis_close(void* handle) {
    redisContext* c = (redisContext*)handle;
    if (c) {
        redisFree(c);
        if (c == wolf_redis_ctx) {
            wolf_redis_ctx = NULL;
        }
    }
}

// ========== Stdlib Strings & JSON ==========

int wolf_strings_contains(const char* s, const char* substr) {
	if (!s || !substr) return 0;
	return strstr(s, substr) != NULL;
}

const char* wolf_strings_upper(const char* s) {
    return wolf_strtoupper(s);
}

const char* wolf_strings_title(const char* s) {
    return wolf_ucwords(s); // Close enough for mock 
}

const char* wolf_strings_trimleft(const char* s, const char* cutset) {
    return wolf_ltrim(s); // Assuming ltrim satisfies the compiler check
}

const char* wolf_strings_trimright(const char* s, const char* cutset) {
    return wolf_rtrim(s);
}

const char* wolf_strings_split(const char* s, const char* sep) {
	return s; // Simple stub for LLVM tests
}

const char* wolf_strings_join(const char* arr, const char* sep) {
	if (!arr) return "";
	size_t len = strlen(arr);
	char* result = (char*)wolf_req_alloc(len + 1);
	for (size_t i = 0; i < len; i++) {
		if (arr[i] == ',') {
			result[i] = sep ? sep[0] : '-';
		} else {
			result[i] = arr[i];
		}
	}
	result[len] = '\0';
	return result;
}

// ========== Data Structures (Arrays & Maps) ==========

typedef struct {
    void** items;
    int64_t length;
    int64_t capacity;
} wolf_array_t;

typedef struct {
    char** keys;
    void** values;
    int64_t size;
    int64_t capacity;
} wolf_map_t;

static wolf_value_t* wolf_val_make(int type) {
    wolf_value_t* v = (wolf_value_t*)wolf_req_alloc(sizeof(wolf_value_t));
    v->type = type;
    return v;
}

// Forward declaration
static char* wolf_json_encode_value(void* val);

// Unwrap any wolf value to its string representation.
// This is the single canonical conversion used by all print/concat/comparison ops.
static const char* wolf_value_unwrap_string(void* val) {
    if (!val) return "";
    wolf_value_t* tagged = (wolf_value_t*)val;
    // Validate type field is in known range before trusting it
    if (tagged->type < WOLF_TYPE_STRING || tagged->type > WOLF_TYPE_ARRAY) {
        // Not a tagged value — treat as plain C string
        return (const char*)val;
    }
    switch (tagged->type) {
        case WOLF_TYPE_INT: {
            char* buf = (char*)wolf_req_alloc(32);
            if (!buf) return "";
            snprintf(buf, 32, "%lld", (long long)tagged->val.i);
            return buf;
        }
        case WOLF_TYPE_FLOAT: {
            char* buf = (char*)wolf_req_alloc(64);
            if (!buf) return "";
            snprintf(buf, 64, "%g", tagged->val.f);
            return buf;
        }
        case WOLF_TYPE_BOOL:
            return tagged->val.b ? "true" : "false";
        case WOLF_TYPE_NULL:
            return "";
        case WOLF_TYPE_STRING:
            return tagged->val.s ? tagged->val.s : "";
        case WOLF_TYPE_MAP:
        case WOLF_TYPE_ARRAY:
            return wolf_json_encode_value(val);
        default:
            return (const char*)val;
    }
}

wolf_value_t* wolf_val_int(int64_t i) {
    wolf_value_t* v = wolf_val_make(WOLF_TYPE_INT);
    v->val.i = i;
    return v;
}

wolf_value_t* wolf_val_float(double f) {
    wolf_value_t* v = wolf_val_make(WOLF_TYPE_FLOAT);
    v->val.f = f;
    return v;
}

wolf_value_t* wolf_val_bool(int b) {
    wolf_value_t* v = wolf_val_make(WOLF_TYPE_BOOL);
    v->val.b = b;
    return v;
}

// Dynamic string buffer for safe JSON building
typedef struct {
    char* data;
    size_t len;
    size_t cap;
} wolf_strbuf_t;

static wolf_strbuf_t* wolf_strbuf_new() {
    wolf_strbuf_t* b = (wolf_strbuf_t*)wolf_req_alloc(sizeof(wolf_strbuf_t));
    if (!b) return NULL;
    b->cap = 256;
    b->len = 0;
    b->data = (char*)wolf_req_alloc(b->cap);
    if (!b->data) { free(b); return NULL; }
    b->data[0] = '\0';
    return b;
}

static int wolf_strbuf_append(wolf_strbuf_t* b, const char* s) {
    if (!b || !s) return 0;
    size_t slen = strlen(s);
    while (b->len + slen + 1 > b->cap) {
        b->cap *= 2;
        char* newdata = (char*)realloc(b->data, b->cap);
        if (!newdata) return 0;
        b->data = newdata;
    }
    memcpy(b->data + b->len, s, slen);
    b->len += slen;
    b->data[b->len] = '\0';
    return 1;
}

static char* wolf_strbuf_take(wolf_strbuf_t* b) {
    if (!b) return wolf_req_strdup("");
    char* result = b->data;
    free(b);
    return result;
}

static char* wolf_json_encode_map(wolf_map_t* m) {
    if (!m) return wolf_req_strdup("{}");
    int64_t n = m->size;
    int64_t* order = (int64_t*)wolf_req_alloc(n * sizeof(int64_t));
    if (!order) return wolf_req_strdup("{}");
    for (int64_t i = 0; i < n; i++) order[i] = i;
    for (int64_t i = 0; i < n - 1; i++) {
        for (int64_t j = i + 1; j < n; j++) {
            if (strcmp(m->keys[order[i]], m->keys[order[j]]) > 0) {
                int64_t tmp = order[i]; order[i] = order[j]; order[j] = tmp;
            }
        }
    }
    wolf_strbuf_t* buf = wolf_strbuf_new();
    if (!buf) { free(order); return wolf_req_strdup("{}"); }
    wolf_strbuf_append(buf, "{");
    for (int64_t i = 0; i < n; i++) {
        if (i > 0) wolf_strbuf_append(buf, ",");
        wolf_strbuf_append(buf, "\"");
        wolf_strbuf_append(buf, m->keys[order[i]]);
        wolf_strbuf_append(buf, "\":");
        char* val = wolf_json_encode_value(m->values[order[i]]);
        wolf_strbuf_append(buf, val);
        free(val);
    }
    wolf_strbuf_append(buf, "}");
    free(order);
    return wolf_strbuf_take(buf);
}

static char* wolf_json_encode_array(wolf_array_t* a) {
    if (!a) return wolf_req_strdup("[]");
    wolf_strbuf_t* buf = wolf_strbuf_new();
    if (!buf) return wolf_req_strdup("[]");
    wolf_strbuf_append(buf, "[");
    for (int64_t i = 0; i < a->length; i++) {
        if (i > 0) wolf_strbuf_append(buf, ",");
        char* val = wolf_json_encode_value(a->items[i]);
        wolf_strbuf_append(buf, val);
        free(val);
    }
    wolf_strbuf_append(buf, "]");
    return wolf_strbuf_take(buf);
}


static char* wolf_json_encode_value(void* val) {
    if (!val) return wolf_req_strdup("null");

    // Check if this is a tagged wolf_value_t
    wolf_value_t* tagged = (wolf_value_t*)val;
    if (tagged->type == WOLF_TYPE_INT) {
        char* buf = (char*)wolf_req_alloc(32);
        snprintf(buf, 32, "%lld", (long long)tagged->val.i);
        return buf;
    }
    if (tagged->type == WOLF_TYPE_FLOAT) {
        char* buf = (char*)wolf_req_alloc(64);
        snprintf(buf, 64, "%g", tagged->val.f);
        return buf;
    }
    if (tagged->type == WOLF_TYPE_BOOL) {
        return wolf_req_strdup(tagged->val.b ? "true" : "false");
    }
    if (tagged->type == WOLF_TYPE_NULL) {
        return wolf_req_strdup("null");
    }
    if (tagged->type == WOLF_TYPE_STRING) {
        const char* s = tagged->val.s;
        size_t len = strlen(s);
        char* out = (char*)wolf_req_alloc(len * 2 + 3);
        char* w = out;
        *w++ = '"';
        for (size_t i = 0; i < len; i++) {
            if (s[i] == '"')       { *w++ = '\\'; *w++ = '"'; }
            else if (s[i] == '\\') { *w++ = '\\'; *w++ = '\\'; }
            else if (s[i] == '\n') { *w++ = '\\'; *w++ = 'n'; }
            else if (s[i] == '\r') { *w++ = '\\'; *w++ = 'r'; }
            else if (s[i] == '\t') { *w++ = '\\'; *w++ = 't'; }
            else *w++ = s[i];
        }
        *w++ = '"';
        *w = '\0';
        return out;
    }
    if (tagged->type == WOLF_TYPE_MAP) {
        wolf_map_t* m = (wolf_map_t*)tagged->val.ptr;
        return wolf_json_encode_map(m);
    }
    if (tagged->type == WOLF_TYPE_ARRAY) {
        wolf_array_t* a = (wolf_array_t*)tagged->val.ptr;
        return wolf_json_encode_array(a);
    }

  // Fallback: treat as raw pointer — could be map, array, or plain string
    // Try array FIRST (before map) — array has items ptr, map has keys ptr
    wolf_array_t* a = (wolf_array_t*)val;
    if (a != NULL && a->capacity > 0 && a->capacity <= 65536 &&
        a->length >= 0 && a->length <= a->capacity && a->items != NULL) {
        return wolf_json_encode_array(a);
    }
    // Try map
    wolf_map_t* m = (wolf_map_t*)val;
    if (m != NULL && m->capacity > 0 && m->capacity <= 65536 &&
        m->size >= 0 && m->size <= m->capacity && m->keys != NULL) {
        return wolf_json_encode_map(m);
    }
    // Plain string
    const char* s = (const char*)val;

    // Plain string — quote it
    size_t len = strlen(s);
    char* out = (char*)wolf_req_alloc(len * 2 + 3);
    char* w = out;
    *w++ = '"';
    for (size_t i = 0; i < len; i++) {
        if (s[i] == '"')       { *w++ = '\\'; *w++ = '"'; }
        else if (s[i] == '\\') { *w++ = '\\'; *w++ = '\\'; }
        else if (s[i] == '\n') { *w++ = '\\'; *w++ = 'n'; }
        else if (s[i] == '\r') { *w++ = '\\'; *w++ = 'r'; }
        else if (s[i] == '\t') { *w++ = '\\'; *w++ = 't'; }
        else *w++ = s[i];
    }
    *w++ = '"';
    *w = '\0';
    return out;
}

const char* wolf_json_encode(void* obj) {
    if (!obj) return wolf_req_strdup("null");
    return wolf_json_encode_value(obj);
}

void* wolf_array_create() {
    wolf_array_t* arr = (wolf_array_t*)wolf_req_alloc(sizeof(wolf_array_t));
    arr->capacity = 8;
    arr->length = 0;
    arr->items = (void**)wolf_req_alloc(sizeof(void*) * arr->capacity);
    return arr;
}

void wolf_array_push(void* a, void* item) {
    if (!a) return;
    wolf_array_t* arr = (wolf_array_t*)a;
    if (arr->length >= arr->capacity) {
        arr->capacity *= 2;
        arr->items = (void**)realloc(arr->items, sizeof(void*) * arr->capacity);
    }
    arr->items[arr->length++] = item;
}

void* wolf_array_get(void* a, int64_t index) {
    if (!a) return NULL;
    wolf_array_t* arr = (wolf_array_t*)a;
    if (index < 0 || index >= arr->length) return NULL;
    void* val = arr->items[index];
    if (!val) return NULL;
    wolf_value_t* tagged = (wolf_value_t*)val;
    if (tagged->type < WOLF_TYPE_STRING || tagged->type > WOLF_TYPE_ARRAY) {
        return val; // plain string pointer, return as-is
    }
    // Unwrap typed values to string for use in Wolf code
    return (void*)wolf_value_unwrap_string(val);
}

int64_t wolf_array_length(void* a) {
    if (!a) return 0;
    return ((wolf_array_t*)a)->length;
}

// ========== Phase 2 Stdlib — Array Functions ==========

// count() — alias for array_length
int64_t wolf_count(void* a) {
    return wolf_array_length(a);
}

// in_array($val, $arr) — search for value (string comparison)
int wolf_in_array(const char* val, void* a) {
    if (!a || !val) return 0;
    wolf_array_t* arr = (wolf_array_t*)a;
    for (int64_t i = 0; i < arr->length; i++) {
        if (arr->items[i] && strcmp((const char*)arr->items[i], val) == 0) return 1;
    }
    return 0;
}

// array_search($val, $arr) — returns index or -1
int64_t wolf_array_search(const char* val, void* a) {
    if (!a || !val) return -1;
    wolf_array_t* arr = (wolf_array_t*)a;
    for (int64_t i = 0; i < arr->length; i++) {
        if (arr->items[i] && strcmp((const char*)arr->items[i], val) == 0) return i;
    }
    return -1;
}

// array_pop($arr) — remove and return last element
void* wolf_array_pop(void* a) {
    if (!a) return NULL;
    wolf_array_t* arr = (wolf_array_t*)a;
    if (arr->length == 0) return NULL;
    return arr->items[--arr->length];
}

// array_shift($arr) — remove and return first element
void* wolf_array_shift(void* a) {
    if (!a) return NULL;
    wolf_array_t* arr = (wolf_array_t*)a;
    if (arr->length == 0) return NULL;
    void* first = arr->items[0];
    for (int64_t i = 0; i < arr->length - 1; i++) {
        arr->items[i] = arr->items[i + 1];
    }
    arr->length--;
    return first;
}

// array_unshift($arr, $val) — add to beginning
void wolf_array_unshift(void* a, void* item) {
    if (!a) return;
    wolf_array_t* arr = (wolf_array_t*)a;
    if (arr->length >= arr->capacity) {
        arr->capacity *= 2;
        arr->items = (void**)realloc(arr->items, sizeof(void*) * arr->capacity);
    }
    for (int64_t i = arr->length; i > 0; i--) {
        arr->items[i] = arr->items[i - 1];
    }
    arr->items[0] = item;
    arr->length++;
}

// array_reverse($arr) — return new reversed array
void* wolf_array_reverse(void* a) {
    if (!a) return wolf_array_create();
    wolf_array_t* arr = (wolf_array_t*)a;
    void* result = wolf_array_create();
    for (int64_t i = arr->length - 1; i >= 0; i--) {
        wolf_array_push(result, arr->items[i]);
    }
    return result;
}

// array_unique($arr) — return new array with duplicates removed
void* wolf_array_unique(void* a) {
    if (!a) return wolf_array_create();
    wolf_array_t* arr = (wolf_array_t*)a;
    void* result = wolf_array_create();
    for (int64_t i = 0; i < arr->length; i++) {
        int found = 0;
        wolf_array_t* res = (wolf_array_t*)result;
        for (int64_t j = 0; j < res->length; j++) {
            if (arr->items[i] && res->items[j] &&
                strcmp((const char*)arr->items[i], (const char*)res->items[j]) == 0) {
                found = 1; break;
            }
        }
        if (!found) wolf_array_push(result, arr->items[i]);
    }
    return result;
}

// array_merge($a, $b) — merge two arrays
void* wolf_array_merge(void* a, void* b) {
    void* result = wolf_array_create();
    if (a) {
        wolf_array_t* arr_a = (wolf_array_t*)a;
        for (int64_t i = 0; i < arr_a->length; i++) {
            wolf_array_push(result, arr_a->items[i]);
        }
    }
    if (b) {
        wolf_array_t* arr_b = (wolf_array_t*)b;
        for (int64_t i = 0; i < arr_b->length; i++) {
            wolf_array_push(result, arr_b->items[i]);
        }
    }
    return result;
}

// array_slice($arr, $offset, $length) — return slice
void* wolf_array_slice(void* a, int64_t offset, int64_t len) {
    void* result = wolf_array_create();
    if (!a) return result;
    wolf_array_t* arr = (wolf_array_t*)a;
    if (offset < 0) offset = arr->length + offset;
    if (offset < 0) offset = 0;
    if (len < 0) len = arr->length - offset + len;
    if (len <= 0) return result;
    for (int64_t i = offset; i < offset + len && i < arr->length; i++) {
        wolf_array_push(result, arr->items[i]);
    }
    return result;
}

// sort($arr) — sort array in place (string comparison)
void wolf_sort(void* a) {
    if (!a) return;
    wolf_array_t* arr = (wolf_array_t*)a;
    // Bubble sort — works for moderate sizes
    for (int64_t i = 0; i < arr->length - 1; i++) {
        for (int64_t j = 0; j < arr->length - i - 1; j++) {
            const char* s1 = arr->items[j] ? (const char*)arr->items[j] : "";
            const char* s2 = arr->items[j+1] ? (const char*)arr->items[j+1] : "";
            if (strcmp(s1, s2) > 0) {
                void* tmp = arr->items[j];
                arr->items[j] = arr->items[j+1];
                arr->items[j+1] = tmp;
            }
        }
    }
}

// rsort($arr) — sort descending in place
void wolf_rsort(void* a) {
    if (!a) return;
    wolf_array_t* arr = (wolf_array_t*)a;
    for (int64_t i = 0; i < arr->length - 1; i++) {
        for (int64_t j = 0; j < arr->length - i - 1; j++) {
            const char* s1 = arr->items[j] ? (const char*)arr->items[j] : "";
            const char* s2 = arr->items[j+1] ? (const char*)arr->items[j+1] : "";
            if (strcmp(s1, s2) < 0) {
                void* tmp = arr->items[j];
                arr->items[j] = arr->items[j+1];
                arr->items[j+1] = tmp;
            }
        }
    }
}

// array_sum($arr) — sum all values (treats as doubles via atof)
double wolf_array_sum(void* a) {
    if (!a) return 0.0;
    wolf_array_t* arr = (wolf_array_t*)a;
    double sum = 0.0;
    for (int64_t i = 0; i < arr->length; i++) {
        if (arr->items[i]) sum += atof((const char*)arr->items[i]);
    }
    return sum;
}

// array_keys($map) — return keys as array (for wolf_map_t)
void* wolf_array_keys(void* m) {
    if (!m) return wolf_array_create();
    wolf_map_t* map = (wolf_map_t*)m;
    void* result = wolf_array_create();
    for (int64_t i = 0; i < map->size; i++) {
        wolf_array_push(result, (void*)map->keys[i]);
    }
    return result;
}

// array_values($map) — return values as array (for wolf_map_t)
void* wolf_array_values(void* m) {
    if (!m) return wolf_array_create();
    wolf_map_t* map = (wolf_map_t*)m;
    void* result = wolf_array_create();
    for (int64_t i = 0; i < map->size; i++) {
        wolf_array_push(result, map->values[i]);
    }
    return result;
}

// array_diff($a, $b) — values in a not in b
void* wolf_array_diff(void* a, void* b) {
    void* result = wolf_array_create();
    if (!a) return result;
    wolf_array_t* arr_a = (wolf_array_t*)a;
    for (int64_t i = 0; i < arr_a->length; i++) {
        if (!wolf_in_array((const char*)arr_a->items[i], b)) {
            wolf_array_push(result, arr_a->items[i]);
        }
    }
    return result;
}

// array_intersect($a, $b) — values in both
void* wolf_array_intersect(void* a, void* b) {
    void* result = wolf_array_create();
    if (!a || !b) return result;
    wolf_array_t* arr_a = (wolf_array_t*)a;
    for (int64_t i = 0; i < arr_a->length; i++) {
        if (wolf_in_array((const char*)arr_a->items[i], b)) {
            wolf_array_push(result, arr_a->items[i]);
        }
    }
    return result;
}

// array_flip($arr) — swap keys/values (returns map from array)
void* wolf_array_flip(void* a) {
    if (!a) return wolf_map_create();
    wolf_array_t* arr = (wolf_array_t*)a;
    void* result = wolf_map_create();
    for (int64_t i = 0; i < arr->length; i++) {
        char idx[32];
        snprintf(idx, sizeof(idx), "%lld", (long long)i);
        wolf_map_set(result, (const char*)arr->items[i], wolf_req_strdup(idx));
    }
    return result;
}

// range($start, $end) — create array [start..end]
void* wolf_range(int64_t start, int64_t end) {
    void* result = wolf_array_create();
    int64_t step = start <= end ? 1 : -1;
    for (int64_t i = start; step > 0 ? i <= end : i >= end; i += step) {
        char* s = (char*)wolf_req_alloc(32);
        snprintf(s, 32, "%lld", (long long)i);
        wolf_array_push(result, s);
    }
    return result;
}

void* wolf_map_create() {
    wolf_map_t* m = (wolf_map_t*)wolf_req_alloc(sizeof(wolf_map_t));
    m->capacity = 8;
    m->size = 0;
    m->keys = (char**)wolf_req_alloc(sizeof(char*) * m->capacity);
    m->values = (void**)wolf_req_alloc(sizeof(void*) * m->capacity);
    return m;
}


static wolf_value_t* wolf_val_from_ptr(void* ptr) {
    // If it's already a tagged value, return as-is
    // Otherwise wrap as string
    if (!ptr) {
        wolf_value_t* v = wolf_val_make(WOLF_TYPE_NULL);
        return v;
    }
    wolf_value_t* v = wolf_val_make(WOLF_TYPE_STRING);
    v->val.s = wolf_req_strdup((const char*)ptr);
    return v;
}
void wolf_map_set_int(void* map_ptr, const char* key, int64_t value) {
    if (!map_ptr || !key) return;
    wolf_value_t* v = wolf_val_make(WOLF_TYPE_INT);
    v->val.i = value;
    wolf_map_t* m = (wolf_map_t*)map_ptr;
    // Check if key exists, update if so
    for (int64_t i = 0; i < m->size; i++) {
        if (strcmp(m->keys[i], key) == 0) {
            m->values[i] = v;
            return;
        }
    }
    if (m->size >= m->capacity) {
        m->capacity *= 2;
        m->keys = (char**)realloc(m->keys, sizeof(char*) * m->capacity);
        m->values = (void**)realloc(m->values, sizeof(void*) * m->capacity);
    }
    m->keys[m->size] = wolf_req_strdup(key);
    m->values[m->size] = v;
    m->size++;
}

void wolf_map_set_float(void* map_ptr, const char* key, double value) {
    if (!map_ptr || !key) return;
    wolf_value_t* v = wolf_val_make(WOLF_TYPE_FLOAT);
    v->val.f = value;
    wolf_map_t* m = (wolf_map_t*)map_ptr;
    for (int64_t i = 0; i < m->size; i++) {
        if (strcmp(m->keys[i], key) == 0) {
            m->values[i] = v;
            return;
        }
    }
    if (m->size >= m->capacity) {
        m->capacity *= 2;
        m->keys = (char**)realloc(m->keys, sizeof(char*) * m->capacity);
        m->values = (void**)realloc(m->values, sizeof(void*) * m->capacity);
    }
    m->keys[m->size] = wolf_req_strdup(key);
    m->values[m->size] = v;
    m->size++;
}

void wolf_map_set_bool(void* map_ptr, const char* key, int value) {
    if (!map_ptr || !key) return;
    wolf_value_t* v = wolf_val_make(WOLF_TYPE_BOOL);
    v->val.b = value;
    wolf_map_t* m = (wolf_map_t*)map_ptr;
    for (int64_t i = 0; i < m->size; i++) {
        if (strcmp(m->keys[i], key) == 0) {
            m->values[i] = v;
            return;
        }
    }
    if (m->size >= m->capacity) {
        m->capacity *= 2;
        m->keys = (char**)realloc(m->keys, sizeof(char*) * m->capacity);
        m->values = (void**)realloc(m->values, sizeof(void*) * m->capacity);
    }
    m->keys[m->size] = wolf_req_strdup(key);
    m->values[m->size] = v;
    m->size++;
}

void wolf_map_set(void* map_ptr, const char* key, void* value) {
    if (!map_ptr || !key) return;
    wolf_map_t* m = (wolf_map_t*)map_ptr;
    for (int64_t i = 0; i < m->size; i++) {
        if (strcmp(m->keys[i], key) == 0) {
            m->values[i] = value;
            return;
        }
    }
    if (m->size >= m->capacity) {
        m->capacity *= 2;
        m->keys = (char**)realloc(m->keys, sizeof(char*) * m->capacity);
        m->values = (void**)realloc(m->values, sizeof(void*) * m->capacity);
    }
    m->keys[m->size] = wolf_req_strdup(key);
    m->values[m->size] = value;
    m->size++;
}

void* wolf_map_get(void* map_ptr, const char* key) {
    if (!map_ptr || !key) return NULL;
    wolf_map_t* m = (wolf_map_t*)map_ptr;
    for (int64_t i = 0; i < m->size; i++) {
        if (strcmp(m->keys[i], key) == 0) {
            void* val = m->values[i];
            if (!val) return NULL;
            // Unwrap tagged values back to string representation
            wolf_value_t* tagged = (wolf_value_t*)val;
            switch (tagged->type) {
                case WOLF_TYPE_INT: {
                    char* buf = (char*)wolf_req_alloc(32);
                    snprintf(buf, 32, "%lld", (long long)tagged->val.i);
                    return buf;
                }
                case WOLF_TYPE_FLOAT: {
                    char* buf = (char*)wolf_req_alloc(64);
                    snprintf(buf, 64, "%g", tagged->val.f);
                    return buf;
                }
                case WOLF_TYPE_BOOL:
                    return wolf_req_strdup(tagged->val.b ? "true" : "false");
                case WOLF_TYPE_NULL:
                    return NULL;
                case WOLF_TYPE_STRING:
                    return wolf_req_strdup(tagged->val.s);
                case WOLF_TYPE_MAP:
                case WOLF_TYPE_ARRAY:
                    return tagged->val.ptr;
                default:
                    return val;
            }
        }
    }
    return NULL;
}

void* wolf_class_create(const char* name) {
    return wolf_map_create(); // Under the hood, objects are just maps
}

int wolf_env_has(const char* key) {
    return 0; // Return false
}

// ========== Conversions ==========

const char* wolf_int_to_string(int64_t n) {
    char* buf = (char*)wolf_req_alloc(32);
    snprintf(buf, 32, "%lld", (long long)n);
    return buf;
}

const char* wolf_float_to_string(double f) {
    char* buf = (char*)wolf_req_alloc(64);
    snprintf(buf, 64, "%g", f);
    return buf;
}

const char* wolf_bool_to_string(int b) {
    return b ? "true" : "false";
}

// ========== Memory ==========

void* wolf_alloc(int64_t size) {
    return wolf_req_alloc((size_t)size);
}

void wolf_free(void* ptr) {
    free(ptr);
}

// ========== HTTP Server ==========

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
// static pthread_mutex_t db_mutex = PTHREAD_MUTEX_INITIALIZER;
static wolf_http_handler_t global_wolf_handler = NULL;

// Thread-local storage for current request/response context
// Each HTTP worker thread sets these before calling the Wolf handler
static __thread int64_t wolf_current_req_id = -1;
static __thread int64_t wolf_current_res_id = -1;

void wolf_set_current_context(void* req_id, void* res_id) {
    wolf_current_req_id = (intptr_t)req_id;
    wolf_current_res_id = (intptr_t)res_id;
}

// Get request body from thread-local context (Controller.getData() uses this)
const char* wolf_get_request_body(void) {
    if (wolf_current_req_id < 0) return "";
    return wolf_http_req_body(wolf_current_req_id);
}

// Get request header from thread-local context (Controller.getAuthorizationHeader() uses this)
const char* wolf_get_request_header(const char* key) {
    if (wolf_current_req_id < 0 || !key) return "";
    return wolf_http_req_header(wolf_current_req_id, key);
}

// Get request method from thread-local context
const char* wolf_get_request_method(void) {
    if (wolf_current_req_id < 0) return "";
    return wolf_http_req_method(wolf_current_req_id);
}

// Get request path from thread-local context
const char* wolf_get_request_path(void) {
    if (wolf_current_req_id < 0) return "";
    return wolf_http_req_path(wolf_current_req_id);
}

// input() — get a field from POST JSON body, or entire body if key is NULL
const char* wolf_input(const char* key) {
    const char* body = wolf_get_request_body();
    if (!body || strlen(body) == 0) return "";
    if (!key || strlen(key) == 0) return body;
    // Simple JSON key extraction (for real usage, uses json_decode internally)
    // For now, return the whole body — the Wolf-level input() calls json_decode + map access
    return body;
}

// http_response_code() — set HTTP status on current response
void wolf_http_response_code(int64_t code) {
    if (wolf_current_res_id < 0) return;
    wolf_http_res_status(wolf_current_res_id, code);
}

// Write response body to current context (used by sendResponse -> print -> output)
void wolf_http_write_response(const char* body) {
    if (wolf_current_res_id < 0) return;
    wolf_http_res_header(wolf_current_res_id, "Content-Type", "application/json");
    wolf_http_res_write(wolf_current_res_id, body);
}

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
        ctx->body = wolf_req_strdup(body_start);
    } else {
        ctx->body = wolf_req_strdup("");
    }
    
    char* saveptr;
    char* line = strtok_r(raw_req, "\r\n", &saveptr);
    if (!line) return;
    
    char* l_save;
    char* method = strtok_r(line, " ", &l_save);
    char* full_path = strtok_r(NULL, " ", &l_save);
    
    if (method) ctx->method = wolf_req_strdup(method);
    if (full_path) {
        char* q_mark = strchr(full_path, '?');
        if (q_mark) {
            *q_mark = '\0';
            ctx->path = wolf_req_strdup(full_path);
            ctx->query = wolf_req_strdup(q_mark + 1);
        } else {
            ctx->path = wolf_req_strdup(full_path);
            ctx->query = wolf_req_strdup("");
        }
    }
    
    while ((line = strtok_r(NULL, "\r\n", &saveptr))) {
        char* colon = strchr(line, ':');
        if (colon && ctx->header_count < 32) {
            *colon = '\0';
            char* val = colon + 1;
            while (*val == ' ') val++;
            ctx->header_keys[ctx->header_count] = wolf_req_strdup(line);
            ctx->header_vals[ctx->header_count] = wolf_req_strdup(val);
            ctx->header_count++;
        }
    }
}

static void* http_worker(void* arg) {
#if !defined(WOLF_DB_POSTGRES) && !defined(WOLF_DB_MSSQL)
    mysql_thread_init();  // ← add this
#endif
    int id = (int)(intptr_t)arg;
    wolf_http_context_t* ctx = &http_contexts[id];
    
    char buffer[WOLF_MAX_REQUEST_SIZE];
    memset(buffer, 0, sizeof(buffer));
    
    ssize_t bytes_read = read(ctx->client_fd, buffer, sizeof(buffer) - 1);
    if (bytes_read > 0) {
        parse_http_request(id, buffer, bytes_read);
        
        if (global_wolf_handler) {
            wolf_req_arena_init();  /* ← start per-request arena */
            wolf_set_current_context((void*)(intptr_t)id, (void*)(intptr_t)id);
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
    wolf_req_arena_flush();  /* ← free all arena allocations for this request */
#if !defined(WOLF_DB_POSTGRES) && !defined(WOLF_DB_MSSQL)
    mysql_thread_end();  // ← add this
#endif
    return NULL;
}

void wolf_http_serve(int64_t port, void* handler_ptr) {
    global_wolf_handler = (wolf_http_handler_t)handler_ptr;
#if !defined(WOLF_DB_POSTGRES) && !defined(WOLF_DB_MSSQL)
    mysql_library_init(0, NULL, NULL);  // ← add this
#endif
    
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
    
    char* q_copy = wolf_req_strdup(query);
    char* saveptr;
    char* pair = strtok_r(q_copy, "&", &saveptr);
    
    while (pair) {
        char* eq = strchr(pair, '=');
        if (eq) {
            *eq = '\0';
            if (strcmp(pair, key) == 0) {
                char* result = wolf_req_strdup(eq + 1);
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
        ctx->res_header_keys[ctx->res_header_count] = wolf_req_strdup(key);
        ctx->res_header_vals[ctx->res_header_count] = wolf_req_strdup(value);
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
        char* new_body = wolf_req_alloc(old_len + new_len + 1);
        strcpy(new_body, ctx->res_body);
        strcat(new_body, body);
        free(ctx->res_body);
        ctx->res_body = new_body;
    } else {
        ctx->res_body = wolf_req_strdup(body);
    }
}

// ========== Phase 1 Stdlib — Strings ==========

const char* wolf_strtoupper(const char* s) { return wolf_string_upper(s); }
const char* wolf_strtolower(const char* s) { return wolf_string_lower(s); }

const char* wolf_ucfirst(const char* s) {
    if (!s || !*s) return s ? wolf_req_strdup(s) : wolf_req_strdup("");
    char* r = wolf_req_strdup(s);
    r[0] = (char)toupper((unsigned char)r[0]);
    return r;
}

const char* wolf_ucwords(const char* s) {
    if (!s) return wolf_req_strdup("");
    char* r = wolf_req_strdup(s);
    int cap = 1;
    for (int i = 0; r[i]; i++) {
        if (r[i] == ' ' || r[i] == '\t' || r[i] == '\n') { cap = 1; }
        else if (cap) { r[i] = (char)toupper((unsigned char)r[i]); cap = 0; }
    }
    return r;
}

const char* wolf_lcfirst(const char* s) {
    if (!s || !*s) return s ? wolf_req_strdup(s) : wolf_req_strdup("");
    char* r = wolf_req_strdup(s);
    r[0] = (char)tolower((unsigned char)r[0]);
    return r;
}

const char* wolf_trim(const char* s) { return wolf_string_trim(s); }

const char* wolf_ltrim(const char* s) {
    if (!s) return wolf_req_strdup("");
    while (*s && isspace((unsigned char)*s)) s++;
    return wolf_req_strdup(s);
}

const char* wolf_rtrim(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len = strlen(s);
    while (len > 0 && isspace((unsigned char)s[len - 1])) len--;
    char* r = (char*)wolf_req_alloc(len + 1);
    memcpy(r, s, len);
    r[len] = '\0';
    return r;
}

int wolf_str_contains(const char* s, const char* sub) {
    if (!s || !sub) return 0;
    return strstr(s, sub) != NULL;
}

int wolf_str_starts_with(const char* s, const char* prefix) {
    if (!s || !prefix) return 0;
    return strncmp(s, prefix, strlen(prefix)) == 0;
}

int wolf_str_ends_with(const char* s, const char* suffix) {
    if (!s || !suffix) return 0;
    size_t sl = strlen(s), ul = strlen(suffix);
    if (ul > sl) return 0;
    return strcmp(s + sl - ul, suffix) == 0;
}

const char* wolf_str_replace(const char* find, const char* rep, const char* s) {
    if (!s || !find || !rep || !*find) return s ? wolf_req_strdup(s) : wolf_req_strdup("");
    size_t fl = strlen(find), rl = strlen(rep);
    // Count occurrences
    int count = 0;
    const char* p = s;
    while ((p = strstr(p, find))) { count++; p += fl; }
    size_t new_len = strlen(s) + count * (rl - fl);
    char* result = (char*)wolf_req_alloc(new_len + 1);
    char* w = result;
    p = s;
    while (*p) {
        if (strncmp(p, find, fl) == 0) {
            memcpy(w, rep, rl); w += rl; p += fl;
        } else { *w++ = *p++; }
    }
    *w = '\0';
    return result;
}

const char* wolf_str_repeat(const char* s, int64_t times) {
    if (!s || times <= 0) return wolf_req_strdup("");
    size_t sl = strlen(s);
    char* r = (char*)wolf_req_alloc(sl * times + 1);
    r[0] = '\0';
    for (int64_t i = 0; i < times; i++) strcat(r, s);
    return r;
}

const char* wolf_str_pad(const char* s, int64_t len, const char* pad) {
    if (!s) return wolf_req_strdup("");
    if (!pad || !*pad) pad = " ";
    size_t sl = strlen(s);
    if ((int64_t)sl >= len) return wolf_req_strdup(s);
    char* r = (char*)wolf_req_alloc(len + 1);
    strcpy(r, s);
    size_t pl = strlen(pad);
    size_t pos = sl;
    while ((int64_t)pos < len) { r[pos] = pad[(pos - sl) % pl]; pos++; }
    r[len] = '\0';
    return r;
}

// explode: split string by separator — returns a real wolf_array_t
void* wolf_explode(const char* sep, const char* s) {
    void* result = wolf_array_create();
    if (!s || !sep || !*sep) return result;
    size_t sep_len = strlen(sep);
    const char* p = s;
    while (1) {
        const char* found = strstr(p, sep);
        if (!found) {
            if (*p) wolf_array_push(result, wolf_req_strdup(p));
            break;
        }
        size_t chunk_len = found - p;
        if (chunk_len > 0) {
            char* chunk = (char*)wolf_req_alloc(chunk_len + 1);
            memcpy(chunk, p, chunk_len);
            chunk[chunk_len] = '\0';
            wolf_array_push(result, chunk);
        }
        p = found + sep_len;
    }
    return result;
}

// implode: join array with separator
const char* wolf_implode(const char* sep, void* arr) {
    if (!arr) return wolf_req_strdup("");
    if (!sep) sep = "";
    wolf_array_t* a = (wolf_array_t*)arr;
    if (a->length == 0) return wolf_req_strdup("");
    size_t sep_len = strlen(sep);
    size_t total = 0;
    for (int64_t i = 0; i < a->length; i++) {
        if (a->items[i]) total += strlen((const char*)a->items[i]);
        if (i < a->length - 1) total += sep_len;
    }
    char* result = (char*)wolf_req_alloc(total + 1);
    char* w = result;
    for (int64_t i = 0; i < a->length; i++) {
        if (a->items[i]) {
            size_t l = strlen((const char*)a->items[i]);
            memcpy(w, a->items[i], l);
            w += l;
        }
        if (i < a->length - 1) {
            memcpy(w, sep, sep_len);
            w += sep_len;
        }
    }
    *w = '\0';
    return result;
}

const char* wolf_substr(const char* s, int64_t start, int64_t len) {
    if (!s) return wolf_req_strdup("");
    size_t sl = strlen(s);
    if (start < 0) start = (int64_t)sl + start;
    if (start < 0) start = 0;
    if ((size_t)start >= sl) return wolf_req_strdup("");
    if (len < 0) len = (int64_t)sl - start + len;
    if (len <= 0) return wolf_req_strdup("");
    if ((size_t)(start + len) > sl) len = (int64_t)sl - start;
    char* r = (char*)wolf_req_alloc(len + 1);
    memcpy(r, s + start, len);
    r[len] = '\0';
    return r;
}

int64_t wolf_strpos(const char* s, const char* sub) {
    if (!s || !sub) return -1;
    const char* p = strstr(s, sub);
    if (!p) return -1;
    return (int64_t)(p - s);
}

int64_t wolf_strrpos(const char* s, const char* sub) {
    if (!s || !sub) return -1;
    int64_t last = -1;
    size_t sl = strlen(sub);
    const char* p = s;
    while ((p = strstr(p, sub))) { last = (int64_t)(p - s); p += sl; }
    return last;
}

int64_t wolf_str_word_count(const char* s) {
    if (!s) return 0;
    int64_t count = 0;
    int in_word = 0;
    while (*s) {
        if (isspace((unsigned char)*s)) { in_word = 0; }
        else if (!in_word) { in_word = 1; count++; }
        s++;
    }
    return count;
}

int64_t wolf_strcmp(const char* a, const char* b) {
    if (!a) a = "";
    if (!b) b = "";
    return (int64_t)strcmp(a, b);
}

const char* wolf_nl2br(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len = strlen(s), nl = 0;
    for (size_t i = 0; i < len; i++) if (s[i] == '\n') nl++;
    char* r = (char*)wolf_req_alloc(len + nl * 5 + 1); // "<br>\n" = 5 chars per \n
    char* w = r;
    for (size_t i = 0; i < len; i++) {
        if (s[i] == '\n') { memcpy(w, "<br>\n", 5); w += 5; }
        else *w++ = s[i];
    }
    *w = '\0';
    return r;
}

const char* wolf_strip_tags(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len = strlen(s);
    char* r = (char*)wolf_req_alloc(len + 1);
    char* w = r;
    int in_tag = 0;
    for (size_t i = 0; i < len; i++) {
        if (s[i] == '<') in_tag = 1;
        else if (s[i] == '>') in_tag = 0;
        else if (!in_tag) *w++ = s[i];
    }
    *w = '\0';
    return r;
}

const char* wolf_htmlspecialchars(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len = strlen(s);
    char* r = (char*)wolf_req_alloc(len * 6 + 1); // worst case &amp; = 5x
    char* w = r;
    for (size_t i = 0; i < len; i++) {
        switch (s[i]) {
            case '&': memcpy(w, "&amp;", 5); w += 5; break;
            case '<': memcpy(w, "&lt;", 4); w += 4; break;
            case '>': memcpy(w, "&gt;", 4); w += 4; break;
            case '"': memcpy(w, "&quot;", 6); w += 6; break;
            case '\'': memcpy(w, "&#039;", 6); w += 6; break;
            default: *w++ = s[i];
        }
    }
    *w = '\0';
    return r;
}

const char* wolf_addslashes(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len = strlen(s);
    char* r = (char*)wolf_req_alloc(len * 2 + 1);
    char* w = r;
    for (size_t i = 0; i < len; i++) {
        if (s[i] == '\'' || s[i] == '"' || s[i] == '\\') *w++ = '\\';
        *w++ = s[i];
    }
    *w = '\0';
    return r;
}

const char* wolf_stripslashes(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len = strlen(s);
    char* r = (char*)wolf_req_alloc(len + 1);
    char* w = r;
    for (size_t i = 0; i < len; i++) {
        if (s[i] == '\\' && i + 1 < len) { i++; *w++ = s[i]; }
        else *w++ = s[i];
    }
    *w = '\0';
    return r;
}

const char* wolf_sprintf(const char* fmt, const char* arg1) {
    // Simplified: only supports one %s or %d replacement
    if (!fmt) return wolf_req_strdup("");
    if (!arg1) arg1 = "";
    char* r = (char*)wolf_req_alloc(strlen(fmt) + strlen(arg1) + 64);
    snprintf(r, strlen(fmt) + strlen(arg1) + 64, fmt, arg1);
    return r;
}

// ========== Math Extras ==========

double wolf_deg2rad(double deg) { return deg * 3.14159265358979323846 / 180.0; }
double wolf_rad2deg(double rad) { return rad * 180.0 / 3.14159265358979323846; }
double wolf_clamp(double n, double mn, double mx) { return n < mn ? mn : (n > mx ? mx : n); }

// ========== Type Casting ==========

int64_t wolf_intval(const char* s) {
    if (!s) return 0;
    return (int64_t)atoll(s);
}

double wolf_floatval(const char* s) {
    if (!s) return 0.0;
    return atof(s);
}

const char* wolf_strval(int64_t n) {
    return wolf_int_to_string(n);
}

int wolf_boolval(const char* s) {
    if (!s || !*s || strcmp(s, "0") == 0 || strcmp(s, "false") == 0) return 0;
    return 1;
}

int64_t wolf_intdiv(int64_t a, int64_t b) {
    if (b == 0) return 0;
    return a / b;
}

const char* wolf_gettype(const char* val) {
    // Simplified — real impl needs tagged unions
    if (!val) return "null";
    return "string";
}

int wolf_is_numeric(const char* s) {
    if (!s || !*s) return 0;
    if (*s == '-' || *s == '+') s++;
    int has_dot = 0;
    while (*s) {
        if (*s == '.') { if (has_dot) return 0; has_dot = 1; }
        else if (!isdigit((unsigned char)*s)) return 0;
        s++;
    }
    return 1;
}

// ========== Encoding ==========

static const char b64_table[] = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

const char* wolf_base64_encode(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len = strlen(s);
    size_t out_len = 4 * ((len + 2) / 3);
    char* out = (char*)wolf_req_alloc(out_len + 1);
    char* p = out;
    for (size_t i = 0; i < len; i += 3) {
        uint32_t n = ((uint32_t)(unsigned char)s[i]) << 16;
        if (i + 1 < len) n |= ((uint32_t)(unsigned char)s[i + 1]) << 8;
        if (i + 2 < len) n |= (uint32_t)(unsigned char)s[i + 2];
        *p++ = b64_table[(n >> 18) & 0x3F];
        *p++ = b64_table[(n >> 12) & 0x3F];
        *p++ = (i + 1 < len) ? b64_table[(n >> 6) & 0x3F] : '=';
        *p++ = (i + 2 < len) ? b64_table[n & 0x3F] : '=';
    }
    *p = '\0';
    return out;
}

static int b64_decode_char(char c) {
    if (c >= 'A' && c <= 'Z') return c - 'A';
    if (c >= 'a' && c <= 'z') return c - 'a' + 26;
    if (c >= '0' && c <= '9') return c - '0' + 52;
    if (c == '+') return 62;
    if (c == '/') return 63;
    return -1;
}

const char* wolf_base64_decode(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len = strlen(s);
    size_t out_len = len * 3 / 4;
    char* out = (char*)wolf_req_alloc(out_len + 1);
    char* p = out;
    for (size_t i = 0; i < len; i += 4) {
        int a = b64_decode_char(s[i]), b = b64_decode_char(s[i+1]);
        int c = (i+2 < len && s[i+2] != '=') ? b64_decode_char(s[i+2]) : 0;
        int d = (i+3 < len && s[i+3] != '=') ? b64_decode_char(s[i+3]) : 0;
        if (a < 0 || b < 0) break;
        *p++ = (char)((a << 2) | (b >> 4));
        if (s[i+2] != '=') *p++ = (char)(((b & 0xF) << 4) | (c >> 2));
        if (s[i+3] != '=') *p++ = (char)(((c & 0x3) << 6) | d);
    }
    *p = '\0';
    return out;
}

const char* wolf_url_encode(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len = strlen(s);
    char* r = (char*)wolf_req_alloc(len * 3 + 1);
    char* w = r;
    for (size_t i = 0; i < len; i++) {
        unsigned char c = (unsigned char)s[i];
        if (isalnum(c) || c == '-' || c == '_' || c == '.' || c == '~') *w++ = c;
        else { sprintf(w, "%%%02X", c); w += 3; }
    }
    *w = '\0';
    return r;
}

const char* wolf_url_decode(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len = strlen(s);
    char* r = (char*)wolf_req_alloc(len + 1);
    char* w = r;
    for (size_t i = 0; i < len; i++) {
        if (s[i] == '%' && i + 2 < len) {
            char hex[3] = { s[i+1], s[i+2], 0 };
            *w++ = (char)strtol(hex, NULL, 16);
            i += 2;
        } else if (s[i] == '+') { *w++ = ' '; }
        else { *w++ = s[i]; }
    }
    *w = '\0';
    return r;
}

// ========== Security (Stubs — real impl needs OpenSSL) ==========

const char* wolf_md5(const char* s) {
    // Simplified stub — returns fixed-length hash representation
    if (!s) return wolf_req_strdup("d41d8cd98f00b204e9800998ecf8427e"); // md5 of empty string
    unsigned long hash = 5381;
    while (*s) hash = ((hash << 5) + hash) + (unsigned char)*s++;
    char* r = (char*)wolf_req_alloc(33);
    snprintf(r, 33, "%016lx%016lx", hash, hash ^ 0xDEADBEEF);
    return r;
}

const char* wolf_sha256(const char* s) {
    // Stub — real impl needs OpenSSL
    if (!s) return wolf_req_strdup("");
    unsigned long hash = 5381;
    while (*s) hash = ((hash << 5) + hash) + (unsigned char)*s++;
    char* r = (char*)wolf_req_alloc(65);
    snprintf(r, 65, "%016lx%016lx%016lx%016lx", hash, hash ^ 0xCAFEBABE, hash ^ 0xDEADBEEF, hash ^ 0xBAADF00D);
    return r;
}

const char* wolf_password_hash(const char* password) {
    // Stub — real impl needs bcrypt library
    return wolf_sha256(password);
}

int wolf_password_verify(const char* password, const char* hash) {
    const char* computed = wolf_sha256(password);
    return strcmp(computed, hash) == 0;
}

const char* wolf_uuid_v4() {
    char* r = (char*)wolf_req_alloc(37);
    srand(time(NULL) ^ clock());
    snprintf(r, 37, "%04x%04x-%04x-%04x-%04x-%04x%04x%04x",
             rand() & 0xFFFF, rand() & 0xFFFF,
             rand() & 0xFFFF,
             (rand() & 0x0FFF) | 0x4000,
             (rand() & 0x3FFF) | 0x8000,
             rand() & 0xFFFF, rand() & 0xFFFF, rand() & 0xFFFF);
    return r;
}

const char* wolf_rand_hex(int64_t length) {
    char* r = (char*)wolf_req_alloc(length * 2 + 1);
    srand(time(NULL) ^ clock());
    for (int64_t i = 0; i < length; i++) {
        sprintf(r + i * 2, "%02x", rand() & 0xFF);
    }
    r[length * 2] = '\0';
    return r;
}

// ========== Output & Debugging ==========

void wolf_dump(const char* val) {
    if (val) printf("[dump] string(%zu) \"%s\"\n", strlen(val), val);
    else printf("[dump] NULL\n");
}

void wolf_dd(const char* val) {
    wolf_dump(val);
    exit(0);
}

void wolf_log_info(const char* msg) {
    if (msg) fprintf(stderr, "[INFO] %s\n", msg);
}

void wolf_log_warning(const char* msg) {
    if (msg) fprintf(stderr, "[WARN] %s\n", msg);
}

void wolf_log_error(const char* msg) {
    if (msg) fprintf(stderr, "[ERROR] %s\n", msg);
}

const char* wolf_json_pretty(const char* json) {
    // Stub — returns as-is for now
    return json ? wolf_req_strdup(json) : wolf_req_strdup("null");
}

void* wolf_json_decode(const char* json) {
    if (!json) return wolf_map_create();
    while (*json && (*json == ' ' || *json == '\t' || *json == '\n')) json++;

    // Handle JSON arrays
    if (*json == '[') {
        void* arr = wolf_array_create();
        json++; // skip '['
        while (*json) {
            while (*json && (*json == ' ' || *json == '\t' ||
                   *json == '\n' || *json == ',')) json++;
            if (*json == ']') break;
            if (*json == '"') {
                json++;
                size_t vcap = 256;
                char* val = (char*)wolf_req_alloc(vcap);
                int vi = 0;
                while (*json && *json != '"') {
                    if ((size_t)vi >= vcap - 8) {
                        vcap *= 2;
                        char* nv = (char*)realloc(val, vcap);
                        if (!nv) { free(val); val = NULL; break; }
                        val = nv;
                    }
                    if (*json == '\\' && *(json+1)) {
                        json++;
                        switch (*json) {
                            case 'n':  val[vi++] = '\n'; break;
                            case 't':  val[vi++] = '\t'; break;
                            case 'r':  val[vi++] = '\r'; break;
                            case 'b':  val[vi++] = '\b'; break;
                            case 'f':  val[vi++] = '\f'; break;
                            case '"':  val[vi++] = '"';  break;
                            case '\\': val[vi++] = '\\'; break;
                            case '/':  val[vi++] = '/';  break;
                            case 'u': {
                                /* \uXXXX — decode to UTF-8 */
                                json++;
                                unsigned int codepoint = 0;
                                for (int ci = 0; ci < 4 && *json; ci++, json++) {
                                    codepoint <<= 4;
                                    char hx = *json;
                                    if (hx >= '0' && hx <= '9') codepoint |= (hx - '0');
                                    else if (hx >= 'a' && hx <= 'f') codepoint |= (hx - 'a' + 10);
                                    else if (hx >= 'A' && hx <= 'F') codepoint |= (hx - 'A' + 10);
                                }
                                json--; /* loop will ++ */
                                /* encode as UTF-8 */
                                if (codepoint < 0x80) {
                                    val[vi++] = (char)codepoint;
                                } else if (codepoint < 0x800) {
                                    val[vi++] = (char)(0xC0 | (codepoint >> 6));
                                    val[vi++] = (char)(0x80 | (codepoint & 0x3F));
                                } else {
                                    val[vi++] = (char)(0xE0 | (codepoint >> 12));
                                    val[vi++] = (char)(0x80 | ((codepoint >> 6) & 0x3F));
                                    val[vi++] = (char)(0x80 | (codepoint & 0x3F));
                                }
                                break;
                            }
                            default: val[vi++] = *json; break;
                        }
                    } else {
                        val[vi++] = *json;
                    }
                    json++;
                }
                if (val) val[vi] = '\0';
                if (*json == '"') json++;
                /* Wrap as typed string value */
                wolf_value_t* sv = wolf_val_make(WOLF_TYPE_STRING);
                sv->val.s = val ? wolf_req_strdup(val) : wolf_req_strdup("");
                free(val);
                wolf_array_push(arr, sv);
            } else if (*json == '{') {
                const char* start = json;
                int depth = 0;
                while (*json) {
                    if (*json == '{') depth++;
                    else if (*json == '}') { depth--; if (depth == 0) { json++; break; } }
                    json++;
                }
                char* nested = strndup(start, json - start);
                wolf_array_push(arr, wolf_json_decode(nested));
                free(nested);
            } else if (*json == '[') {
                const char* start = json;
                int depth = 0;
                while (*json) {
                    if (*json == '[') depth++;
                    else if (*json == ']') { depth--; if (depth == 0) { json++; break; } }
                    json++;
                }
                char* nested = strndup(start, json - start);
                wolf_array_push(arr, wolf_json_decode(nested));
                free(nested);
            } else if (*json == 't' && strncmp(json, "true", 4) == 0) {
                json += 4;
                wolf_value_t* v = wolf_val_make(WOLF_TYPE_BOOL);
                v->val.b = 1;
                wolf_array_push(arr, v);
            } else if (*json == 'f' && strncmp(json, "false", 5) == 0) {
                json += 5;
                wolf_value_t* v = wolf_val_make(WOLF_TYPE_BOOL);
                v->val.b = 0;
                wolf_array_push(arr, v);
            } else if (*json == 'n' && strncmp(json, "null", 4) == 0) {
                json += 4;
                wolf_array_push(arr, NULL);
            } else {
                // Number
                char num[64] = {0};
                int ni = 0;
                int is_float = 0;
                if (*json == '-') num[ni++] = *json++;
                while (*json && ((*json >= '0' && *json <= '9') ||
                       *json == '.') && ni < 63) {
                    if (*json == '.') is_float = 1;
                    num[ni++] = *json++;
                }
                wolf_value_t* v = wolf_val_make(is_float ? WOLF_TYPE_FLOAT : WOLF_TYPE_INT);
                if (is_float) v->val.f = atof(num);
                else v->val.i = atoll(num);
                wolf_array_push(arr, v);
            }
        }
        return arr;
    }

    if (*json != '{') return wolf_map_create();

    void* map = wolf_map_create();
    json++; // skip '{'

    while (*json) {
        // Skip whitespace
        while (*json && (*json == ' ' || *json == '\t' || *json == '\n' || *json == ',')) json++;
        if (*json == '}') break;
        if (*json != '"') break;
        json++; // skip opening quote

        // Read key
        size_t key_cap = 256;
        char* key = (char*)wolf_req_alloc(key_cap);
        if (!key) break;
        int ki = 0;
        while (*json && *json != '"') {
            if ((size_t)ki >= key_cap - 1) {
                key_cap *= 2;
                char* newkey = (char*)realloc(key, key_cap);
                if (!newkey) { free(key); break; }
                key = newkey;
            }
            key[ki++] = *json++;
        }
        key[ki] = '\0';
        if (*json == '"') json++; // skip closing quote

        // Skip whitespace and colon
        while (*json && (*json == ' ' || *json == '\t' || *json == ':')) json++;

        // Read value
        if (*json == '"') {
            json++; // skip opening quote
            size_t val_cap = 256;
            char* val = (char*)wolf_req_alloc(val_cap);
            if (!val) { free(key); break; }
            int vi = 0;
            while (*json && *json != '"') {
                if ((size_t)vi >= val_cap - 8) {
                    val_cap *= 2;
                    char* newval = (char*)realloc(val, val_cap);
                    if (!newval) { free(val); val = NULL; break; }
                    val = newval;
                }
                if (*json == '\\' && *(json+1)) {
                    json++;
                    switch (*json) {
                        case 'n':  val[vi++] = '\n'; break;
                        case 't':  val[vi++] = '\t'; break;
                        case 'r':  val[vi++] = '\r'; break;
                        case 'b':  val[vi++] = '\b'; break;
                        case 'f':  val[vi++] = '\f'; break;
                        case '"':  val[vi++] = '"';  break;
                        case '\\': val[vi++] = '\\'; break;
                        case '/':  val[vi++] = '/';  break;
                        case 'u': {
                            /* \uXXXX — decode to UTF-8 */
                            json++;
                            unsigned int codepoint = 0;
                            for (int ci = 0; ci < 4 && *json; ci++, json++) {
                                codepoint <<= 4;
                                char hx = *json;
                                if (hx >= '0' && hx <= '9') codepoint |= (hx - '0');
                                else if (hx >= 'a' && hx <= 'f') codepoint |= (hx - 'a' + 10);
                                else if (hx >= 'A' && hx <= 'F') codepoint |= (hx - 'A' + 10);
                            }
                            json--; /* loop will ++ */
                            /* encode as UTF-8 */
                            if (codepoint < 0x80) {
                                val[vi++] = (char)codepoint;
                            } else if (codepoint < 0x800) {
                                val[vi++] = (char)(0xC0 | (codepoint >> 6));
                                val[vi++] = (char)(0x80 | (codepoint & 0x3F));
                            } else {
                                val[vi++] = (char)(0xE0 | (codepoint >> 12));
                                val[vi++] = (char)(0x80 | ((codepoint >> 6) & 0x3F));
                                val[vi++] = (char)(0x80 | (codepoint & 0x3F));
                            }
                            break;
                        }
                        default:  val[vi++] = *json; break;
                    }
                } else {
                    val[vi++] = *json;
                }
                json++;
            }
            if (val) val[vi] = '\0';
            if (*json == '"') json++; // skip closing quote
            /* Wrap as tagged WOLF_TYPE_STRING so json_encode_value identifies it correctly */
            wolf_value_t* sv = wolf_val_make(WOLF_TYPE_STRING);
            sv->val.s = (val && vi > 0) ? wolf_req_strdup(val) : wolf_req_strdup("");
            free(val);
            wolf_map_set(map, key, sv);
        } else if (*json == 't' && strncmp(json, "true", 4) == 0) {
            json += 4;
            wolf_map_set_bool(map, key, 1);
        } else if (*json == 'f' && strncmp(json, "false", 5) == 0) {
            json += 5;
            wolf_map_set_bool(map, key, 0);
        } else if (*json == 'n' && strncmp(json, "null", 4) == 0) {
            json += 4;
            wolf_map_set(map, key, NULL);
        } else if (*json == '[') {
            // Nested array — recurse
            const char* start = json;
            int depth = 0;
            while (*json) {
                if (*json == '[') depth++;
                else if (*json == ']') { depth--; if (depth == 0) { json++; break; } }
                json++;
            }
            char* nested = strndup(start, json - start);
            wolf_value_t* v = wolf_val_make(WOLF_TYPE_ARRAY);
            v->val.ptr = wolf_json_decode(nested);
            free(nested);
            wolf_map_set(map, key, v);
        } else if (*json == '{') {
            // Find the matching closing brace
            int depth = 0;
            const char* start = json;
            while (*json) {
                if (*json == '{') depth++;
                else if (*json == '}') { depth--; if (depth == 0) { json++; break; } }
                json++;
            }
            char* nested = strndup(start, json - start);
            wolf_map_set(map, key, wolf_json_decode(nested));
            free(nested);
        } else {
            // Number
            char num[64] = {0};
            int ni = 0;
            int is_float = 0;
            if (*json == '-') num[ni++] = *json++;
            while (*json && ((*json >= '0' && *json <= '9') || *json == '.') && ni < 63) {
                if (*json == '.') is_float = 1;
                num[ni++] = *json++;
            }
            if (is_float) {
                wolf_map_set_float(map, key, atof(num));
            } else {
                wolf_map_set_int(map, key, atoll(num));
            }
        }
    }
    return map;
}

// ========== Phase 3 Stdlib — Date/Time Extras ==========

int64_t wolf_time_ms() {
    struct timespec ts;
    clock_gettime(CLOCK_REALTIME, &ts);
    return (int64_t)(ts.tv_sec * 1000LL + ts.tv_nsec / 1000000LL);
}

int64_t wolf_time_ns() {
    struct timespec ts;
    clock_gettime(CLOCK_REALTIME, &ts);
    return (int64_t)(ts.tv_sec * 1000000000LL + ts.tv_nsec);
}

int64_t wolf_mktime(int64_t hour, int64_t min, int64_t sec, int64_t mon, int64_t day, int64_t year) {
    struct tm t = {0};
    t.tm_hour = (int)hour;
    t.tm_min = (int)min;
    t.tm_sec = (int)sec;
    t.tm_mon = (int)mon - 1;
    t.tm_mday = (int)day;
    t.tm_year = (int)year - 1900;
    t.tm_isdst = -1;
    return (int64_t)mktime(&t);
}

int64_t wolf_date_diff(int64_t ts1, int64_t ts2) {
    return ts2 > ts1 ? ts2 - ts1 : ts1 - ts2;
}

const char* wolf_date_format(int64_t timestamp, const char* format) {
    return wolf_time_date(format, timestamp);
}

int64_t wolf_day_of_week(int64_t timestamp) {
    time_t t = (time_t)timestamp;
    struct tm* info = localtime(&t);
    return (int64_t)info->tm_wday;
}

int64_t wolf_days_in_month(int64_t month, int64_t year) {
    int days_per_month[] = {31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31};
    if (month < 1 || month > 12) return 0;
    int d = days_per_month[month - 1];
    if (month == 2 && (year % 4 == 0 && (year % 100 != 0 || year % 400 == 0))) d = 29;
    return (int64_t)d;
}

int wolf_is_leap_year(int64_t year) {
    return (year % 4 == 0 && (year % 100 != 0 || year % 400 == 0));
}

int64_t wolf_strtotime(const char* str) {
    // Simple parser for common date formats
    if (!str) return 0;
    struct tm t = {0};
    // Try "YYYY-MM-DD HH:MM:SS"
    if (sscanf(str, "%d-%d-%d %d:%d:%d", &t.tm_year, &t.tm_mon, &t.tm_mday,
               &t.tm_hour, &t.tm_min, &t.tm_sec) >= 3) {
        t.tm_year -= 1900;
        t.tm_mon -= 1;
        t.tm_isdst = -1;
        return (int64_t)mktime(&t);
    }
    // Try "YYYY-MM-DD"
    if (sscanf(str, "%d-%d-%d", &t.tm_year, &t.tm_mon, &t.tm_mday) == 3) {
        t.tm_year -= 1900;
        t.tm_mon -= 1;
        t.tm_isdst = -1;
        return (int64_t)mktime(&t);
    }
    // Special strings
    if (strcmp(str, "now") == 0) return (int64_t)time(NULL);
    if (strcmp(str, "today") == 0) {
        time_t now = time(NULL);
        struct tm* tm_now = localtime(&now);
        tm_now->tm_hour = 0; tm_now->tm_min = 0; tm_now->tm_sec = 0;
        return (int64_t)mktime(tm_now);
    }
    if (strcmp(str, "tomorrow") == 0) {
        return (int64_t)time(NULL) + 86400;
    }
    if (strcmp(str, "yesterday") == 0) {
        return (int64_t)time(NULL) - 86400;
    }
    // "+N days" / "-N days"
    int n;
    if (sscanf(str, "+%d days", &n) == 1) return (int64_t)time(NULL) + n * 86400;
    if (sscanf(str, "-%d days", &n) == 1) return (int64_t)time(NULL) - n * 86400;
    if (sscanf(str, "+%d hours", &n) == 1) return (int64_t)time(NULL) + n * 3600;
    if (sscanf(str, "-%d hours", &n) == 1) return (int64_t)time(NULL) - n * 3600;
    return 0;
}

// ========== Phase 3 Stdlib — Validation ==========

int wolf_is_email(const char* s) {
    if (!s) return 0;
    const char* at = strchr(s, '@');
    if (!at || at == s) return 0;
    const char* dot = strchr(at, '.');
    if (!dot || dot == at + 1 || *(dot + 1) == '\0') return 0;
    // Check no spaces
    for (const char* p = s; *p; p++) {
        if (isspace((unsigned char)*p)) return 0;
    }
    return 1;
}

int wolf_is_url(const char* s) {
    if (!s) return 0;
    return (strncmp(s, "http://", 7) == 0 || strncmp(s, "https://", 8) == 0 ||
            strncmp(s, "ftp://", 6) == 0);
}

int wolf_is_phone(const char* s) {
    if (!s) return 0;
    const char* p = s;
    if (*p == '+') p++;
    int digits = 0;
    while (*p) {
        if (isdigit((unsigned char)*p)) digits++;
        else if (*p != ' ' && *p != '-' && *p != '(' && *p != ')') return 0;
        p++;
    }
    return digits >= 7 && digits <= 15;
}

int wolf_is_uuid(const char* s) {
    if (!s || strlen(s) != 36) return 0;
    // Check format: 8-4-4-4-12
    for (int i = 0; i < 36; i++) {
        if (i == 8 || i == 13 || i == 18 || i == 23) {
            if (s[i] != '-') return 0;
        } else {
            if (!isxdigit((unsigned char)s[i])) return 0;
        }
    }
    return 1;
}

int wolf_is_json(const char* s) {
    if (!s || !*s) return 0;
    // Skip whitespace
    while (isspace((unsigned char)*s)) s++;
    // Must start with { or [
    return *s == '{' || *s == '[';
}

int wolf_is_ip(const char* s) {
    if (!s) return 0;
    int parts = 0, val = 0, has_digit = 0;
    while (*s) {
        if (isdigit((unsigned char)*s)) {
            val = val * 10 + (*s - '0');
            has_digit = 1;
            if (val > 255) return 0;
        } else if (*s == '.') {
            if (!has_digit) return 0;
            parts++; val = 0; has_digit = 0;
        } else return 0;
        s++;
    }
    return has_digit && parts == 3;
}

int wolf_is_alpha(const char* s) {
    if (!s || !*s) return 0;
    while (*s) {
        if (!isalpha((unsigned char)*s)) return 0;
        s++;
    }
    return 1;
}

int wolf_is_alpha_num(const char* s) {
    if (!s || !*s) return 0;
    while (*s) {
        if (!isalnum((unsigned char)*s)) return 0;
        s++;
    }
    return 1;
}

// ========== Phase 3 Stdlib — File System ==========

int wolf_file_exists(const char* path) {
    if (!path) return 0;
    FILE* f = fopen(path, "r");
    if (f) { fclose(f); return 1; }
    return 0;
}

const char* wolf_file_read(const char* path) {
    if (!path) return wolf_req_strdup("");
    FILE* f = fopen(path, "rb");
    if (!f) return wolf_req_strdup("");
    fseek(f, 0, SEEK_END);
    long size = ftell(f);
    fseek(f, 0, SEEK_SET);
    char* buf = (char*)wolf_req_alloc(size + 1);
    fread(buf, 1, size, f);
    buf[size] = '\0';
    fclose(f);
    return buf;
}

int wolf_file_write(const char* path, const char* data) {
    if (!path || !data) return 0;
    FILE* f = fopen(path, "wb");
    if (!f) return 0;
    fwrite(data, 1, strlen(data), f);
    fclose(f);
    return 1;
}

int wolf_file_append(const char* path, const char* data) {
    if (!path || !data) return 0;
    FILE* f = fopen(path, "ab");
    if (!f) return 0;
    fwrite(data, 1, strlen(data), f);
    fclose(f);
    return 1;
}

int wolf_file_delete(const char* path) {
    if (!path) return 0;
    return remove(path) == 0;
}

int64_t wolf_file_size(const char* path) {
    if (!path) return -1;
    FILE* f = fopen(path, "rb");
    if (!f) return -1;
    fseek(f, 0, SEEK_END);
    long size = ftell(f);
    fclose(f);
    return (int64_t)size;
}

const char* wolf_file_extension(const char* path) {
    if (!path) return wolf_req_strdup("");
    const char* dot = strrchr(path, '.');
    if (!dot || dot == path) return wolf_req_strdup("");
    return wolf_req_strdup(dot + 1);
}

const char* wolf_file_basename(const char* path) {
    if (!path) return wolf_req_strdup("");
    const char* slash = strrchr(path, '/');
    if (!slash) return wolf_req_strdup(path);
    return wolf_req_strdup(slash + 1);
}

const char* wolf_file_dirname(const char* path) {
    if (!path) return wolf_req_strdup("");
    const char* slash = strrchr(path, '/');
    if (!slash) return wolf_req_strdup(".");
    size_t len = slash - path;
    char* r = (char*)wolf_req_alloc(len + 1);
    memcpy(r, path, len);
    r[len] = '\0';
    return r;
}

int wolf_dir_exists(const char* path) {
    if (!path) return 0;
    struct stat st;
    return (stat(path, &st) == 0 && S_ISDIR(st.st_mode));
}

// ========== Phase 3 Stdlib — Slug & Truncate ==========

const char* wolf_slug(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len = strlen(s);
    char* r = (char*)wolf_req_alloc(len + 1);
    char* w = r;
    for (size_t i = 0; i < len; i++) {
        unsigned char c = (unsigned char)s[i];
        if (isalnum(c)) *w++ = (char)tolower(c);
        else if (c == ' ' || c == '_') {
            if (w > r && *(w-1) != '-') *w++ = '-';
        }
        // skip other chars (apostrophes, etc.)
    }
    // Remove trailing dash
    if (w > r && *(w-1) == '-') w--;
    *w = '\0';
    return r;
}

const char* wolf_truncate(const char* s, int64_t len, const char* suffix) {
    if (!s) return wolf_req_strdup("");
    if (!suffix) suffix = "...";
    size_t sl = strlen(s);
    if ((int64_t)sl <= len) return wolf_req_strdup(s);
    size_t suf_len = strlen(suffix);
    int64_t cut = len - (int64_t)suf_len;
    if (cut < 0) cut = 0;
    char* r = (char*)wolf_req_alloc(cut + suf_len + 1);
    memcpy(r, s, cut);
    memcpy(r + cut, suffix, suf_len);
    r[cut + suf_len] = '\0';
    return r;
}

// ========== Sanitization Functions ==========

// sanitize_string — removes tags and special chars, like FILTER_SANITIZE_STRING
const char* wolf_sanitize_string(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len = strlen(s);
    char* r = (char*)wolf_req_alloc(len + 1);
    char* w = r;
    int in_tag = 0;
    for (size_t i = 0; i < len; i++) {
        if (s[i] == '<') { in_tag = 1; continue; }
        if (s[i] == '>') { in_tag = 0; continue; }
        if (!in_tag && s[i] != '\'' && s[i] != '"') {
            *w++ = s[i];
        }
    }
    *w = '\0';
    return r;
}

// sanitize_email — removes chars not valid in email
const char* wolf_sanitize_email(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len = strlen(s);
    char* r = (char*)wolf_req_alloc(len + 1);
    char* w = r;
    for (size_t i = 0; i < len; i++) {
        unsigned char c = (unsigned char)s[i];
        // Allow alphanumeric, @, ., _, -, +
        if (isalnum(c) || c == '@' || c == '.' || c == '_' || c == '-' || c == '+') {
            *w++ = s[i];
        }
    }
    *w = '\0';
    return r;
}

// sanitize_url — removes chars not valid in URLs
const char* wolf_sanitize_url(const char* s) {
    if (!s) return wolf_req_strdup("");
    size_t len = strlen(s);
    char* r = (char*)wolf_req_alloc(len + 1);
    char* w = r;
    for (size_t i = 0; i < len; i++) {
        unsigned char c = (unsigned char)s[i];
        if (isalnum(c) || c == ':' || c == '/' || c == '.' || c == '-' || c == '_' ||
            c == '~' || c == '?' || c == '#' || c == '[' || c == ']' || c == '@' ||
            c == '!' || c == '$' || c == '&' || c == '\'' || c == '(' || c == ')' ||
            c == '*' || c == '+' || c == ',' || c == ';' || c == '=' || c == '%') {
            *w++ = s[i];
        }
    }
    *w = '\0';
    return r;
}

// sanitize_int — keeps only digits and leading minus
const char* wolf_sanitize_int(const char* s) {
    if (!s) return wolf_req_strdup("0");
    size_t len = strlen(s);
    char* r = (char*)wolf_req_alloc(len + 1);
    char* w = r;
    for (size_t i = 0; i < len; i++) {
        if (isdigit((unsigned char)s[i]) || (i == 0 && s[i] == '-')) {
            *w++ = s[i];
        }
    }
    *w = '\0';
    if (w == r) { free(r); return wolf_req_strdup("0"); }
    return r;
}

// sanitize_float — keeps digits, minus, and decimal point
const char* wolf_sanitize_float(const char* s) {
    if (!s) return wolf_req_strdup("0");
    size_t len = strlen(s);
    char* r = (char*)wolf_req_alloc(len + 1);
    char* w = r;
    int has_dot = 0;
    for (size_t i = 0; i < len; i++) {
        if (isdigit((unsigned char)s[i]) || (i == 0 && s[i] == '-')) {
            *w++ = s[i];
        } else if (s[i] == '.' && !has_dot) {
            *w++ = s[i];
            has_dot = 1;
        }
    }
    *w = '\0';
    if (w == r) { free(r); return wolf_req_strdup("0"); }
    return r;
}

// ========== JWT (Stub — Phase 4) ==========

// jwt_encode — creates a simple base64-encoded JWT (stub for now)
const char* wolf_jwt_encode(const char* payload, const char* secret) {
    if (!payload || !secret) return wolf_req_strdup("");
    // Stub: return a base64-encoded payload as a mock token
    // Real implementation needs HMAC-SHA256 signing
    const char* header = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9";
    const char* encoded_payload = wolf_base64_encode(payload);
    const char* signature = wolf_base64_encode(secret);
    
    size_t total = strlen(header) + 1 + strlen(encoded_payload) + 1 + strlen(signature) + 1;
    char* token = (char*)wolf_req_alloc(total);
    snprintf(token, total, "%s.%s.%s", header, encoded_payload, signature);
    return token;
}

// jwt_decode — decodes a JWT token (stub for now)
const char* wolf_jwt_decode(const char* token, const char* secret) {
    if (!token || !secret) return NULL;
    // Stub: extract and decode the payload (middle part)
    // Real implementation needs HMAC-SHA256 verification
    const char* first_dot = strchr(token, '.');
    if (!first_dot) return NULL;
    const char* second_dot = strchr(first_dot + 1, '.');
    if (!second_dot) return NULL;
    
    size_t payload_len = second_dot - first_dot - 1;
    char* payload_b64 = (char*)wolf_req_alloc(payload_len + 1);
    memcpy(payload_b64, first_dot + 1, payload_len);
    payload_b64[payload_len] = '\0';
    
    const char* decoded = wolf_base64_decode(payload_b64);
    free(payload_b64);
    return decoded;
}


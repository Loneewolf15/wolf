import re

with open("runtime/wolf_runtime.c", "r") as f:
    code = f.read()

db_mock_start = code.find("// ========== Database (Mock) ==========")
db_mock_end = code.find("// ========== Redis (In-Memory Mock) ==========")

if db_mock_start == -1 or db_mock_end == -1:
    print("Could not find mock db section")
    exit(1)

real_db_c = """// ========== Database (Real MySQL C Bindings) ==========
#include <mysql/mysql.h>

typedef struct {
    MYSQL *conn;
    char *sql;
    MYSQL_RES *last_result;
} WolfDBStmt;

// Helper for simple string replace
static char* wolf_internal_str_replace(const char* orig, const char* rep, const char* with) {
    char *result;
    char *ins;
    char *tmp;
    int len_rep;
    int len_with;
    int len_front;
    int count;

    if (!orig || !rep) return NULL;
    len_rep = strlen(rep);
    if (len_rep == 0) return NULL;
    if (!with) with = "";
    len_with = strlen(with);

    ins = (char *)orig;
    for (count = 0; (tmp = strstr(ins, rep)); ++count) {
        ins = tmp + len_rep;
    }

    tmp = result = malloc(strlen(orig) + (len_with - len_rep) * count + 1);
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

void* wolf_db_connect(const char* host, const char* user, const char* pass, const char* dbname) {
    MYSQL *conn = mysql_init(NULL);
    if (conn == NULL) {
        printf("[WOLF-DB] mysql_init() failed\\n");
        return NULL;
    }
    if (mysql_real_connect(conn, host, user, pass, dbname, 0, NULL, 0) == NULL) {
        printf("[WOLF-DB] Connection failed: %s\\n", mysql_error(conn));
        mysql_close(conn);
        return NULL;
    }
    return conn;
}

void* wolf_db_prepare(void* conn, const char* sql) {
    if (!conn) return NULL;
    WolfDBStmt* stmt = (WolfDBStmt*)malloc(sizeof(WolfDBStmt));
    stmt->conn = (MYSQL*)conn;
    stmt->sql = strdup(sql ? sql : "");
    stmt->last_result = NULL;
    return stmt;
}

void wolf_db_bind(void* stmt_ptr, const char* param, const char* value) {
    if (!stmt_ptr || !param) return;
    WolfDBStmt* stmt = (WolfDBStmt*)stmt_ptr;
    
    if (!value) value = "";
    
    // Escape value
    size_t val_len = strlen(value);
    char *escaped = malloc(val_len * 2 + 1);
    mysql_real_escape_string(stmt->conn, escaped, value, val_len);
    
    // Add quotes around the escaped value
    char *quoted = malloc(strlen(escaped) + 3);
    sprintf(quoted, "'%s'", escaped);
    
    char *new_sql = wolf_internal_str_replace(stmt->sql, param, quoted);
    if (new_sql) {
        free(stmt->sql);
        stmt->sql = new_sql;
    }
    
    free(escaped);
    free(quoted);
}

int64_t wolf_db_execute(void* stmt_ptr) {
    if (!stmt_ptr) return 0;
    WolfDBStmt* stmt = (WolfDBStmt*)stmt_ptr;
    
    if (stmt->last_result) {
        mysql_free_result(stmt->last_result);
        stmt->last_result = NULL;
    }
    
    if (mysql_query(stmt->conn, stmt->sql)) {
        printf("[WOLF-DB] Query failed: %s\\n", mysql_error(stmt->conn));
        return 0; // Failure
    }
    
    stmt->last_result = mysql_store_result(stmt->conn);
    return 1; // Success
}

void* wolf_db_fetch_all(void* stmt_ptr) {
    void* arr = wolf_array_create();
    if (!stmt_ptr) return arr;
    WolfDBStmt* stmt = (WolfDBStmt*)stmt_ptr;
    
    if (!stmt->last_result) return arr;
    
    int num_fields = mysql_num_fields(stmt->last_result);
    MYSQL_FIELD *fields = mysql_fetch_fields(stmt->last_result);
    
    MYSQL_ROW row_data;
    while ((row_data = mysql_fetch_row(stmt->last_result))) {
        unsigned long *lengths = mysql_fetch_lengths(stmt->last_result);
        void *row = wolf_map_create();
        
        for (int i = 0; i < num_fields; i++) {
            if (row_data[i]) {
                char *val = strndup(row_data[i], lengths[i]);
                wolf_map_set(row, fields[i].name, val);
                free(val);
            } else {
                wolf_map_set(row, fields[i].name, NULL);
            }
        }
        wolf_array_push(arr, row);
    }
    
    return arr;
}

void* wolf_db_fetch_one(void* stmt_ptr) {
    if (!stmt_ptr) return wolf_map_create();
    WolfDBStmt* stmt = (WolfDBStmt*)stmt_ptr;
    
    if (!stmt->last_result) return wolf_map_create();
    
    int num_fields = mysql_num_fields(stmt->last_result);
    MYSQL_FIELD *fields = mysql_fetch_fields(stmt->last_result);
    
    MYSQL_ROW row_data = mysql_fetch_row(stmt->last_result);
    if (row_data) {
        unsigned long *lengths = mysql_fetch_lengths(stmt->last_result);
        void *row = wolf_map_create();
        for (int i = 0; i < num_fields; i++) {
            if (row_data[i]) {
                char *val = strndup(row_data[i], lengths[i]);
                wolf_map_set(row, fields[i].name, val);
                free(val);
            } else {
                wolf_map_set(row, fields[i].name, NULL);
            }
        }
        return row;
    }
    return wolf_map_create();
}

int64_t wolf_db_row_count(void* stmt_ptr) {
    if (!stmt_ptr) return 0;
    WolfDBStmt* stmt = (WolfDBStmt*)stmt_ptr;
    if (stmt->last_result) {
        return (int64_t)mysql_num_rows(stmt->last_result);
    }
    return (int64_t)mysql_affected_rows(stmt->conn);
}

int64_t wolf_db_last_insert_id(void* conn_ptr) {
    if (!conn_ptr) return 0;
    MYSQL *conn = (MYSQL*)conn_ptr;
    return (int64_t)mysql_insert_id(conn);
}

void wolf_db_close(void* conn_ptr) {
    if (!conn_ptr) return;
    MYSQL *conn = (MYSQL*)conn_ptr;
    mysql_close(conn);
}

void wolf_db_begin_transaction(void* conn_ptr) {
    if (!conn_ptr) return;
    MYSQL *conn = (MYSQL*)conn_ptr;
    mysql_query(conn, "START TRANSACTION");
}

void wolf_db_commit(void* conn_ptr) {
    if (!conn_ptr) return;
    MYSQL *conn = (MYSQL*)conn_ptr;
    mysql_query(conn, "COMMIT");
}

void wolf_db_rollback(void* conn_ptr) {
    if (!conn_ptr) return;
    MYSQL *conn = (MYSQL*)conn_ptr;
    mysql_query(conn, "ROLLBACK");
}

// ========== Redis (In-Memory Mock) =========="""

new_code = code[:db_mock_start] + real_db_c + code[db_mock_end + len("// ========== Redis (In-Memory Mock) =========="):]

with open("runtime/wolf_runtime.c", "w") as f:
    f.write(new_code)

print("Replaced Database Mock with Real MySQL C Bindings")

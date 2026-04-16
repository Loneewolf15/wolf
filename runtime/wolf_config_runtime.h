/*
 * wolf_config_runtime.h
 * Add an #include of this file at the TOP of wolf_runtime.c, before any
 * other #define blocks. It sets up all the compile-time constants that
 * wolf.config bakes in via -D flags, with safe defaults for every value
 * so the runtime still compiles when built without the config system.
 *
 * Usage in wolf_runtime.c:
 *   #include "wolf_config_runtime.h"   ← first include, before wolf_runtime.h
 *   #include "wolf_runtime.h"
 */

#ifndef WOLF_CONFIG_RUNTIME_H
#define WOLF_CONFIG_RUNTIME_H

/* ---- Database pool ---- */
#ifndef WOLF_DB_POOL_SIZE
#  define WOLF_DB_POOL_SIZE 10
#endif

#ifndef WOLF_DB_POOL_MIN_IDLE
#  define WOLF_DB_POOL_MIN_IDLE 2
#endif

#ifndef WOLF_DB_POOL_TIMEOUT
#  define WOLF_DB_POOL_TIMEOUT 30
#endif

#ifndef WOLF_DB_MAX_RETRIES
#  define WOLF_DB_MAX_RETRIES 3
#endif

/* ---- DB credentials (baked at compile time, never in source) ---- */
#ifndef WOLF_DB_HOST
#  define WOLF_DB_HOST "localhost"
#endif

#ifndef WOLF_DB_PORT
#  define WOLF_DB_PORT 3306
#endif

#ifndef WOLF_DB_NAME
#  define WOLF_DB_NAME ""
#endif

#ifndef WOLF_DB_USER
#  define WOLF_DB_USER ""
#endif

#ifndef WOLF_DB_PASS
#  define WOLF_DB_PASS ""
#endif

/* ---- Server limits ---- */
#ifndef WOLF_MAX_CONCURRENT_REQUESTS
#  define WOLF_MAX_CONCURRENT_REQUESTS 1024
#endif

#ifndef WOLF_MAX_REQUEST_SIZE
#  define WOLF_MAX_REQUEST_SIZE 65536
#endif

#ifndef WOLF_MAX_UPLOADS
#  define WOLF_MAX_UPLOADS 8   /* max multipart file parts per request — set via wolf.config: server.max_uploads */
#endif

/* ---- Mail Configuration ---- */
#ifndef WOLF_MAIL_FROM_EMAIL
#  define WOLF_MAIL_FROM_EMAIL ""
#endif

#ifndef WOLF_MAIL_HOST
#  define WOLF_MAIL_HOST ""
#endif

/* ---- App environment ---- */
#ifndef WOLF_APP_ENV
#  define WOLF_APP_ENV "development"
#endif

#ifndef WOLF_APP_DEBUG
#  define WOLF_APP_DEBUG 0
#endif

/* ---- Derived helpers ---- */

/* wolf_is_production() — evaluates to 1 in production builds.
 * Use this to gate expensive debug logging in the runtime. */
#define wolf_is_production() (strcmp(WOLF_APP_ENV, "production") == 0)

/* wolf_is_debug() — evaluates to 1 when APP_DEBUG=true */
#define wolf_is_debug() (WOLF_APP_DEBUG == 1)

#endif /* WOLF_CONFIG_RUNTIME_H */
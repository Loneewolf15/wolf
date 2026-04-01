#include "wolf_mailer.h"
#include "wolf_runtime.h"
#include "wolf_config_runtime.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <curl/curl.h>

#ifndef WOLF_MAIL_PROVIDER
#define WOLF_MAIL_PROVIDER "smtp"
#endif

#ifndef WOLF_MAIL_HOST
#define WOLF_MAIL_HOST "127.0.0.1"
#endif

#ifndef WOLF_MAIL_PORT
#define WOLF_MAIL_PORT 25
#endif

#ifndef WOLF_MAIL_USER
#define WOLF_MAIL_USER ""
#endif

#ifndef WOLF_MAIL_PASS
#define WOLF_MAIL_PASS ""
#endif

#ifndef WOLF_MAIL_FROM
#define WOLF_MAIL_FROM "noreply@localhost"
#endif

struct upload_status {
    const char *payload;
    size_t bytes_read;
    size_t length;
};

static size_t payload_source(char *ptr, size_t size, size_t nmemb, void *userp) {
    struct upload_status *upload_ctx = (struct upload_status *)userp;
    const char *data;
    size_t room = size * nmemb;

    if ((size == 0) || (nmemb == 0) || ((size*nmemb) < 1)) {
        return 0;
    }

    data = &upload_ctx->payload[upload_ctx->bytes_read];
    size_t len = upload_ctx->length - upload_ctx->bytes_read;

    if (len > 0) {
        if (len > room) len = room;
        memcpy(ptr, data, len);
        upload_ctx->bytes_read += len;
        return len;
    }
    return 0;
}

void wolf_mailer_validate_config(void) {
    if (strlen(WOLF_MAIL_HOST) == 0 && wolf_is_debug()) {
        fprintf(stderr, "[WolfMailer] Warning: WOLF_MAIL_HOST is empty. Mail operations will fail.\n");
    }
}

static int internal_mailer_send(const char* to, const char* subject, const char* body, int is_html) {
    CURL *curl;
    CURLcode res = CURLE_OK;
    struct curl_slist *recipients = NULL;
    struct upload_status upload_ctx = {0};

    curl = curl_easy_init();
    if (!curl) return 0;

    char url[256];
    snprintf(url, sizeof(url), "smtp://%s:%d", WOLF_MAIL_HOST, WOLF_MAIL_PORT);
    curl_easy_setopt(curl, CURLOPT_URL, url);

    if (strlen(WOLF_MAIL_USER) > 0) {
        curl_easy_setopt(curl, CURLOPT_USERNAME, WOLF_MAIL_USER);
        curl_easy_setopt(curl, CURLOPT_PASSWORD, WOLF_MAIL_PASS);
        curl_easy_setopt(curl, CURLOPT_USE_SSL, (long)CURLUSESSL_ALL);
    }

    curl_easy_setopt(curl, CURLOPT_MAIL_FROM, WOLF_MAIL_FROM);
    recipients = curl_slist_append(recipients, to);
    curl_easy_setopt(curl, CURLOPT_MAIL_RCPT, recipients);

    // Build Payload
    size_t max_len = strlen(to) + strlen(WOLF_MAIL_FROM) + strlen(subject) + strlen(body) + 512;
    char *payload_text = malloc(max_len);
    snprintf(payload_text, max_len,
             "To: %s\r\n"
             "From: %s\r\n"
             "Subject: %s\r\n"
             "Content-Type: %s; charset=\"utf-8\"\r\n"
             "\r\n"
             "%s\r\n",
             to, WOLF_MAIL_FROM, subject, is_html ? "text/html" : "text/plain", body);

    upload_ctx.payload = payload_text;
    upload_ctx.length = strlen(payload_text);
    upload_ctx.bytes_read = 0;

    curl_easy_setopt(curl, CURLOPT_READFUNCTION, payload_source);
    curl_easy_setopt(curl, CURLOPT_READDATA, &upload_ctx);
    curl_easy_setopt(curl, CURLOPT_UPLOAD, 1L);

    res = curl_easy_perform(curl);

    if (res != CURLE_OK) {
        fprintf(stderr, "wolf_mailer_send failed: %s\n", curl_easy_strerror(res));
    }

    curl_slist_free_all(recipients);
    curl_easy_cleanup(curl);
    free(payload_text);

    return (res == CURLE_OK) ? 1 : 0;
}

int wolf_mailer_send(const char* to, const char* subject, const char* body) {
    return internal_mailer_send(to, subject, body, 0);
}

int wolf_mailer_send_html(const char* to, const char* subject, const char* html_body) {
    return internal_mailer_send(to, subject, html_body, 1);
}

// Simple template replace helper for wolf_mailer_send_template
// Supports {{key}} syntax replacement using the wolf map structure
int wolf_mailer_send_template(const char* to, const char* subject, const char* template_path, void* data_map) {
    if (!wolf_file_exists(template_path)) {
        return 0;
    }
    const char* tpl_content = wolf_file_read(template_path);
    if (!tpl_content) return 0;

    // Use runtime arena alloc logic to perform replacements
    const char* final_body = tpl_content;
    void* keys = wolf_array_keys(data_map);
    int64_t count = wolf_array_length(keys);
    
    for (int64_t i = 0; i < count; i++) {
        const char* key = (const char*)wolf_array_get(keys, i);
        wolf_value_t* val = wolf_map_get(data_map, key);
        if (val && val->type == WOLF_TYPE_STRING) {
            char placeholder[256];
            snprintf(placeholder, sizeof(placeholder), "{{%s}}", key);
            final_body = wolf_str_replace(placeholder, val->val.s, final_body);
        }
    }
    
    return wolf_mailer_send_html(to, subject, final_body);
}

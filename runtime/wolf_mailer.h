#ifndef WOLF_MAILER_H
#define WOLF_MAILER_H

#include <stdint.h>

/* Mailer Subsystem Initialization */
void wolf_mailer_validate_config(void);

// Returns 1 on success, 0 on failure
int wolf_mailer_send(const char* to, const char* subject, const char* body);
int wolf_mailer_send_html(const char* to, const char* subject, const char* html_body);
int wolf_mailer_send_template(const char* to, const char* subject, const char* template_path, void* data_map);

#endif // WOLF_MAILER_H

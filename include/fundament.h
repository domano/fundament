#pragma once

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef void *fundament_session_ref;

typedef struct {
    int32_t code;
    const char *message; // UTF-8, owned by caller after error populated
} fundament_error;

typedef struct {
    const char *data; // UTF-8, owned by caller until freed via fundament_buffer_free
    int64_t length;
} fundament_buffer;

typedef struct {
    int32_t state;
    int32_t reason;
} fundament_availability;

typedef void (*fundament_stream_cb)(const char *chunk, bool is_final, void *userdata);

fundament_session_ref fundament_session_create(const char *instructions, fundament_error *out_error);
void fundament_session_destroy(fundament_session_ref session);

bool fundament_session_check_availability(fundament_availability *out_availability, fundament_error *out_error);

bool fundament_session_respond(fundament_session_ref session, const char *prompt, const char *options_json, fundament_buffer *out_buffer, fundament_error *out_error);

bool fundament_session_respond_structured(fundament_session_ref session, const char *prompt, const char *schema_json, const char *options_json, fundament_buffer *out_buffer, fundament_error *out_error);

bool fundament_session_stream(fundament_session_ref session, const char *prompt, const char *options_json, fundament_stream_cb callback, void *userdata, fundament_error *out_error);

void fundament_buffer_free(void *buffer);
void fundament_error_free(void *error);

#ifdef __cplusplus
} // extern "C"
#endif

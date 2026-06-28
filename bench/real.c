#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <pthread.h>
#include <sys/socket.h>
#include <netinet/in.h>

#define PORT 8086

const char *http_response_template = 
    "HTTP/1.1 200 OK\r\n"
    "Content-Type: application/json\r\n"
    "Content-Length: %d\r\n"
    "\r\n"
    "%s";

void *handle_client(void *arg) {
    int client_socket = *((int *)arg);
    free(arg);

    char buffer[1024];
    read(client_socket, buffer, sizeof(buffer) - 1);

    // 1. JSON parse (simulated via sscanf on the hardcoded string)
    const char *data = "{\"user_id\": 999, \"action\": \"login\", \"timestamp\": 160000}";
    int user_id, timestamp;
    char action[16];
    sscanf(data, "{\"user_id\": %d, \"action\": \"%[^\"]\", \"timestamp\": %d}", &user_id, action, &timestamp);

    // 2. Arithmetic (factorial logic)
    int res = 1;
    for (int i = 1; i <= 20; i++) {
        res += i * 2;
    }

    // 3. Mock DB
    const char *sql = "SELECT * FROM users WHERE id = 999 LIMIT 10";

    // 4. JSON Encode & Reply
    char json_body[256];
    int body_len = snprintf(json_body, sizeof(json_body), "{\"sql\":\"%s\",\"math_result\":%d}", sql, res);

    char response[512];
    int resp_len = snprintf(response, sizeof(response), http_response_template, body_len, json_body);

    write(client_socket, response, resp_len);
    close(client_socket);
    return NULL;
}

int main() {
    int server_fd;
    struct sockaddr_in address;
    int opt = 1;

    if ((server_fd = socket(AF_INET, SOCK_STREAM, 0)) == 0) {
        perror("socket failed");
        exit(EXIT_FAILURE);
    }

    if (setsockopt(server_fd, SOL_SOCKET, SO_REUSEADDR | SO_REUSEPORT, &opt, sizeof(opt))) {
        perror("setsockopt");
        exit(EXIT_FAILURE);
    }
    
    address.sin_family = AF_INET;
    address.sin_addr.s_addr = INADDR_ANY;
    address.sin_port = htons(PORT);

    if (bind(server_fd, (struct sockaddr *)&address, sizeof(address)) < 0) {
        perror("bind failed");
        exit(EXIT_FAILURE);
    }
    
    if (listen(server_fd, 1000) < 0) {
        perror("listen");
        exit(EXIT_FAILURE);
    }

    while (1) {
        int client_socket = accept(server_fd, NULL, NULL);
        if (client_socket < 0) continue;

        int *new_sock = malloc(sizeof(int));
        *new_sock = client_socket;

        pthread_t thread_id;
        pthread_attr_t attr;
        pthread_attr_init(&attr);
        pthread_attr_setdetachstate(&attr, PTHREAD_CREATE_DETACHED);
        pthread_create(&thread_id, &attr, handle_client, (void *)new_sock);
    }
    return 0;
}

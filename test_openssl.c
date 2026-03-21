#include <openssl/evp.h>
int main() {
    EVP_PKEY_CTX *ctx = NULL;
    EVP_DigestSignUpdate(NULL, NULL, 0);
    return 0;
}

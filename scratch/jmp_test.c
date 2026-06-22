#include <setjmp.h>
#include <stdio.h>

int main() {
    jmp_buf buf;
    if (setjmp(buf) == 0) {
        printf("first time\n");
        longjmp(buf, 1);
    } else {
        printf("second time\n");
    }
    return 0;
}

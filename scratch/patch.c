#include <setjmp.h>
#include <stdio.h>

jmp_buf oom_jump;

void try_alloc(int fail) {
    if (fail) {
        printf("Failing allocation, jumping back\n");
        longjmp(oom_jump, 1);
    }
    printf("Allocation success\n");
}

int main() {
    if (setjmp(oom_jump) == 0) {
        printf("Executing handler\n");
        try_alloc(1);
        printf("This should not be reached\n");
    } else {
        printf("Caught OOM error\n");
    }
    return 0;
}

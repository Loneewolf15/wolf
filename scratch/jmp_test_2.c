#include <stdio.h>
#include <setjmp.h>
#include <stdlib.h>

__thread jmp_buf* current_jump = NULL;

void do_alloc() {
    printf("Doing alloc...\n");
    if (current_jump) {
        longjmp(*current_jump, 1);
    }
}

int main() {
    jmp_buf jump;
    current_jump = &jump;
    
    if (setjmp(jump) == 0) {
        printf("Trying alloc...\n");
        do_alloc();
        printf("After alloc\n");
    } else {
        printf("Caught OOM!\n");
    }
    
    return 0;
}

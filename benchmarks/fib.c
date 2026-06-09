#include <stdio.h>
#include <sys/time.h>

long long fib(long long n) {
    if (n <= 1) return n;
    return fib(n - 1) + fib(n - 2);
}

int main() {
    struct timeval start, end;
    gettimeofday(&start, NULL);
    long long res = fib(35);
    gettimeofday(&end, NULL);
    double ms = (end.tv_sec - start.tv_sec) * 1000.0 + (end.tv_usec - start.tv_usec) / 1000.0;
    printf("Result: %lld\nTime: %.2f ms\n", res, ms);
    return 0;
}

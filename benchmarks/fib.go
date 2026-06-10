package main

import (
	"fmt"
	"time"
)

func fib(n int64) int64 {
	if n <= 1 {
		return n
	}
	return fib(n-1) + fib(n-2)
}
func main() {
	start := time.Now()
	res := fib(35)
	fmt.Printf("Result: %d\nTime: %.2f ms\n", res, float64(time.Since(start).Milliseconds()))
}

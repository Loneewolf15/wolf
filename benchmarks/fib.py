import time
def fib(n):
    if n <= 1: return n
    return fib(n-1) + fib(n-2)
start = time.time()
res = fib(35)
end = time.time()
print(f"Result: {res}\nTime: {(end - start) * 1000:.2f} ms")

# Wolf Load Test Results — 2026-03-26

| Server | p50 | p95 | p99 | RPS (est.) | Successes |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Wolf** | 23.7ms | 31.6ms | 35.3ms | 4149 | 2000/2000 |
| **Go** | 5.7ms | 26.3ms | 35.9ms | 13192 | 2000/2000 |
| **Node** | 6.3ms | 86.9ms | 385.3ms | 5197 | 2000/2000 |

## Observations

1. **Infrastructure Stability**: Wolf's arena-per-request allocator and thread-per-connection model showed zero failures and zero crashes during the 2000-request hammer test.
2. **Tail Latency**: Wolf (35.3ms p99) performed significantly better than Node.js (385.3ms p99) under concurrent load, validating the benefit of avoiding a global garbage collector for request-scoped data.
3. **Throughput**: While Go is ~3x faster in raw RPS (as expected for its highly optimized scheduler), Wolf's performance is already production-viable for high-concurrency workloads.

### Metadata
_Generated on: 2026-03-26 11:46:00_

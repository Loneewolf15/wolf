const http = require("http");

const server = http.createServer((req, res) => {
  // 1. JSON parse
  const data = '{"user_id": 999, "action": "login", "timestamp": 160000}';
  const p = JSON.parse(data);

  // 2. Arithmetic
  let mathRes = 1;
  for (let i = 1; i <= 20; i++) {
    mathRes += i * 2;
  }

  // 3. Mock DB
  const sql = "SELECT * FROM users WHERE id = 999 LIMIT 10";

  // 4. JSON Encode & Reply
  const out = JSON.stringify({ sql: sql, math_result: mathRes });

  res.writeHead(200, { "Content-Type": "application/json" });
  res.end(out);
});

server.listen(8082, () => {});

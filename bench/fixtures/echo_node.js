// Node.js echo server — benchmark baseline
// Run: node echo_node.js
const http = require("http");

const server = http.createServer((req, res) => {
  res.writeHead(200, { "Content-Type": "application/json" });
  res.end(
    JSON.stringify({
      status: "ok",
      server: "node",
      ts: Math.floor(Date.now() / 1000),
    }),
  );
});

server.listen(8092, () => {
  console.log("Node.js echo server on :8092");
});

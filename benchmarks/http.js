const http = require('http');

const server = http.createServer((req, res) => {
    res.writeHead(200, { 'Content-Type': 'text/plain' });
    res.end('Hello from JS Server!');
});

const PORT = 8083;
server.listen(PORT, () => {
    console.log(`JS server listening on port ${PORT}`);
});

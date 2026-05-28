const WebSocket = require('ws');
const ws = new WebSocket('ws://localhost:8080/ws-test-path?query=abc');

ws.on('open', function open() {
  console.log('Connected!');
  ws.send('Hello from Node');
});

ws.on('message', function incoming(data) {
  console.log('Received: %s', data);
  ws.close();
});

ws.on('error', function error(err) {
  console.error('Error: %s', err);
});

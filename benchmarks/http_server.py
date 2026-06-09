from http.server import BaseHTTPRequestHandler, HTTPServer

class RequestHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header('Content-Type', 'text/plain')
        self.end_headers()
        self.wfile.write(b"Hello from Py Server!")
    
    # Suppress logging to keep benchmark clean
    def log_message(self, format, *args):
        pass

def run(server_class=HTTPServer, handler_class=RequestHandler, port=8084):
    server_address = ('', port)
    httpd = server_class(server_address, handler_class)
    print(f'Py server listening on port {port}...')
    httpd.serve_forever()

if __name__ == '__main__':
    run()

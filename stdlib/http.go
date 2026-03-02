// stdlib_http.go defines the HTTP Request/Response objects and the Serve function.
package stdlib

// HttpSource provides the Wolf Native HTTP Server implementation.
const HttpSource = `
# --- Request Class ---
class Request {
    $id: int

    func __construct($req_id: int) {
        $this->id = $req_id
    }

    func method() -> string {
        return wolf_http_req_method($this->id)
    }

    func path() -> string {
        return wolf_http_req_path($this->id)
    }

    func query($key: string) -> string {
        return wolf_http_req_query($this->id, $key)
    }

    func header($key: string) -> string {
        return wolf_http_req_header($this->id, $key)
    }

    func body() -> string {
        return wolf_http_req_body($this->id)
    }
}

# --- Response Class ---
class Response {
    $id: int

    func __construct($res_id: int) {
        $this->id = $res_id
    }

    func header($key: string, $value: string) {
        wolf_http_res_header($this->id, $key, $value)
    }

    func status($code: int) {
        wolf_http_res_status($this->id, $code)
    }
    
    # Writes raw text to the browser
    func text($t: string) {
        $this->header("Content-Type", "text/plain")
        $this->status(200)
        wolf_http_res_write($this->id, $t)
    }
    
    # Writes JSON object to the browser
    func json($data: map) {
        $this->header("Content-Type", "application/json")
        $this->status(200)
        $json_str = json_encode($data)
        wolf_http_res_write($this->id, $json_str)
    }
    
    # Helper to emit standard error responses
    func sendError($code: int, $msg: string) {
        $this->status($code)
        $this->json({"error": $msg})
    }
}
`

import socket
import sys

def test_payload(name, payload, expect_status):
    print(f"--- Testing {name} ---")
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.connect(('127.0.0.1', 8080))
    s.sendall(payload)
    
    response = s.recv(4096)
    s.close()
    
    if not response:
        print("FAIL: No response received")
        return False
        
    status_line = response.split(b'\r\n')[0].decode('utf-8')
    
    if expect_status in status_line:
        print(f"PASS: Received expected {expect_status} -> {status_line}")
        return True
    else:
        print(f"FAIL: Expected {expect_status}, got {status_line}")
        return False

success = True

# Rule 1: CL + TE coexistence -> 400
payload_cl_te = b"POST / HTTP/1.1\r\nHost: localhost\r\nContent-Length: 5\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n"
success &= test_payload("Rule 1 (CL+TE)", payload_cl_te, "400")

# Rule 3: Bad TE value -> 400
payload_bad_te = b"POST / HTTP/1.1\r\nHost: localhost\r\nTransfer-Encoding: xchunked\r\n\r\n"
success &= test_payload("Rule 3 (Bad TE 'xchunked')", payload_bad_te, "400")

# Rule 3: Obfuscated TE -> 400
payload_obf_te = b"POST / HTTP/1.1\r\nHost: localhost\r\nTransfer-Encoding: chunked, trailers\r\n\r\n"
success &= test_payload("Rule 3 (Obfuscated TE 'chunked, trailers')", payload_obf_te, "400")

# Rule 4: Duplicate CL -> 400
payload_dup_cl = b"POST / HTTP/1.1\r\nHost: localhost\r\nContent-Length: 5\r\nContent-Length: 6\r\n\r\n"
success &= test_payload("Rule 4 (Duplicate CL)", payload_dup_cl, "400")

# Rule 5: Bare CR in headers -> 400
payload_bare_cr = b"GET / HTTP/1.1\r\nHost: localhost\r\nX-Bad: this has a bare\rCR\r\n\r\n"
success &= test_payload("Rule 5 (Bare CR)", payload_bare_cr, "400")

# Baseline: Valid chunked -> 200/404/etc (but NOT 400 from parser)
payload_valid_chunked = b"POST / HTTP/1.1\r\nHost: localhost\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n0\r\n\r\n"
success &= test_payload("Baseline Valid TE", payload_valid_chunked, "200") # or 404, just not 400

if not success:
    sys.exit(1)
print("ALL LIVE PROBES PASSED")

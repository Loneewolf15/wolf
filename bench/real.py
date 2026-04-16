import asyncio
import json

async def handle_client(reader, writer):
    while True:
        line = await reader.readline()
        if not line or line == b'\r\n':
            break
            
    # 1. JSON parse
    data = '{"user_id": 999, "action": "login", "timestamp": 160000}'
    p = json.loads(data)

    # 2. Arithmetic
    math_res = 1
    for i in range(1, 21):
        math_res += i * 2

    # 3. Mock DB
    sql = "SELECT * FROM users WHERE id = 999 LIMIT 10"

    # 4. JSON Encode & Reply
    out = json.dumps({"sql": sql, "math_result": math_res})
    
    response = f"HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: {len(out)}\r\n\r\n{out}".encode()
    writer.write(response)
    await writer.drain()
    writer.close()

async def main():
    server = await asyncio.start_server(handle_client, '0.0.0.0', 8083)
    async with server:
        await server.serve_forever()

asyncio.run(main())

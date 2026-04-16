import asyncio

async def handle_client(reader, writer):
    while True:
        line = await reader.readline()
        if not line or line == b'\r\n':
            break
            
    response = b"HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 23\r\n\r\nHello World from Python"
    writer.write(response)
    await writer.drain()
    writer.close()

async def main():
    server = await asyncio.start_server(handle_client, '0.0.0.0', 8083)
    async with server:
        await server.serve_forever()

asyncio.run(main())

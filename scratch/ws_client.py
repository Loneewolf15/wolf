import asyncio
import websockets

async def test_ws():
    uri = "ws://localhost:8080/ws-test-path?query=123"
    async with websockets.connect(uri) as websocket:
        await websocket.send("Hello Wolf!")
        response = await websocket.recv()
        print(f"Server replied: {response}")

if __name__ == "__main__":
    asyncio.run(test_ws())

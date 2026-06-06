import asyncio
import aiohttp
import websockets
import json
import time
from strategy import decide_next_move

API_BASE = "https://api.game.redd.lnstw.xyz"
WS_BASE = "wss://api.game.redd.lnstw.xyz"

class SnakeBot:
    def __init__(self, bot_name: str):
        self.bot_name = bot_name
        self.token = None
        self.ws = None
        self.is_playing = False

    async def login(self) -> bool:
        """非同步登入取得 Token"""
        print(f"[{self.bot_name}] 正在登入...")
        async with aiohttp.ClientSession() as session:
            async with session.post(f"{API_BASE}/api/login/guest", json={"username": self.bot_name}) as resp:
                if resp.status == 200:
                    data = await resp.json()
                    self.token = data.get("token")
                    print(f"[{self.bot_name}] 登入成功")
                    return True
                else:
                    text = await resp.text()
                    print(f"[{self.bot_name}] 登入失敗: {text}")
                    return False

    async def heartbeat(self):
        """定期發送 Ping 保持連線"""
        while True:
            if self.ws and self.ws.open:
                try:
                    await self.ws.send(json.dumps({
                        "type": "ping", 
                        "payload": {"time": int(time.time() * 1000)}
                    }))
                except Exception as e:
                    print(f"[{self.bot_name}] Heartbeat error: {e}")
                    break
            await asyncio.sleep(1.5)

    async def run(self):
        """主連線迴圈"""
        if not await self.login():
            return

        ws_url = f"{WS_BASE}/api/ws?token={self.token}"
        try:
            async with websockets.connect(ws_url) as ws:
                self.ws = ws
                print(f"[{self.bot_name}] WebSocket 連線成功！")
                
                # 啟動心跳任務
                asyncio.create_task(self.heartbeat())
                
                # 發送開局指令
                await self.ws.send(json.dumps({"type": "start_game", "payload": {}}))
                
                # 接收訊息迴圈
                async for message in ws:
                    await self.handle_message(message)
        except Exception as e:
            print(f"[{self.bot_name}] 連線中斷: {e}")

    async def handle_message(self, message: str):
        data = json.loads(message)
        
        if data["type"] == "game_update":
            payload = data["payload"]
            my_snake = payload["snakes"].get(self.bot_name)
            
            if my_snake:
                self.is_playing = True
                move = decide_next_move(
                    my_snake, payload["foods"], payload["snakes"], 
                    payload["cols"], payload["rows"]
                )
                await self.ws.send(json.dumps({"type": "move", "payload": move}))
            else:
                self.is_playing = False

        elif data["type"] == "game_over":
            score = data["payload"].get("score", 0)
            print(f"[{self.bot_name}] 死亡！分數：{score}。3秒後重開...")
            await asyncio.sleep(3)
            await self.ws.send(json.dumps({"type": "start_game", "payload": {}}))
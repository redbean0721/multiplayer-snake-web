import multiprocessing
import asyncio
import random
from client import SnakeBot

def run_bot_process(bot_name: str):
    """這是一個獨立的進程，每個進程都有自己的 Event Loop"""
    bot = SnakeBot(bot_name)
    asyncio.run(bot.run())

if __name__ == "__main__":
    NUM_BOTS = 1
    
    processes = []
    print(f"準備啟動 {NUM_BOTS} 隻陪玩機器人...")
    
    for _ in range(NUM_BOTS):
        bot_name = f"Bot_{random.randint(1000, 9999)}"
        # 建立獨立進程
        p = multiprocessing.Process(target=run_bot_process, args=(bot_name,))
        processes.append(p)
        p.start()

    # 等待所有進程結束 (通常會一直跑)
    for p in processes:
        p.join()
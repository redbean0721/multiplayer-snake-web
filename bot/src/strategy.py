import random

def get_manhattan_distance(p1, p2):
    """計算曼哈頓距離"""
    return abs(p1["x"] - p2["x"]) + abs(p1["y"] - p2["y"])

def is_safe_move(nx: int, ny: int, my_snake: dict, snakes: dict, cols: int, rows: int) -> bool:
    """檢查下一步移動是否安全"""
    if nx < 0 or nx >= cols or ny < 0 or ny >= rows:
        return False  # 超出邊界
    
    if len(my_snake["body"]) > 1:
        neck = my_snake["body"][1]
        if nx == neck["x"] and ny == neck["y"]:
            return False  # 不要移動到自己的脖子
    
    for snake in snakes.values():
        for segment in snake["body"]:
            if nx == segment["x"] and ny == segment["y"]:
                return False  # 不要移動到其他蛇的身體
    
    return True

def decide_next_move(my_snake: dict, foods: list, snakes: dict, cols: int, rows: int) -> dict:
    """決定下一步 (貪婪演算法 + 20% 隨機探索)"""
    head = my_snake['body'][0]
    
    possible_moves = [
        {"x": 0, "y": -1},  # 上
        {"x": 0, "y": 1},   # 下
        {"x": -1, "y": 0},  # 左
        {"x": 1, "y": 0}    # 右
    ]
    
    safe_moves = []
    for move in possible_moves:
        nx = head["x"] + move["x"]
        ny = head["y"] + move["y"]
        if is_safe_move(nx, ny, my_snake, snakes, cols, rows):
            safe_moves.append(move)
    
    if not safe_moves:
        return possible_moves[0]  # 沒有安全的移動，隨便選一個
    
    if random.random() < 0.2:  # 20% 機率隨機選擇
        return random.choice(safe_moves)
    
    # 貪婪選擇：選擇距離最近的食物
    target_food = None
    if foods:
        stars = [f for f in foods if f.get("type") == "stars"]
        if stars and random.random() < 0.5:  # 50% 機率優先選擇星星
            target_food = min(stars, key=lambda f: get_manhattan_distance(head, f))
        else:
            target_food = min(foods, key=lambda f: get_manhattan_distance(head, f))
    
    if target_food:
        best_move = safe_moves[0]
        min_distance = float("inf")
        
        for move in safe_moves:
            nx = head["x"] + move["x"]
            ny = head["y"] + move["y"]
            distance = get_manhattan_distance({"x": nx, "y": ny}, target_food)
            
            if distance < min_distance:
                min_distance = distance
                best_move = move
        return best_move
    
    return random.choice(safe_moves)  # 沒有食物，隨機選擇安全的移動
<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { sendWS, onWS } from '../services/websocket'

const props = defineProps<{
  isInMatch: boolean,
  playerName: string
}>()

const emit = defineEmits<{
  (e: 'update-score', score: number): void
}>()

interface Tile {
  x: number; y: number
  isSnake: boolean; isHead: boolean; isFood: boolean
}

const showcaseRef = ref<HTMLElement | null>(null)
const gameGrid = ref<Tile[][]>([])
const gridCols = ref(0)
const gridRows = ref(0)
const isPlaying = ref(false)

const tileSize = ref(22) 

const initGrid = (cols: number, rows: number) => {
  const grid: Tile[][] = []
  for (let y = 0; y < rows; y++) {
    const row: Tile[] = []
    for (let x = 0; x < cols; x++) {
      row.push({ x, y, isSnake: false, isHead: false, isFood: false })
    }
    grid.push(row)
  }
  return grid
}

const calculateTileSize = () => {
  if (!showcaseRef.value || gridCols.value === 0 || gridRows.value === 0) return
  
  const availableWidth = showcaseRef.value.clientWidth - 32
  const availableHeight = showcaseRef.value.clientHeight - 32

  const maxTileWidth = Math.floor(availableWidth / gridCols.value) - 1
  const maxTileHeight = Math.floor(availableHeight / gridRows.value) - 1

  tileSize.value = Math.max(1, Math.min(maxTileWidth, maxTileHeight))
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
  window.addEventListener('resize', calculateTileSize) 

  onWS('game_update', (payload: any) => {
    // 1. 同步伺服器地圖大小
    if (gridCols.value !== payload.cols || gridRows.value !== payload.rows) {
      gridCols.value = payload.cols
      gridRows.value = payload.rows
      gameGrid.value = initGrid(payload.cols, payload.rows)
      
      nextTick(() => { calculateTileSize() })
    }

    // 2. 清空畫布
    for (let y = 0; y < gridRows.value; y++) {
      for (let x = 0; x < gridCols.value; x++) {
        gameGrid.value[y][x].isSnake = false
        gameGrid.value[y][x].isHead = false
        gameGrid.value[y][x].isFood = false
      }
    }

    // ✨ 3. 畫出所有食物
    if (payload.foods && Array.isArray(payload.foods)) {
      payload.foods.forEach((f: any) => {
        if (f.y >= 0 && f.y < gridRows.value && f.x >= 0 && f.x < gridCols.value) {
          gameGrid.value[f.y][f.x].isFood = true
        }
      })
    }

    // 4. 畫出世界上的所有蛇
    const snakesMap = payload.snakes
    for (const key in snakesMap) {
      const snake = snakesMap[key]
      snake.body.forEach((segment: any, index: number) => {
        if (segment.y >= 0 && segment.y < gridRows.value && segment.x >= 0 && segment.x < gridCols.value) {
          gameGrid.value[segment.y][segment.x].isSnake = true
          if (index === 0) gameGrid.value[segment.y][segment.x].isHead = true
        }
      })
    }

    // 5. 更新狀態
    const mySnake = snakesMap[props.playerName]
    if (mySnake) {
      emit('update-score', mySnake.score)
      isPlaying.value = true
    } else {
      isPlaying.value = false 
    }
  })

  onWS('game_over', (payload: any) => {
    isPlaying.value = false
    alert(`遊戲結束！你這次的得分是：${payload.score}`)
  })
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('resize', calculateTileSize)
})

const startGame = () => {
  if (isPlaying.value) return
  sendWS('start_game', { name: props.playerName })
}

const handleKeydown = (e: KeyboardEvent) => {
  if (e.target instanceof HTMLInputElement) return
  if (['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', ' '].includes(e.key)) e.preventDefault()
  if (!isPlaying.value) return

  let x = 0, y = 0
  switch (e.key) {
    case 'ArrowUp': case 'w': case 'W': x = 0; y = -1; break
    case 'ArrowDown': case 's': case 'S': x = 0; y = 1; break
    case 'ArrowLeft': case 'a': case 'A': x = -1; y = 0; break
    case 'ArrowRight': case 'd': case 'D': x = 1; y = 0; break
    default: return
  }

  sendWS('move', { x, y })
}
</script>

<template>
  <div class="play-area">
    <div class="snake-showcase" ref="showcaseRef">
      <div 
        v-if="gameGrid.length > 0" 
        class="snake-grid" 
        :style="{ '--cols': gridCols, '--rows': gridRows, '--tile-size': tileSize + 'px' }"
      >
        <template v-for="(row, y) in gameGrid" :key="'row-' + y">
          <div
            v-for="(tile, x) in row"
            :key="'tile-' + x + '-' + y"
            class="tile"
            :class="{ 'snake': tile.isSnake, 'head': tile.isHead, 'food': tile.isFood }"
          ></div>
        </template>
      </div>
    </div>

    <div class="bottom-actions">
      <button class="secondary">商店</button>
      <button class="secondary">圖鑑</button>
      <button 
        class="start" 
        @click="startGame"
        :disabled="isPlaying"
        :style="{ opacity: isPlaying ? 0.6 : 1, cursor: isPlaying ? 'not-allowed' : 'pointer' }"
      >
        {{ isPlaying ? '遊戲進行中...' : (isInMatch ? '開始遊戲（房間中）' : '開始遊戲') }}
      </button>
    </div>
  </div>
</template>
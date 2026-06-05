<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { sendWS, onWS } from '../services/websocket'

const props = defineProps<{
  isInMatch: boolean
}>()

const emit = defineEmits<{
  (e: 'update-score', score: number): void
}>()

const TILE_SIZE = 22

interface Tile {
  x: number; y: number
  isSnake: boolean; isHead: boolean; isFood: boolean
}

const showcaseRef = ref<HTMLElement | null>(null)
const gameGrid = ref<Tile[][]>([])
const gridCols = ref(0)
const gridRows = ref(0)

const isPlaying = ref(false)

// 這些資料現在完全由伺服器派發
const snake = ref<{ x: number; y: number }[]>([])
const food = ref<{ x: number; y: number }>({ x: -1, y: -1 })

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

// 根據伺服器傳來的蛇與食物資料來渲染
const renderGrid = () => {
  if (gridCols.value === 0 || gridRows.value === 0) return
  for (let y = 0; y < gridRows.value; y++) {
    for (let x = 0; x < gridCols.value; x++) {
      gameGrid.value[y][x].isSnake = false
      gameGrid.value[y][x].isHead = false
      gameGrid.value[y][x].isFood = false
    }
  }
  if (food.value.y >= 0 && food.value.y < gridRows.value && food.value.x >= 0 && food.value.x < gridCols.value) {
    gameGrid.value[food.value.y][food.value.x].isFood = true
  }
  snake.value.forEach((segment, index) => {
    if (segment.y >= 0 && segment.y < gridRows.value && segment.x >= 0 && segment.x < gridCols.value) {
      gameGrid.value[segment.y][segment.x].isSnake = true
      if (index === 0) gameGrid.value[segment.y][segment.x].isHead = true
    }
  })
}

const calculateGrid = () => {
  if (!showcaseRef.value) return
  const availableWidth = showcaseRef.value.clientWidth - 32
  const availableHeight = showcaseRef.value.clientHeight - 32
  const unitSize = TILE_SIZE + 1
  const cols = Math.floor(availableWidth / unitSize)
  const rows = Math.floor(availableHeight / unitSize)

  if (cols !== gridCols.value || rows !== gridRows.value) {
    gridCols.value = cols
    gridRows.value = rows
    gameGrid.value = initGrid(cols, rows)
    renderGrid()
  }
}

// 接收來自伺服器的指令
onMounted(() => {
  calculateGrid()
  window.addEventListener('resize', calculateGrid)
  window.addEventListener('keydown', handleKeydown)

  // ✨ 接收遊戲更新
  onWS('game_update', (payload: any) => {
    snake.value = payload.snake
    food.value = payload.food
    emit('update-score', payload.score)
    isPlaying.value = true
    renderGrid()
  })

  // ✨ 接收遊戲結束
  onWS('game_over', (payload: any) => {
    isPlaying.value = false
    alert(`遊戲結束！你這次的得分是：${payload.score}`)
  })
})

onUnmounted(() => {
  window.removeEventListener('resize', calculateGrid)
  window.removeEventListener('keydown', handleKeydown)
})

const startGame = () => {
  if (isPlaying.value) return
  if (gridCols.value < 5 || gridRows.value < 5) return alert('視窗太小啦，請放大視窗！')

  // 將開始遊戲的請求與現在的網格大小告訴伺服器
  sendWS('start_game', { cols: gridCols.value, rows: gridRows.value })
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

  // ✨ 將按鍵轉為方向，直接丟給伺服器
  sendWS('move', { x, y })
}
</script>

<template>
  <div class="play-area">
    <div class="snake-showcase" ref="showcaseRef">
      <div 
        v-if="gameGrid.length > 0" 
        class="snake-grid" 
        :style="{ '--cols': gridCols, '--rows': gridRows, '--tile-size': TILE_SIZE + 'px' }"
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
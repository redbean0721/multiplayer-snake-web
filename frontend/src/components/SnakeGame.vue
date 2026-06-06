<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { sendWS, onWS } from '../services/websocket'

const props = defineProps<{
  isInMatch: boolean,
  playerName: string
}>()

// ✨ 新增 open-pokedex 的事件宣告
const emit = defineEmits<{
  (e: 'update-score', score: number): void
  (e: 'open-pokedex'): void 
}>()

interface Tile {
  x: number; y: number
  isSnake: boolean; isHead: boolean; foodType: string
}

const showcaseRef = ref<HTMLElement | null>(null)
const gameGrid = ref<Tile[][]>([])
const gridCols = ref(0)
const gridRows = ref(0)
const isPlaying = ref(false)

const tileSize = ref(22) 

const topPlayers = ref<{ name: string; score: number }[]>([])

const initGrid = (cols: number, rows: number) => {
  const grid: Tile[][] = []
  for (let y = 0; y < rows; y++) {
    const row: Tile[] = []
    for (let x = 0; x < cols; x++) {
      row.push({ x, y, isSnake: false, isHead: false, foodType: '' })
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
    if (gridCols.value !== payload.cols || gridRows.value !== payload.rows) {
      gridCols.value = payload.cols
      gridRows.value = payload.rows
      gameGrid.value = initGrid(payload.cols, payload.rows)
      nextTick(() => { calculateTileSize() })
    }

    for (let y = 0; y < gridRows.value; y++) {
      for (let x = 0; x < gridCols.value; x++) {
        gameGrid.value[y][x].isSnake = false
        gameGrid.value[y][x].isHead = false
        gameGrid.value[y][x].foodType = ''
      }
    }

    if (payload.foods && Array.isArray(payload.foods)) {
      payload.foods.forEach((f: any) => {
        if (f.y >= 0 && f.y < gridRows.value && f.x >= 0 && f.x < gridCols.value) {
          gameGrid.value[f.y][f.x].foodType = f.type 
        }
      })
    }

    const snakesMap = payload.snakes
    const currentPlayers: { name: string; score: number }[] = []

    for (const key in snakesMap) {
      const snake = snakesMap[key]
      currentPlayers.push({ name: key, score: snake.score })

      snake.body.forEach((segment: any, index: number) => {
        if (segment.y >= 0 && segment.y < gridRows.value && segment.x >= 0 && segment.x < gridCols.value) {
          gameGrid.value[segment.y][segment.x].isSnake = true
          if (index === 0) gameGrid.value[segment.y][segment.x].isHead = true
        }
      })
    }

    currentPlayers.sort((a, b) => b.score - a.score)
    topPlayers.value = currentPlayers.slice(0, 5)

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
    alert(`遊戲結束！\n🍎 最終得分：${payload.score}\n🪙 獲得金幣：${payload.coins}`)
  })
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('resize', calculateTileSize)
})

const startGame = () => {
  if (isPlaying.value) return
  sendWS('start_game', {})
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
      
      <div class="leaderboard" v-if="topPlayers.length > 0">
        <h3>🏆 戰況排名</h3>
        <ol>
          <li v-for="(p, index) in topPlayers" :key="p.name" :class="{ 'is-me': p.name === playerName }">
            <span class="rank">#{{ index + 1 }}</span>
            <span class="name">{{ p.name }}</span>
            <span class="score">{{ p.score }}</span>
          </li>
        </ol>
      </div>

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
            :class="{ 
              'snake': tile.isSnake, 
              'head': tile.isHead, 
              'apple': tile.foodType === 'apple',
              'star': tile.foodType === 'star'
            }"
          ></div>
        </template>
      </div>
    </div>

    <div class="bottom-actions">
      <button class="secondary">商店</button>
      <button class="secondary" @click="emit('open-pokedex')">圖鑑</button>
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
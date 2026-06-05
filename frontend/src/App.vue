<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { connectWS, sendWS, onWS } from './services/websocket'
import ChatRoom from './components/ChatRoom.vue'
import SnakeGame from './components/SnakeGame.vue'

type Message = {
  id: number
  user: string
  content: string
  time: string
}

// 玩家與房間狀態
const playerName = ref('redbean0721')
const isInMatch = ref(true)
const activeRoomName = computed(() => (isInMatch.value ? '私人房間 ALPHA-23' : '全伺服器聊天'))
const currentScore = ref(0)

const messages = ref<Message[]>([
  { id: 1, user: '系統', content: '歡迎來到多人貪吃蛇競技場。', time: '01:15' },
  { id: 2, user: 'ewcbor', content: '私人房差一位，要來嗎？', time: '01:16' },
  { id: 3, user: 'ayfp7g', content: '上一場打得很漂亮。', time: '01:16' },
])

// 網路真實延遲計算
const latencyMs = ref(0)
let pingInterval: number | undefined

const latencyLevel = computed<'good' | 'mid' | 'bad'>(() => {
  if (latencyMs.value === 0) return 'mid'
  if (latencyMs.value < 50) return 'good'
  if (latencyMs.value < 120) return 'mid'
  return 'bad'
})

const signalBars = computed(() => {
  const level = latencyLevel.value
  if (level === 'good') return 4
  if (level === 'mid') return 2
  return 1
})

const latencyLabel = computed(() => {
  if (latencyMs.value === 0) return '連接中'
  if (latencyLevel.value === 'good') return '穩定'
  if (latencyLevel.value === 'mid') return '普通'
  return '偏高'
})

onMounted(() => {
  connectWS()

  onWS('ping', (payload: any) => {
    const serverTime = payload.time
    latencyMs.value = Date.now() - serverTime
  })

  pingInterval = window.setInterval(() => {
    sendWS('ping', { time: Date.now() })
  }, 1500)
})

onUnmounted(() => {
  if (pingInterval) clearInterval(pingInterval)
})
</script>

<template>
  <div class="home-root">
    <div class="latency-float" :class="latencyLevel" aria-label="網路延遲狀態">
      <div class="signal-bars" aria-hidden="true">
        <span class="bar" :class="{ on: signalBars >= 1 }"></span>
        <span class="bar" :class="{ on: signalBars >= 2 }"></span>
        <span class="bar" :class="{ on: signalBars >= 3 }"></span>
        <span class="bar" :class="{ on: signalBars >= 4 }"></span>
      </div>
      <p>{{ latencyMs }}ms</p>
      <small>{{ latencyLabel }}</small>
    </div>

    <div class="home-page">
      <header class="topbar">
        <div class="player-chip">
          <div class="avatar">🐍</div>
          <div>
            <p class="name">{{ playerName }}</p>
            <p class="sub">Lv.18</p>
          </div>
        </div>

        <button class="pill sign">七日登入</button>

        <div class="topbar-tools">
          <button class="pill resource">💎 219</button>
          <button class="pill resource">⭐ 261</button>
          <button class="pill resource coin">🪙 562,983</button>
          <button class="pill task">任務</button>
        </div>

        <button class="round topbar-settings">⚙</button>
      </header>

      <div class="content">
        <section class="center-panel">
          
          <div class="panel-top-bar">
            <div class="room-hint">{{ activeRoomName }}</div>
            <div class="score-wrap">
              <div class="score-board">🍎 分數：<span>{{ currentScore }}</span></div>
            </div>
          </div>

          <div class="hero-area">
            <ChatRoom :messages="messages" :player-name="playerName" />

            <SnakeGame 
              :is-in-match="isInMatch"
              @update-score="currentScore = $event"
            />
          </div>
        </section>

        <aside class="side right">
          <button>好友</button>
          <button>排名</button>
        </aside>
      </div>
    </div>
  </div>
</template>
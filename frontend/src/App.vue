<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'

type Message = {
  id: number
  user: string
  content: string
  time: string
}

const playerName = ref('redbean0721')
const isInMatch = ref(true)
const latencyMs = ref(32)

const activeRoomName = computed(() => (isInMatch.value ? '私人房間 ALPHA-23' : '全伺服器聊天'))

const messages = ref<Message[]>([
  { id: 1, user: '系統', content: '歡迎來到多人貪吃蛇競技場。', time: '01:15' },
  { id: 2, user: 'ewcbor', content: '私人房差一位，要來嗎？', time: '01:16' },
  { id: 3, user: 'ayfp7g', content: '上一場打得很漂亮。', time: '01:16' },
])

let latencyTimer: number | undefined

onMounted(() => {
  latencyTimer = window.setInterval(() => {
    const drift = Math.floor(Math.random() * 9) - 4
    latencyMs.value = Math.max(12, Math.min(150, latencyMs.value + drift))
  }, 1400)
})

onUnmounted(() => {
  if (latencyTimer) {
    window.clearInterval(latencyTimer)
  }
})

const latencyLevel = computed<'good' | 'mid' | 'bad'>(() => {
  if (latencyMs.value < 50) return 'good'
  if (latencyMs.value < 95) return 'mid'
  return 'bad'
})

const signalBars = computed(() => {
  const level = latencyLevel.value
  if (level === 'good') return 4
  if (level === 'mid') return 2
  return 1
})

const latencyLabel = computed(() => {
  if (latencyLevel.value === 'good') return '穩定'
  if (latencyLevel.value === 'mid') return '普通'
  return '偏高'
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
          <div class="room-hint">{{ activeRoomName }}</div>

          <div class="hero-area">
            <aside class="chat-mini">
              <ul>
                <li v-for="item in messages" :key="item.id">
                  <span>{{ item.user }}：{{ item.content }}</span>
                  <time>{{ item.time }}</time>
                </li>
              </ul>
              <div class="chat-input-wrap">
                <input type="text" placeholder="聊天（Demo）" disabled />
                <button disabled>送出</button>
              </div>
            </aside>

            <div class="play-area">
              <div class="snake-showcase">
                <span class="snake-part s1"></span>
                <span class="snake-part s2"></span>
                <span class="snake-part s3"></span>
                <span class="snake-head"></span>
              </div>

              <div class="bottom-actions">
                <button class="secondary">商店</button>
                <button class="secondary">圖鑑</button>
                <button class="start" @click="isInMatch = !isInMatch">
                  {{ isInMatch ? '開始遊戲（房間中）' : '開始遊戲' }}
                </button>
              </div>
            </div>
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

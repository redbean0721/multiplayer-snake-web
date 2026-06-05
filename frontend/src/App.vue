<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { connectWS, sendWS, onWS } from './services/websocket'
import ChatRoom from './components/ChatRoom.vue'
import SnakeGame from './components/SnakeGame.vue'

// ==========================================
// 1. 登入系統與狀態保存邏輯
// ==========================================
const isLoggedIn = ref(false)
const guestNameInput = ref('')
const playerName = ref('')

const loginAsGuest = async () => {
  if (!guestNameInput.value.trim()) {
    alert('請輸入使用者名稱！')
    return
  }

  try {
    const res = await fetch('http://10.0.0.110:8080/api/login/guest', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include', // ✨ 允許接收並發送 HttpOnly Cookie
      body: JSON.stringify({ username: guestNameInput.value.trim() })
    })
    
    const data = await res.json()
    if (res.ok) {
      playerName.value = data.username
      isLoggedIn.value = true
      
      // ✨ 存在 localStorage，下次重整就不會斷線了
      localStorage.setItem('game_token', data.token)
      localStorage.setItem('game_username', data.username)
      
      initGameConnection(data.token)
    } else {
      alert(data.error || '登入失敗')
    }
  } catch (error) {
    console.error(error)
    alert('無法連線到伺服器，請確認後端是否正在運行。')
  }
}

// ✨ 登出功能
const logout = () => {
  localStorage.removeItem('game_token')
  localStorage.removeItem('game_username')
  location.reload()
}

// ==========================================
// 2. 遊戲大廳狀態邏輯
// ==========================================
type Message = { id: number; user: string; content: string; time: string }

const isInMatch = ref(true)
const activeRoomName = computed(() => (isInMatch.value ? '私人房間 ALPHA-23' : '全伺服器聊天'))
const currentScore = ref(0)
const messages = ref<Message[]>([{ id: 1, user: '系統', content: '歡迎來到多人貪吃蛇競技場。', time: '01:15' }])

const latencyMs = ref(0)
let pingInterval: number | undefined

const latencyLevel = computed<'good' | 'mid' | 'bad'>(() => {
  if (latencyMs.value === 0) return 'mid'
  if (latencyMs.value < 50) return 'good'
  if (latencyMs.value < 120) return 'mid'
  return 'bad'
})

const signalBars = computed(() => {
  if (latencyLevel.value === 'good') return 4
  if (latencyLevel.value === 'mid') return 2
  return 1
})

const latencyLabel = computed(() => {
  if (latencyMs.value === 0) return '連接中'
  if (latencyLevel.value === 'good') return '穩定'
  return '偏高'
})

const initGameConnection = (token: string) => {
  connectWS(token)

  onWS('ping', (payload: any) => {
    latencyMs.value = Date.now() - payload.time
  })

  pingInterval = window.setInterval(() => {
    sendWS('ping', { time: Date.now() })
  }, 1500)
}

onMounted(() => {
  // ✨ 網頁載入時，自動檢查是否登入過
  const savedToken = localStorage.getItem('game_token')
  const savedUsername = localStorage.getItem('game_username')
  
  if (savedToken && savedUsername) {
    playerName.value = savedUsername
    isLoggedIn.value = true
    initGameConnection(savedToken)
  }
})

onUnmounted(() => {
  if (pingInterval) clearInterval(pingInterval)
})
</script>

<template>
  <div v-if="!isLoggedIn" class="login-screen">
    <div class="login-card">
      <div class="login-header">
        <h1>🐍 貪吃蛇大亂鬥</h1>
        <p>請輸入您的玩家名稱以進入競技場</p>
      </div>

      <input 
        type="text" 
        v-model="guestNameInput" 
        placeholder="例如：redbean0721" 
        @keyup.enter="loginAsGuest"
        maxlength="16"
      />
      
      <button class="pill sign" @click="loginAsGuest">訪客遊玩</button>
      
      <div class="divider"><span>或者</span></div>
      
      <button class="pill discord" disabled>
        Discord 登入 (開發中)
      </button>
    </div>
  </div>

  <div v-else class="home-root">
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
            <p class="sub">Lv.1</p>
          </div>
        </div>

        <button class="pill sign" @click="logout">登出</button>

        <div class="topbar-tools">
          <button class="pill resource">💎 0</button>
          <button class="pill resource">⭐ 0</button>
          <button class="pill resource coin">🪙 0</button>
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
              :player-name="playerName" 
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
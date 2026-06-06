<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { connectWS, sendWS, onWS, disconnectWS, onDisconnectWS } from './services/websocket'
import ChatRoom from './components/ChatRoom.vue'
import SnakeGame from './components/SnakeGame.vue'

// ✨ 將 API 網址統一抽出來管理
const API_BASE = 'https://api.game.redd.lnstw.xyz'

const isLoggedIn = ref(false)
const guestNameInput = ref('')
const playerName = ref('')

const userCoins = ref(0)
const userStars = ref(0)
const userDiamonds = ref(0)

const showRankingModal = ref(false)
const activeRankingTab = ref<'score' | 'wealth'>('score') 
const globalRankings = ref<{ username: string; value: number }[]>([])

// ✨ 將清除狀態抽離成獨立函式，用於被踢下線或驗證失敗時
const clearLocalState = () => {
  localStorage.removeItem('game_token')
  localStorage.removeItem('game_username')
  disconnectWS()
  if (pingInterval) {
    clearInterval(pingInterval)
    pingInterval = undefined
  }
  latencyMs.value = 0
  playerName.value = ''
  isLoggedIn.value = false
}

const loginAsGuest = async () => {
  if (!guestNameInput.value.trim()) return alert('請輸入使用者名稱！')

  try {
    const res = await fetch(`${API_BASE}/api/login/guest`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ username: guestNameInput.value.trim() })
    })
    
    const data = await res.json()
    if (res.ok) {
      completeLogin(data.username, data.token)
    } else {
      alert(data.error || '登入失敗')
    }
  } catch (error) {
    alert('無法連線到伺服器。')
  }
}

const loginWithDiscord = () => {
  window.location.href = `${API_BASE}/api/auth/discord/login`
}

const completeLogin = (username: string, token: string) => {
  playerName.value = username
  isLoggedIn.value = true
  localStorage.setItem('game_token', token)
  localStorage.setItem('game_username', username)
  initGameConnection(token)
}

const logout = async () => {
  try {
    await fetch(`${API_BASE}/api/logout`, { method: 'POST', credentials: 'include' })
  } catch (e) {}
  clearLocalState()
}

const fetchRankings = async (type: 'score' | 'wealth' = 'score') => {
  activeRankingTab.value = type
  try {
    const res = await fetch(`${API_BASE}/api/rankings?type=${type}`)
    if (res.ok) {
      globalRankings.value = await res.json()
      showRankingModal.value = true
    }
  } catch (e) {
    alert('無法取得排行榜')
  }
}

const isInMatch = ref(true)
const activeRoomName = computed(() => (isInMatch.value ? '私人房間 ALPHA-23' : '全伺服器聊天'))
const currentScore = ref(0)

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

  // ✨ 加入 WS 異常斷線監聽
  onDisconnectWS(() => {
    alert('連線異常中斷或登入失效，請重新登入！')
    clearLocalState()
  })

  onWS('ping', (payload: any) => { latencyMs.value = Date.now() - payload.time })
  onWS('resource_update', (payload: any) => {
    userCoins.value = payload.coins
    userStars.value = payload.stars
    userDiamonds.value = payload.diamonds
  })
  pingInterval = window.setInterval(() => { sendWS('ping', { time: Date.now() }) }, 1500)
}

onMounted(async () => {
  const urlParams = new URLSearchParams(window.location.search)
  const urlToken = urlParams.get('token')
  const urlUsername = urlParams.get('username')
  const urlError = urlParams.get('error')

  if (urlError) {
    alert(`Discord 登入失敗：${urlError}`)
    window.history.replaceState({}, document.title, window.location.pathname)
  } else if (urlToken && urlUsername) {
    window.history.replaceState({}, document.title, window.location.pathname)
    completeLogin(urlUsername, urlToken)
    return
  }

  const savedToken = localStorage.getItem('game_token')
  const savedUsername = localStorage.getItem('game_username')
  
  // ✨ 開場時呼叫後端 API，驗證此 Token 是否真的存活在 DB 裡
  if (savedToken && savedUsername) {
    try {
      const res = await fetch(`${API_BASE}/api/me`, { credentials: 'include' })
      if (res.ok) {
        completeLogin(savedUsername, savedToken)
      } else {
        clearLocalState() // 驗證失敗，清除殘留假資料
      }
    } catch (e) {
      clearLocalState() // 伺服器離線，退回登入畫面
    }
  }
})

onUnmounted(() => { if (pingInterval) clearInterval(pingInterval) })
</script>

<template>
  <div v-if="!isLoggedIn" class="login-screen">
    <div class="login-card">
      <div class="login-header"><h1>🐍 貪吃蛇大亂鬥</h1><p>請輸入您的玩家名稱以進入競技場</p></div>
      <input type="text" v-model="guestNameInput" placeholder="例如：player1" @keyup.enter="loginAsGuest" maxlength="16" />
      <button class="pill sign" @click="loginAsGuest">訪客遊玩</button>
      <div class="divider"><span>或者</span></div>
      <button class="pill discord" @click="loginWithDiscord">使用 Discord 登入</button>
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
          <div><p class="name">{{ playerName }}</p><p class="sub">Lv.1</p></div>
        </div>

        <button class="pill sign" @click="logout">登出</button>

        <div class="topbar-tools">
          <button class="pill resource">💎 {{ userDiamonds }}</button>
          <button class="pill resource">⭐ {{ userStars }}</button>
          <button class="pill resource coin">🪙 {{ userCoins }}</button>
          <button class="pill task">任務</button>
        </div>

        <button class="round topbar-settings">⚙</button>
      </header>

      <div class="content">
        <section class="center-panel">
          <div class="panel-top-bar">
            <div class="room-hint">{{ activeRoomName }}</div>
            <div class="score-wrap"><div class="score-board">🍎 分數：<span>{{ currentScore }}</span></div></div>
          </div>
          <div class="hero-area">
            <ChatRoom :player-name="playerName" />
            <SnakeGame :is-in-match="isInMatch" :player-name="playerName" @update-score="currentScore = $event" />
          </div>
        </section>
        <aside class="side right">
          <button>好友</button>
          <button @click="fetchRankings('score')">排名</button>
        </aside>
      </div>
    </div>

    <div v-if="showRankingModal" class="modal-overlay" @click.self="showRankingModal = false">
      <div class="modal-content">
        <h2>🏆 全服排行榜</h2>
        
        <div class="ranking-tabs">
          <button :class="{ active: activeRankingTab === 'score' }" @click="fetchRankings('score')">戰績排名</button>
          <button :class="{ active: activeRankingTab === 'wealth' }" @click="fetchRankings('wealth')">財富排名</button>
        </div>

        <ul class="ranking-list">
          <li v-for="(rank, index) in globalRankings" :key="index" :class="{'is-me': rank.username === playerName}">
            <div class="rank-number">#{{ index + 1 }}</div>
            <div class="rank-name">{{ rank.username }}</div>
            <div class="rank-score">
              {{ rank.value }} 
              {{ activeRankingTab === 'score' ? '分' : '🪙' }}
            </div>
          </li>
          <li v-if="globalRankings.length === 0" class="empty-hint">目前還沒有資料喔</li>
        </ul>
        <button class="close-btn" @click="showRankingModal = false">關閉</button>
      </div>
    </div>
  </div>
</template>
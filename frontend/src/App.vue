<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { connectWS, sendWS, onWS, disconnectWS, onDisconnectWS } from './services/websocket'
import ChatRoom from './components/ChatRoom.vue'
import SnakeGame from './components/SnakeGame.vue'

const API_BASE = import.meta.env.DEV ? 'http://localhost:8080' : 'https://api.game.redd.lnstw.xyz'
const isLoggedIn = ref(false)
const guestNameInput = ref('')
const playerName = ref('')

const userCoins = ref(0)
const userStars = ref(0)
const userDiamonds = ref(0)

const showRankingModal = ref(false)
const activeRankingTab = ref<'score' | 'wealth'>('score') 
const globalRankings = ref<{ username: string; value: number }[]>([])

const showFriendModal = ref(false)
const friendsList = ref<string[]>([])
const pendingInvites = ref<string[]>([]) 
const newFriendInput = ref('')

const showTaskModal = ref(false)
const tasksList = ref<any[]>([])

// ✨ 新增圖鑑狀態
const showPokedexModal = ref(false)

const clearLocalState = () => {
  localStorage.removeItem('game_token')
  localStorage.removeItem('game_username')
  disconnectWS()
  if (pingInterval) { clearInterval(pingInterval); pingInterval = undefined }
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
    if (res.ok) { completeLogin(data.username, data.token) } 
    else { alert(data.error || '登入失敗') }
  } catch (error) { alert('無法連線到伺服器。') }
}

const loginWithDiscord = () => { window.location.href = `${API_BASE}/api/auth/discord/login` }

const completeLogin = (username: string, token: string) => {
  playerName.value = username; isLoggedIn.value = true
  localStorage.setItem('game_token', token); localStorage.setItem('game_username', username)
  initGameConnection(token)
}

const logout = async () => {
  try { await fetch(`${API_BASE}/api/logout`, { method: 'POST', credentials: 'include' }) } catch (e) {}
  clearLocalState()
}

const fetchRankings = async (type: 'score' | 'wealth' = 'score') => {
  activeRankingTab.value = type
  try {
    const res = await fetch(`${API_BASE}/api/rankings?type=${type}`)
    if (res.ok) { globalRankings.value = await res.json(); showRankingModal.value = true }
  } catch (e) { alert('無法取得排行榜') }
}

const fetchFriends = async () => {
  try {
    const res = await fetch(`${API_BASE}/api/friends`, { credentials: 'include' })
    if (res.ok) {
      const data = await res.json()
      friendsList.value = data.friends || []
      pendingInvites.value = data.pending_invites || []
      showFriendModal.value = true
    }
  } catch (e) {}
}

const sendFriendRequest = async () => {
  if (!newFriendInput.value.trim()) return
  try {
    const res = await fetch(`${API_BASE}/api/friends/request`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ friend_name: newFriendInput.value.trim() })
    })
    const data = await res.json()
    alert(data.message || data.error)
    if (res.ok) {
      newFriendInput.value = ''
      fetchFriends() 
    }
  } catch (e) {}
}

const acceptFriend = async (requester: string) => {
  try {
    const res = await fetch(`${API_BASE}/api/friends/accept`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ requester })
    })
    if (res.ok) fetchFriends()
  } catch (e) {}
}

const rejectFriend = async (requester: string) => {
  try {
    const res = await fetch(`${API_BASE}/api/friends/reject`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ requester })
    })
    if (res.ok) fetchFriends()
  } catch (e) {}
}

const removeFriend = async (name: string) => {
  if (!confirm(`確定要將 ${name} 從好友名單移除嗎？`)) return
  try {
    const res = await fetch(`${API_BASE}/api/friends/${name}`, {
      method: 'DELETE',
      credentials: 'include'
    })
    if (res.ok) fetchFriends()
  } catch (e) {}
}

const fetchTasks = async () => {
  try {
    const res = await fetch(`${API_BASE}/api/tasks`, { credentials: 'include' })
    if (res.ok) {
      tasksList.value = await res.json()
      showTaskModal.value = true
    }
  } catch (e) {}
}

const claimTask = async (taskId: string) => {
  try {
    const res = await fetch(`${API_BASE}/api/tasks/claim/${taskId}`, { method: 'POST', credentials: 'include' })
    const data = await res.json()
    if (res.ok) {
      alert(data.message)
      fetchTasks()
      // ✨ 通知後端立刻更新錢包資訊，解決需要 Reload 才更新的問題！
      sendWS('sync_resources', {}) 
    } else {
      alert(data.error)
    }
  } catch (e) {}
}

const isInMatch = ref(true)
const activeRoomName = computed(() => (isInMatch.value ? '私人房間 ALPHA-23' : '全伺服器聊天'))
const currentScore = ref(0)

const latencyMs = ref(0); let pingInterval: number | undefined
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
  onDisconnectWS(() => { alert('連線異常中斷或登入失效，請重新登入！'); clearLocalState() })
  onWS('ping', (payload: any) => { latencyMs.value = Date.now() - payload.time })
  onWS('resource_update', (payload: any) => {
    userCoins.value = payload.coins; userStars.value = payload.stars; userDiamonds.value = payload.diamonds
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
  if (savedToken && savedUsername) {
    try {
      const res = await fetch(`${API_BASE}/api/me`, { credentials: 'include' })
      if (res.ok) { completeLogin(savedUsername, savedToken) } 
      else { clearLocalState() }
    } catch (e) { clearLocalState() }
  }
})

// ✨ 商店系統狀態
const showShopModal = ref(false)
const shopCatalog = ref<any[]>([])
const ownedSkins = ref<string[]>([])
const currentSkin = ref('')

const fetchShop = async () => {
  try {
    const res = await fetch(`${API_BASE}/api/shop`, { credentials: 'include' })
    if (res.ok) {
      const data = await res.json()
      shopCatalog.value = data.catalog
      ownedSkins.value = data.owned
      currentSkin.value = data.current
      showShopModal.value = true
    }
  } catch (e) {}
}

const buySkin = async (skinId: string) => {
  try {
    const res = await fetch(`${API_BASE}/api/shop/buy`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      credentials: 'include', body: JSON.stringify({ skin_id: skinId })
    })
    const data = await res.json()
    alert(data.message || data.error)
    if (res.ok) {
      fetchShop()
      sendWS('sync_resources', {}) // 刷新錢包
    }
  } catch (e) {}
}

const equipSkin = async (skinId: string) => {
  try {
    const res = await fetch(`${API_BASE}/api/shop/equip`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      credentials: 'include', body: JSON.stringify({ skin_id: skinId })
    })
    const data = await res.json()
    if (res.ok) {
      alert('裝備成功！(下一局生效)')
      fetchShop()
    } else { alert(data.error) }
  } catch (e) {}
}

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
          <button class="pill task" @click="fetchTasks">任務</button>
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
            
            <SnakeGame 
              :is-in-match="isInMatch" 
              :player-name="playerName" 
              @update-score="currentScore = $event" 
              @open-pokedex="showPokedexModal = true"
              @open-shop="fetchShop"
            />
            
          </div>
        </section>
        <aside class="side right">
          <button @click="fetchFriends">好友</button>
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
            <div class="rank-score">{{ rank.value }} {{ activeRankingTab === 'score' ? '分' : '🪙' }}</div>
          </li>
          <li v-if="globalRankings.length === 0" class="empty-hint">目前還沒有資料喔</li>
        </ul>
        <button class="close-btn" @click="showRankingModal = false">關閉</button>
      </div>
    </div>

    <div v-if="showFriendModal" class="modal-overlay" @click.self="showFriendModal = false">
      <div class="modal-content">
        <h2>👥 我的聯絡簿</h2>
        <div class="add-friend-wrap">
          <input type="text" v-model="newFriendInput" placeholder="輸入玩家名稱發送邀請" @keyup.enter="sendFriendRequest" />
          <button @click="sendFriendRequest">發送</button>
        </div>
        <div v-if="pendingInvites.length > 0" class="pending-section">
          <h4>🔔 好友邀請</h4>
          <ul class="ranking-list friend-list">
            <li v-for="req in pendingInvites" :key="req">
              <div class="rank-name">{{ req }}</div>
              <div class="action-group">
                <button class="accept-btn" @click="acceptFriend(req)">同意</button>
                <button class="reject-btn" @click="rejectFriend(req)">拒絕</button>
              </div>
            </li>
          </ul>
        </div>
        <h4>👨‍👩‍👧‍👦 我的好友</h4>
        <ul class="ranking-list friend-list">
          <li v-for="friend in friendsList" :key="friend">
            <div class="rank-name">{{ friend }}</div>
            <button class="remove-btn" @click="removeFriend(friend)">刪除</button>
          </li>
          <li v-if="friendsList.length === 0" class="empty-hint">目前沒有好友，快去大廳發送邀請吧！</li>
        </ul>
        <button class="close-btn" @click="showFriendModal = false">關閉</button>
      </div>
    </div>

    <div v-if="showTaskModal" class="modal-overlay" @click.self="showTaskModal = false">
      <div class="modal-content">
        <h2>📜 每日任務</h2>
        <ul class="ranking-list task-list">
          <li v-for="task in tasksList" :key="task.id" class="task-item">
            <div class="task-info">
              <div class="task-desc">{{ task.desc }}</div>
              <div class="task-reward">獎勵：{{ task.reward_text }}</div>
              <div class="progress-bar-bg">
                <div class="progress-bar-fill" :style="{ width: Math.min((task.progress / task.target) * 100, 100) + '%' }"></div>
              </div>
              <div class="progress-text">{{ task.progress }} / {{ task.target }}</div>
            </div>
            <button v-if="task.claimed" class="reject-btn" disabled>已領取</button>
            <button v-else-if="task.progress >= task.target" class="accept-btn claim-glow" @click="claimTask(task.id)">領取獎勵</button>
            <button v-else class="reject-btn" disabled>未達成</button>
          </li>
        </ul>
        <button class="close-btn" @click="showTaskModal = false">關閉</button>
      </div>
    </div>

    <div v-if="showPokedexModal" class="modal-overlay" @click.self="showPokedexModal = false">
      <div class="modal-content">
        <h2>📖 遊戲圖鑑</h2>
        <ul class="ranking-list pokedex-list">
          <li>
            <div class="item-icon apple-icon"></div>
            <div class="item-info">
              <div class="item-name">紅蘋果 <span>(一般)</span></div>
              <div class="item-desc">最常見的食物，吃掉後長度增加，分數 +1，遊戲結束時每 1 分可換取 10 枚 🪙 金幣。</div>
            </div>
          </li>
          <li>
            <div class="item-icon star-icon"></div>
            <div class="item-info">
              <div class="item-name">閃耀星星 <span>(稀有)</span></div>
              <div class="item-desc">10% 機率出現！吃掉後分數 +5，且會讓你的蛇獲得持續 1.8 秒的「2倍速衝刺」效果！</div>
            </div>
          </li>
          <li>
            <div class="item-icon snake-icon">🐍</div>
            <div class="item-info">
              <div class="item-name">敵方玩家 <span>(危險)</span></div>
              <div class="item-desc">運用風騷的走位讓對手的頭撞上你的身體，擊殺對手後，你將立刻獲得 1 顆 💎 鑽石！</div>
            </div>
          </li>
        </ul>
        <button class="close-btn" @click="showPokedexModal = false">關閉</button>
      </div>
    </div>

    <div v-if="showShopModal" class="modal-overlay" @click.self="showShopModal = false">
      <div class="modal-content shop-content">
        <h2>🛍️ 造型商店</h2>
        
        <div class="shop-grid">
          <div v-for="item in shopCatalog" :key="item.id" class="shop-item">
            
            <div class="skin-preview" 
                 :class="{ 
                   'rainbow-snake': item.id === 'rainbow', 
                   'golden-snake': item.id === 'golden' 
                 }"
                 :style="!['rainbow', 'golden'].includes(item.id) ? { backgroundColor: item.id } : {}">
            </div>

            <div class="shop-item-name">{{ item.name }}</div>

            <button v-if="item.id === currentSkin" class="reject-btn" disabled>使用中</button>
            <button v-else-if="ownedSkins.includes(item.id)" class="accept-btn" @click="equipSkin(item.id)">裝備</button>
            <button v-else class="buy-btn" @click="buySkin(item.id)">
              {{ item.price }}
              <span v-if="item.currency === 'coins'">🪙</span>
              <span v-else-if="item.currency === 'stars'">⭐</span>
              <span v-else>💎</span>
            </button>
          </div>
        </div>

        <button class="close-btn" @click="showShopModal = false">關閉</button>
      </div>
    </div>

  </div>
</template>
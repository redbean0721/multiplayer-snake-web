<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { sendWS, onWS } from '../services/websocket';

// 接收初始訊息與玩家名稱
const props = defineProps<{
  messages: { id: number; user: string; content: string; time: string }[],
  playerName: string
}>();

// 建立一個響應式的訊息列表
const localMessages = ref([...props.messages]);

onMounted(() => {
  // 監聽後端廣播的聊天訊息
  onWS('chat', (payload: any) => {
    // 由於 CSS 使用 column-reverse，我們用 unshift 塞到最前面，讓它出現在畫面最底下
    localMessages.value.unshift({ 
      id: payload.id, 
      user: payload.user, 
      content: payload.content, 
      time: payload.time 
    });
  });
});

const chatInput = ref('');

const sendMessage = () => {
  if (chatInput.value.trim() === '') return;
  
  // 發送時帶上玩家名字與內容
  sendWS('chat', { 
    user: props.playerName, 
    content: chatInput.value 
  });
  
  chatInput.value = '';
};
</script>

<template>
  <aside class="chat-mini">
    <ul>
      <li v-for="item in localMessages" :key="item.id">
        <span>{{ item.user }}：{{ item.content }}</span>
        <time>{{ item.time }}</time>
      </li>
    </ul>
    <div class="chat-input-wrap">
      <input type="text" v-model="chatInput" @keyup.enter="sendMessage" placeholder="輸入訊息..." />
      <button @click="sendMessage">送出</button>
    </div>
  </aside>
</template>
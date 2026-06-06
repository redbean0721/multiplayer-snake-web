<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { sendWS, onWS } from '../services/websocket';

const props = defineProps<{
  playerName: string
}>();

const localMessages = ref<{ id: number; user: string; content: string; time: string }[]>([]);

onMounted(() => {
  onWS('chat_history', (payload: any) => {
    localMessages.value = payload || [];
    
    const now = new Date();
    const timeString = `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`;
    
    localMessages.value.unshift({ 
      id: Date.now(), 
      user: '系統', 
      content: '歡迎來到多人貪吃蛇競技場。', 
      time: timeString 
    });
  });

  onWS('chat', (payload: any) => {
    localMessages.value.unshift({ 
      id: payload.id, 
      user: payload.user, 
      content: payload.content, 
      time: payload.time 
    });
    
    if (localMessages.value.length > 30) {
      localMessages.value.pop();
    }
  });
});

const chatInput = ref('');

const sendMessage = () => {
  if (chatInput.value.trim() === '') return;
  
  sendWS('chat', { 
    user: props.playerName, 
    content: chatInput.value 
  });
  
  chatInput.value = '';
};
</script>

<template>
  <aside class="chat-mini">
    <div class="chat-list-wrapper">
      <ul>
        <li v-for="item in localMessages" :key="item.id">
          <span>{{ item.user }}：{{ item.content }}</span>
          <time>{{ item.time }}</time>
        </li>
      </ul>
    </div>
    <div class="chat-input-wrap">
      <input type="text" v-model="chatInput" @keyup.enter="sendMessage" placeholder="輸入訊息..." />
      <button @click="sendMessage">送出</button>
    </div>
  </aside>
</template>
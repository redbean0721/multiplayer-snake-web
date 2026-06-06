let socket: WebSocket | null = null;
const listeners = new Map<string, (payload: any) => void>();
let onDisconnectCallback: (() => void) | null = null;

const WS_BASE = import.meta.env.DEV ? 'ws://localhost:8080' : 'wss://api.game.redd.lnstw.xyz';

export const connectWS = (token: string) => {
  const WS_URL = `${WS_BASE}/api/ws?token=${token}`;
  socket = new WebSocket(WS_URL);
  
  socket.onmessage = (event) => {
    const data = JSON.parse(event.data);
    if (listeners.has(data.type)) {
      listeners.get(data.type)!(data.payload);
    }
  };

  // ✨ 監聽斷線事件
  socket.onclose = () => {
    if (onDisconnectCallback) onDisconnectCallback();
  };
};

export const sendWS = (type: string, payload: any) => {
  if (socket?.readyState === WebSocket.OPEN) {
    socket.send(JSON.stringify({ type, payload }));
  }
};

export const onWS = (type: string, callback: (payload: any) => void) => {
  listeners.set(type, callback);
};

export const onDisconnectWS = (callback: () => void) => {
  onDisconnectCallback = callback;
};

export const disconnectWS = () => {
  if (socket) {
    socket.onclose = null; // 故意登出時不觸發 onDisconnectCallback
    socket.close();
    socket = null;
  }
  listeners.clear();
};
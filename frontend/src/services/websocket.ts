let socket: WebSocket | null = null;
const listeners = new Map<string, (payload: any) => void>();

export const connectWS = (token: string) => {
  const WS_URL = `ws://10.0.0.110:8080/api/ws?token=${token}`;
  socket = new WebSocket(WS_URL);
  
  socket.onmessage = (event) => {
    const data = JSON.parse(event.data);
    if (listeners.has(data.type)) {
      listeners.get(data.type)!(data.payload);
    }
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

// ✨ 新增：登出時主動關閉連線並清理監聽器
export const disconnectWS = () => {
  if (socket) {
    socket.close();
    socket = null;
  }
  listeners.clear();
};
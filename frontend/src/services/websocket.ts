// src/services/websocket.ts
const WS_URL = 'ws://10.0.0.110:8080/api/ws';
let socket: WebSocket | null = null;
const listeners = new Map<string, (payload: any) => void>();

export const connectWS = () => {
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
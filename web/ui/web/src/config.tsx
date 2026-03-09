export const API_URL = import.meta.env.VITE_API_URL;
export const WEBSOCKET_URL = API_URL.replace(/^http/, 'ws')

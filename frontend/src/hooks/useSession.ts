import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type { Session } from '../types';

export function useSession(id: string | null) {
  const [session, setSession] = useState<Session | null>(null);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    setSession(null);
    setError('');
    setConnected(false);
    if (!id) return;
    let disposed = false;
    let socket: WebSocket | undefined;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let streaming = false;

    // REST даёт начальный снимок и резервное чтение при обрыве WebSocket.
    // Поздний REST-ответ не должен затереть более свежий снимок из открытого канала.
    const poll = async () => {
      try {
        const snapshot = await api.session(id);
        if (!disposed && !streaming) { setSession(snapshot); setError(''); }
      } catch (e) {
        if (!disposed) setError(e instanceof Error ? e.message : 'Сеанс недоступен');
      }
    };

    const connect = () => {
      const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
      // Подключение к Go /stream: канал передаёт полные снимки, команды отправляются отдельно через REST.
      socket = new WebSocket(`${protocol}//${location.host}/api/sessions/${id}/stream`);
      socket.onopen = () => { if (!disposed) { streaming = true; setConnected(true); setError(''); } };
      socket.onmessage = (event) => {
        if (!disposed) setSession(JSON.parse(String(event.data)) as Session);
      };
      // После обрыва читаем REST и повторяем подключение; сама симуляция продолжает работать на сервере.
      socket.onclose = () => {
        streaming = false;
        if (disposed) return;
        setConnected(false);
        void poll();
        timer = setTimeout(connect, 2000);
      };
    };

    void poll();
    connect();
    // При смене сеанса закрываем старый канал и игнорируем его незавершённые ответы.
    return () => {
      disposed = true;
      clearTimeout(timer);
      socket?.close();
    };
  }, [id]);

  return { session, setSession, connected, error };
}

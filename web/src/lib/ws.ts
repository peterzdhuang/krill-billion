"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { wsURL } from "./config";
import type { ClientMessage, Envelope } from "./protocol";

const TOKEN_KEY = (lobbyId: string) => `kb_reconnect_${lobbyId}`;
const PLAYER_KEY = (lobbyId: string) => `kb_player_${lobbyId}`;
const HOST_KEY = (lobbyId: string) => `kb_host_${lobbyId}`;

export function loadReconnectToken(lobbyId: string): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(TOKEN_KEY(lobbyId));
}

export function loadHostToken(lobbyId: string): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(HOST_KEY(lobbyId));
}

export function saveReconnect(lobbyId: string, playerId: string, token: string) {
  localStorage.setItem(TOKEN_KEY(lobbyId), token);
  localStorage.setItem(PLAYER_KEY(lobbyId), playerId);
}

export function clearReconnect(lobbyId: string) {
  localStorage.removeItem(TOKEN_KEY(lobbyId));
  localStorage.removeItem(PLAYER_KEY(lobbyId));
}

type Handler = (env: Envelope) => void;

export function useGameSocket(lobbyId: string | null, onMessage: Handler) {
  const [connected, setConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const onMessageRef = useRef(onMessage);
  onMessageRef.current = onMessage;
  const retryRef = useRef(0);
  const intentionalClose = useRef(false);

  const send = useCallback((msg: ClientMessage) => {
    const ws = wsRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) return false;
    ws.send(JSON.stringify(msg));
    return true;
  }, []);

  useEffect(() => {
    if (!lobbyId) return;
    intentionalClose.current = false;
    let timer: ReturnType<typeof setTimeout> | undefined;

    const connect = () => {
      const url = wsURL(lobbyId);
      const ws = new WebSocket(url);
      wsRef.current = ws;

      ws.onopen = () => {
        setConnected(true);
        retryRef.current = 0;
      };
      ws.onclose = () => {
        setConnected(false);
        wsRef.current = null;
        if (!intentionalClose.current) {
          const delay = Math.min(8000, 500 * 2 ** retryRef.current);
          retryRef.current += 1;
          timer = setTimeout(connect, delay);
        }
      };
      ws.onerror = () => {
        ws.close();
      };
      ws.onmessage = (ev) => {
        try {
          const env = JSON.parse(String(ev.data)) as Envelope;
          onMessageRef.current(env);
        } catch {
          /* ignore */
        }
      };
    };

    connect();
    return () => {
      intentionalClose.current = true;
      if (timer) clearTimeout(timer);
      wsRef.current?.close();
    };
  }, [lobbyId]);

  return { connected, send };
}

/** Throttle drafts to ~80ms. Returns draft(text) and flushNow(text?) to send immediately. */
export function useDraftThrottle(
  send: (msg: ClientMessage) => boolean,
  intervalMs = 80,
) {
  const lastSent = useRef(0);
  const pending = useRef<string | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const flush = useCallback(() => {
    timer.current = null;
    if (pending.current === null) return;
    const text = pending.current;
    pending.current = null;
    lastSent.current = Date.now();
    send({ type: "draft", data: { text } });
  }, [send]);

  const draft = useCallback(
    (text: string) => {
      const now = Date.now();
      const elapsed = now - lastSent.current;
      if (elapsed >= intervalMs) {
        lastSent.current = now;
        pending.current = null;
        send({ type: "draft", data: { text } });
        return;
      }
      pending.current = text;
      if (!timer.current) {
        timer.current = setTimeout(flush, intervalMs - elapsed);
      }
    },
    [flush, intervalMs, send],
  );

  /** Send pending text immediately — call before lock so the server has the latest draft. */
  const flushNow = useCallback(
    (text?: string) => {
      if (timer.current) {
        clearTimeout(timer.current);
        timer.current = null;
      }
      const out = text ?? pending.current;
      pending.current = null;
      if (out === null) return;
      lastSent.current = Date.now();
      send({ type: "draft", data: { text: out } });
    },
    [send],
  );

  useEffect(
    () => () => {
      if (timer.current) clearTimeout(timer.current);
    },
    [],
  );

  return { draft, flushNow };
}

"use client";

import { useParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { LobbyPanel } from "@/components/LobbyPanel";
import { MatchScreen } from "@/components/MatchScreen";
import { Toast } from "@/components/Toast";
import type {
  Envelope,
  ErrorMsg,
  JoinAckMsg,
  LobbyStateMsg,
  MatchEndMsg,
  RoundResultMsg,
  RoundStartMsg,
  RoomStateMsg,
  Seat,
} from "@/lib/protocol";
import {
  loadReconnectToken,
  saveReconnect,
  useDraftThrottle,
  useGameSocket,
} from "@/lib/ws";

const NAME_KEY = "kb_display_name";

export default function PlayPage() {
  const params = useParams();
  const lobbyId = String(params.id ?? "");

  const [name, setName] = useState("");
  const [nameReady, setNameReady] = useState(false);
  const [joined, setJoined] = useState(false);
  const [playerId, setPlayerId] = useState<string | null>(null);
  const [isHost, setIsHost] = useState(false);
  const [yourSeat, setYourSeat] = useState<Seat | null>(null);
  const [lobby, setLobby] = useState<LobbyStateMsg | null>(null);
  const [room, setRoom] = useState<RoomStateMsg | null>(null);
  const [round, setRound] = useState<RoundStartMsg | null>(null);
  const [result, setResult] = useState<RoundResultMsg | null>(null);
  const [matchEnd, setMatchEnd] = useState<MatchEndMsg | null>(null);
  const [draft, setDraft] = useState("");
  const [locked, setLocked] = useState(false);
  const [opponentLocked, setOpponentLocked] = useState(false);
  const [toast, setToast] = useState<string | null>(null);
  const [inMatch, setInMatch] = useState(false);

  useEffect(() => {
    const saved = localStorage.getItem(NAME_KEY) ?? "";
    if (saved) {
      setName(saved);
      setNameReady(true);
    }
  }, []);

  const onMessage = useCallback(
    (env: Envelope) => {
      switch (env.type) {
        case "join_ack": {
          const d = env.data as JoinAckMsg;
          setPlayerId(d.player_id);
          setIsHost(d.is_host);
          setYourSeat(d.seat ?? null);
          saveReconnect(lobbyId, d.player_id, d.reconnect_token);
          setJoined(true);
          break;
        }
        case "lobby_state": {
          const d = env.data as LobbyStateMsg;
          setLobby(d);
          if (d.your_seat !== undefined) setYourSeat(d.your_seat ?? null);
          if (d.you_are_host !== undefined) setIsHost(!!d.you_are_host);
          if (d.your_player_id) setPlayerId(d.your_player_id);
          if (!inMatch) setMatchEnd(null);
          break;
        }
        case "room_state": {
          setRoom(env.data as RoomStateMsg);
          setInMatch(true);
          break;
        }
        case "round_start": {
          const d = env.data as RoundStartMsg;
          setRound(d);
          setResult(null);
          setDraft("");
          setLocked(false);
          setOpponentLocked(false);
          setInMatch(true);
          setRoom((r) =>
            r
              ? { ...r, phase: "answering", round: d.round }
              : {
                  players: [],
                  hp: { A: 10000, B: 10000 },
                  multipliers: { A: 1, B: 1 },
                  phase: "answering",
                  round: d.round,
                },
          );
          break;
        }
        case "locked": {
          const d = env.data as { seat: Seat };
          if (d.seat === yourSeat) setLocked(true);
          else setOpponentLocked(true);
          break;
        }
        case "round_result": {
          const d = env.data as RoundResultMsg;
          setResult(d);
          setRoom((r) =>
            r
              ? {
                  ...r,
                  phase: "reveal",
                  hp: d.hp,
                  multipliers: d.multipliers,
                  round: d.round,
                }
              : r,
          );
          break;
        }
        case "match_end": {
          setMatchEnd(env.data as MatchEndMsg);
          setRoom((r) => (r ? { ...r, phase: "ended" } : r));
          break;
        }
        case "error": {
          const d = env.data as ErrorMsg;
          setToast(`${d.code}: ${d.message}`);
          break;
        }
        case "pong":
          break;
        default:
          break;
      }
    },
    [inMatch, lobbyId, yourSeat],
  );

  const { connected, send } = useGameSocket(
    nameReady && lobbyId ? lobbyId : null,
    onMessage,
  );

  useEffect(() => {
    if (!connected || !nameReady || joined) return;
    const token = loadReconnectToken(lobbyId) ?? undefined;
    send({
      type: "join",
      data: {
        lobby_id: lobbyId,
        display_name: name.trim(),
        reconnect_token: token,
      },
    });
  }, [connected, nameReady, joined, lobbyId, name, send]);

  const sendDraft = useDraftThrottle(send);

  const onDraftChange = (text: string) => {
    setDraft(text);
    if (!locked) sendDraft(text);
  };

  const submitName = () => {
    const n = name.trim();
    if (!n) {
      setToast("Enter a display name");
      return;
    }
    localStorage.setItem(NAME_KEY, n);
    setNameReady(true);
  };

  if (!nameReady) {
    return (
      <main className="flex min-h-screen flex-col items-center justify-center px-6">
        <div className="w-full max-w-md space-y-4">
          <h1 className="text-4xl text-cyan-glow">JOIN LOBBY</h1>
          <p className="text-slate-400">{lobbyId}</p>
          <div className="flex items-center gap-2 rounded border border-cyan-900/60 bg-[#02060d] px-3 py-2 text-2xl">
            <span className="text-cyan-accent/70">$</span>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && submitName()}
              className="w-full bg-transparent text-cyan-glow outline-none"
              placeholder="display_name"
              autoFocus
            />
          </div>
          <button
            type="button"
            onClick={submitName}
            className="w-full rounded border border-cyan-accent/40 py-2 text-xl text-cyan-glow"
          >
            CONNECT
          </button>
        </div>
        <Toast message={toast} onDismiss={() => setToast(null)} />
      </main>
    );
  }

  return (
    <main className="relative min-h-screen px-6 py-10">
      <div className="mb-8 flex items-center justify-between text-sm tracking-widest text-cyan-accent/40">
        <span>
          KRILL BILLION
          {playerId ? ` · ${playerId.slice(0, 8)}` : ""}
        </span>
        <span>{connected ? "LINK UP" : "RECONNECTING…"}</span>
      </div>

      {!inMatch && lobby && (
        <LobbyPanel
          state={lobby}
          yourSeat={yourSeat}
          isHost={isHost}
          onClaim={(seat) => send({ type: "claim_seat", data: { seat } })}
          onReady={() => send({ type: "ready", data: {} })}
          onStart={() => send({ type: "start", data: {} })}
          onCopyLink={() => {
            const url = `${window.location.origin}/play/${lobbyId}`;
            void navigator.clipboard.writeText(url);
            setToast("Link copied");
          }}
        />
      )}

      {!inMatch && !lobby && (
        <p className="text-center text-2xl text-slate-500">
          {connected ? "Joining lobby…" : "Connecting…"}
        </p>
      )}

      {inMatch && (
        <MatchScreen
          room={room}
          round={round}
          result={result}
          matchEnd={matchEnd}
          yourSeat={yourSeat}
          draft={draft}
          locked={locked}
          opponentLocked={opponentLocked}
          onDraft={onDraftChange}
          onLock={() => {
            setLocked(true);
            send({ type: "lock", data: {} });
          }}
          onRematch={() => {
            setInMatch(false);
            setMatchEnd(null);
            setRound(null);
            setResult(null);
            setRoom(null);
            window.location.href = "/";
          }}
        />
      )}

      <Toast message={toast} onDismiss={() => setToast(null)} />
    </main>
  );
}

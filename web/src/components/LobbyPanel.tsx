"use client";

import type { LobbyStateMsg, Mode, Seat } from "@/lib/protocol";

const SEATS_1V1: Seat[] = ["A1", "B1"];
const SEATS_2V2: Seat[] = ["A1", "A2", "B1", "B2"];

export function LobbyPanel({
  state,
  yourSeat,
  isHost,
  onClaim,
  onReady,
  onStart,
  onCopyLink,
}: {
  state: LobbyStateMsg;
  yourSeat?: Seat | null;
  isHost: boolean;
  onClaim: (seat: Seat) => void;
  onReady: () => void;
  onStart: () => void;
  onCopyLink: () => void;
}) {
  const seats = state.mode === "2v2" ? SEATS_2V2 : SEATS_1V1;
  const bySeat = new Map(state.seats.map((s) => [s.seat, s]));
  const you = yourSeat ? bySeat.get(yourSeat) : undefined;
  const allReady =
    seats.every((id) => {
      const s = bySeat.get(id);
      return s && !s.empty && s.ready;
    });

  return (
    <div className="mx-auto w-full max-w-2xl space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-sm tracking-[0.3em] text-cyan-accent/50">LOBBY</p>
          <p className="text-3xl text-cyan-glow">{state.lobby_id}</p>
        </div>
        <button
          type="button"
          onClick={onCopyLink}
          className="rounded border border-cyan-900/60 px-3 py-1.5 text-lg text-cyan-accent/80 hover:border-cyan-accent/50"
        >
          COPY LINK
        </button>
      </div>

      <div
        className={`grid gap-3 ${state.mode === "2v2" ? "grid-cols-2" : "grid-cols-2"}`}
      >
        {seats.map((seat) => {
          const info = bySeat.get(seat);
          const empty = !info || info.empty;
          const mine = yourSeat === seat;
          return (
            <button
              key={seat}
              type="button"
              disabled={!empty && !mine}
              onClick={() => onClaim(seat)}
              className={`rounded border px-4 py-5 text-left transition ${
                mine
                  ? "border-cyan-accent/60 bg-cyan-950/40"
                  : empty
                    ? "border-slate-700/70 bg-[#02060d]/60 hover:border-cyan-800"
                    : "cursor-default border-slate-800 bg-slate-950/40 opacity-80"
              }`}
            >
              <div className="text-sm tracking-widest text-cyan-accent/40">
                SEAT {seat}
              </div>
              <div className="mt-1 text-2xl text-slate-100">
                {empty ? "— empty —" : info?.display_name}
              </div>
              {!empty && (
                <div className="mt-2 text-lg text-cyan-accent/70">
                  {info?.ready ? "READY" : "NOT READY"}
                </div>
              )}
            </button>
          );
        })}
      </div>

      <div className="flex flex-wrap gap-3">
        {yourSeat && (
          <button
            type="button"
            onClick={onReady}
            className="rounded border border-cyan-accent/40 px-5 py-2 text-xl text-cyan-glow"
          >
            {you?.ready ? "UNREADY" : "READY UP"}
          </button>
        )}
        {isHost && (
          <button
            type="button"
            disabled={!allReady}
            onClick={onStart}
            className="rounded border border-amber-500/40 px-5 py-2 text-xl text-amber-200 disabled:opacity-40"
          >
            START MATCH
          </button>
        )}
      </div>

      <p className="text-base text-slate-500">
        Mode {(state.mode as Mode).toUpperCase()} · Host starts when every seat
        is filled and ready.
      </p>
    </div>
  );
}

"use client";

import { useMemo } from "react";
import { ClimbingScene } from "@/components/ClimbingScene";
import { KrillionMark } from "@/components/KrillionMark";
import { MuteToggle } from "@/components/MuteToggle";
import { TierBadge } from "@/components/TierBadge";
import { useReducedMotion } from "@/lib/hit";
import type {
  MatchEndData,
  RoomStateData,
  RoundResultData,
  TierName,
} from "@/lib/protocol";

function sideOf(seat?: string | null): 0 | 1 | null {
  if (!seat) return null;
  return seat.startsWith("A") ? 0 : 1;
}

function nameFor(
  room: RoomStateData | null,
  side: 0 | 1,
): string {
  const seatPrefix = side === 0 ? "A" : "B";
  const p = room?.players?.find((x) => x.seat.startsWith(seatPrefix));
  return p?.display_name?.trim() || (side === 0 ? "SIDE A" : "SIDE B");
}

function RoundRecapRow({
  r,
  names,
}: {
  r: RoundResultData;
  names: [string, string];
}) {
  const winnerLabel =
    r.tie || r.winner < 0
      ? "TIE"
      : r.winner === 0
        ? names[0]
        : names[1];
  return (
    <div className="border-b border-slate-800/80 pb-3 last:border-0">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <p className="text-sm tracking-[0.2em] text-cyan-accent/45">
          ROUND {r.round}
          {r.sudden_death ? " · SD" : ""}
        </p>
        <p className="text-lg text-slate-400">
          {r.tie || r.sudden_death
            ? winnerLabel
            : `${winnerLabel} · −${r.damage}`}
        </p>
      </div>
      {r.prompt?.text && (
        <p className="mt-1 text-xl text-slate-300">{r.prompt.text}</p>
      )}
      <div className="mt-2 grid grid-cols-1 gap-2 sm:grid-cols-2">
        {r.sides.map((a, i) => (
          <div key={a.seat} className="text-left text-lg text-slate-400">
            <span className="text-slate-500">{names[i]} · </span>
            {!a.valid ? (
              <span className="text-rose-300/80">miss</span>
            ) : (
              <>
                <span className="text-cyan-glow/90">{a.matched_label}</span>
                {a.tier && (
                  <span className="ml-2 inline-block align-middle">
                    <TierBadge
                      tier={a.tier as TierName}
                      className="scale-90"
                    />
                  </span>
                )}
              </>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

function Confetti({ active }: { active: boolean }) {
  const reduce = useReducedMotion();
  const bits = useMemo(
    () =>
      Array.from({ length: 28 }, (_, i) => ({
        id: i,
        left: `${(i * 37) % 100}%`,
        delay: `${(i % 7) * 0.12}s`,
        dur: `${2.4 + (i % 5) * 0.25}s`,
        hue: (i * 47) % 360,
        size: 6 + (i % 4) * 2,
      })),
    [],
  );
  if (!active || reduce) return null;
  return (
    <div
      className="pointer-events-none absolute inset-0 z-20 overflow-hidden"
      aria-hidden
    >
      {bits.map((b) => (
        <span
          key={b.id}
          className="kb-confetti absolute top-0 rounded-sm"
          style={{
            left: b.left,
            width: b.size,
            height: b.size * 0.6,
            background: `hsl(${b.hue} 70% 60%)`,
            animationDelay: b.delay,
            animationDuration: b.dur,
          }}
        />
      ))}
    </div>
  );
}

export function MatchSummary({
  matchEnd,
  room,
  yourSeat,
  onRematch,
  onBack,
}: {
  matchEnd: MatchEndData;
  room: RoomStateData | null;
  yourSeat?: string | null;
  onRematch: () => void;
  onBack: () => void;
}) {
  const mySide = sideOf(yourSeat);
  const youWin =
    !matchEnd.draw && mySide !== null && matchEnd.winner === mySide;
  const names: [string, string] = [nameFor(room, 0), nameFor(room, 1)];
  const winnerName =
    matchEnd.draw || matchEnd.winner === null
      ? null
      : names[matchEnd.winner];
  const sceneResult =
    matchEnd.recap?.length > 0
      ? matchEnd.recap[matchEnd.recap.length - 1]
      : null;
  const sceneRound = sceneResult?.round ?? room?.round ?? 1;
  const hpA = room?.hp?.[0] ?? sceneResult?.hp?.[0] ?? 0;
  const hpB = room?.hp?.[1] ?? sceneResult?.hp?.[1] ?? 0;
  const reduceMotion = useReducedMotion();

  return (
    <>
      <ClimbingScene round={sceneRound} result={sceneResult} />
      <div className="relative z-10 mx-auto max-w-2xl overflow-hidden rounded bg-[#050a14]/95 px-4 py-6 text-center">
        <Confetti active={youWin || Boolean(winnerName && !matchEnd.draw)} />
        <div className="relative z-30 flex items-center justify-end gap-2">
          <KrillionMark name="krillion" size={32} />
          <MuteToggle />
        </div>

        <div
          className={`relative z-10 mt-4 space-y-3 ${
            youWin && !reduceMotion ? "kb-win-aura" : ""
          }`}
        >
          <p className="text-sm tracking-[0.35em] text-cyan-accent/50">
            MATCH END // {String(matchEnd.reason).toUpperCase()}
          </p>
          <div className="flex items-center justify-center gap-3">
            {(youWin || (!matchEnd.draw && matchEnd.winner !== null)) && (
              <KrillionMark tier="Jeffy B" size={56} aura />
            )}
            <h2 className="text-5xl text-cyan-glow">
              {matchEnd.draw
                ? "DRAW"
                : youWin
                  ? "YOU WIN"
                  : winnerName
                    ? `${winnerName} WINS`
                    : "YOU LOSE"}
            </h2>
          </div>
          <p className="text-xl text-slate-400">
            HP {names[0]} {hpA} · {names[1]} {hpB}
          </p>
        </div>

        <div className="relative z-10 mt-6 max-h-80 space-y-3 overflow-y-auto px-1 text-left">
          <p className="text-sm tracking-[0.25em] text-cyan-accent/40">
            ROUND RECAP
          </p>
          {(matchEnd.recap ?? []).map((r) => (
            <RoundRecapRow key={r.round} r={r} names={names} />
          ))}
        </div>

        <div className="relative z-10 mt-8 flex flex-wrap items-center justify-center gap-3">
          <button
            type="button"
            onClick={onRematch}
            className="min-h-11 rounded border border-cyan-accent/40 px-6 text-2xl text-cyan-glow"
          >
            REMATCH
          </button>
          <button
            type="button"
            onClick={onBack}
            className="min-h-11 rounded border border-slate-600/60 px-6 text-2xl text-slate-300"
          >
            BACK
          </button>
        </div>
      </div>
    </>
  );
}

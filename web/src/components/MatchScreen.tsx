"use client";

import { useMemo } from "react";
import { ClimbingScene } from "@/components/ClimbingScene";
import { DepthMeter } from "@/components/DepthMeter";
import { KrillionMark } from "@/components/KrillionMark";
import { MuteToggle } from "@/components/MuteToggle";
import { RoundTimer } from "@/components/RoundTimer";
import { TierBadge } from "@/components/TierBadge";
import { playSfx } from "@/lib/audio";
import { hitMotion, useReducedMotion } from "@/lib/hit";
import { useMatchSfx } from "@/lib/useMatchSfx";
import type {
  MatchEndData,
  RoundResultData,
  RoundStartData,
  RoomStateData,
  TierName,
} from "@/lib/protocol";

function sideOf(seat?: string | null): 0 | 1 | null {
  if (!seat) return null;
  return seat.startsWith("A") ? 0 : 1;
}

export function MatchScreen({
  room,
  round,
  result,
  matchEnd,
  yourSeat,
  draft,
  suggestion,
  locked,
  opponentLocked,
  fieldError,
  onDraft,
  onAcceptSuggestion,
  onLock,
  onRematch,
}: {
  room: RoomStateData | null;
  round: RoundStartData | null;
  result: RoundResultData | null;
  matchEnd: MatchEndData | null;
  yourSeat?: string | null;
  draft: string;
  suggestion: string;
  locked: boolean;
  opponentLocked: boolean;
  fieldError?: string | null;
  onDraft: (text: string) => void;
  onAcceptSuggestion: () => void;
  onLock: () => void;
  onRematch: () => void;
}) {
  const mySide = sideOf(yourSeat);
  const paused = room?.phase === "paused";

  const hpA = room?.hp?.[0] ?? 10000;
  const hpB = room?.hp?.[1] ?? 10000;
  const multA = room?.multipliers?.[0] ?? 1;
  const multB = room?.multipliers?.[1] ?? 1;

  const revealLines = useMemo(() => {
    if (!result) return [];
    return result.sides.map((a) => {
      const short = a.seat.toLowerCase();
      if (!a.valid) {
        return `${short}  ${a.text || "(empty)"}  rarity —  not in bank  0`;
      }
      return `${short}  ${a.matched_label}  rarity ${a.rarity}  ${a.tier}  ${a.score}`;
    });
  }, [result]);

  useMatchSfx({
    round,
    phase: room?.phase,
    result,
    matchEnd,
    yourSeat,
  });
  const reduceMotion = useReducedMotion();
  const motion = hitMotion(result, room?.phase, mySide);
  const hudShake =
    !reduceMotion && motion.shake
      ? motion.hard
        ? "kb-shake-hard"
        : "kb-shake"
      : "";

  const activeSuggestion =
    !locked && suggestion && suggestion !== draft ? suggestion : "";

  const lockAnswer = () => {
    if (locked) return;
    if (activeSuggestion) onAcceptSuggestion();
    playSfx("submit");
    onLock();
  };

  const acceptOnly = () => {
    if (locked || !activeSuggestion) return;
    onAcceptSuggestion();
  };

  const sceneResult =
    result ??
    (matchEnd?.recap && matchEnd.recap.length > 0
      ? matchEnd.recap[matchEnd.recap.length - 1]
      : null);
  const sceneRound = sceneResult?.round ?? round?.round ?? room?.round ?? 1;

  if (matchEnd) {
    const youWin =
      !matchEnd.draw &&
      mySide !== null &&
      matchEnd.winner === mySide;
    return (
      <>
      <ClimbingScene round={sceneRound} result={sceneResult} />
      <div className="relative z-10 mx-auto max-w-2xl space-y-6 rounded bg-[#050a14] px-4 py-6 text-center">
        <div className="flex items-center justify-end gap-2">
          <KrillionMark name="krillion" size={32} />
          <MuteToggle />
        </div>
        <p className="text-sm tracking-[0.35em] text-cyan-accent/50">
          MATCH END // {String(matchEnd.reason).toUpperCase()}
        </p>
        <h2 className="text-5xl text-cyan-glow">
          {matchEnd.draw ? "DRAW" : youWin ? "YOU WIN" : "YOU LOSE"}
        </h2>
        <p className="text-xl text-slate-400">
          HP A {matchEnd.recap?.length ? hpA : hpA} · B {hpB}
        </p>
        <div className="max-h-64 space-y-2 overflow-y-auto text-left font-mono text-lg text-slate-300">
          {(matchEnd.recap ?? []).map((r) => (
            <div key={r.round} className="border-b border-slate-800/80 pb-2">
              <div className="text-cyan-accent/50">round {r.round}</div>
              {r.sides?.map((a) => (
                <div key={a.seat}>
                  {a.seat}: {a.valid ? a.matched_label : "miss"} ({a.score})
                </div>
              ))}
              <div>damage {r.damage}</div>
            </div>
          ))}
        </div>
        <button
          type="button"
          onClick={onRematch}
          className="min-h-11 rounded border border-cyan-accent/40 px-6 text-2xl text-cyan-glow"
        >
          REMATCH / LOBBY
        </button>
      </div>
      </>
    );
  }

  return (
    <>
    <ClimbingScene
      round={sceneRound}
      result={sceneResult}
      spark={!reduceMotion && motion.spark}
      rung={!reduceMotion && motion.rung}
    />
    {!reduceMotion && motion.flash && (
      <div className="kb-flash pointer-events-none fixed inset-0 z-40 bg-white" />
    )}
    {!reduceMotion && motion.gold && (
      <div className="kb-gold pointer-events-none fixed inset-0 z-40" />
    )}
    <div className={`relative z-10 mx-auto w-full max-w-3xl space-y-6 ${hudShake}`}>
      <div className="relative z-30 flex items-center justify-end gap-2">
        <KrillionMark name="krillion" size={32} />
        <MuteToggle />
      </div>
      {paused && (
        <div className="absolute inset-0 z-20 flex items-center justify-center bg-[#050a14]/85 backdrop-blur-sm">
          <div className="rounded border border-amber-500/40 bg-slate-950/90 px-8 py-6 text-center">
            <p className="text-sm tracking-widest text-amber-400/70">
              RECONNECT WINDOW
            </p>
            <p className="mt-2 text-3xl text-amber-100">
              Opponent reconnecting…
            </p>
          </div>
        </div>
      )}

      {(round?.sudden_death || room?.sudden_death) && (
        <div className="rounded border border-rose-500/50 bg-rose-950/40 py-2 text-center text-2xl text-rose-200">
          SUDDEN DEATH — MULTIPLIERS FROZEN
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <HPBlock label="SIDE A" hp={hpA} mult={multA} highlight={mySide === 0} />
        <HPBlock label="SIDE B" hp={hpB} mult={multB} highlight={mySide === 1} />
      </div>

      <div className="rounded bg-[#050a14] px-3 py-2 text-center">
        <p className="text-sm tracking-[0.3em] text-cyan-accent/40">
          ROUND {room?.round ?? round?.round ?? 0}
        </p>
        <RoundTimer deadlineTs={round?.deadline_ts} phase={room?.phase} />
        <h2 className="mt-2 text-4xl text-cyan-glow sm:text-5xl">
          {round?.prompt.text ?? "Waiting for prompt…"}
        </h2>
        {round?.prompt.category && (
          <p className="mt-1 text-lg text-slate-500">{round.prompt.category}</p>
        )}
      </div>

      {room?.phase === "answering" && (
        <div className="kb-phase space-y-3">
          <div
            className={`flex items-center gap-2 rounded border bg-[#02060d] px-3 py-3 text-2xl ${
              fieldError ? "border-rose-500/70" : "border-cyan-900/60"
            }`}
          >
            <span className="text-cyan-accent/70">$</span>
            <div className="relative min-w-0 flex-1">
              <input
                value={draft}
                disabled={locked}
                onChange={(e) => onDraft(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Tab" && activeSuggestion) {
                    e.preventDefault();
                    acceptOnly();
                    return;
                  }
                  if (e.key === "Enter") {
                    e.preventDefault();
                    lockAnswer();
                  }
                }}
                placeholder="type answer…"
                className="relative z-10 w-full bg-transparent text-cyan-glow outline-none caret-cyan-glow placeholder:text-slate-600 disabled:opacity-60"
                autoFocus
                autoComplete="off"
                spellCheck={false}
              />
              {activeSuggestion && (
                <p
                  className="pointer-events-none absolute inset-0 truncate text-slate-600"
                  aria-hidden
                >
                  <span className="invisible whitespace-pre">{draft}</span>
                  {draft &&
                  activeSuggestion
                    .toLowerCase()
                    .startsWith(draft.toLowerCase())
                    ? activeSuggestion.slice(draft.length)
                    : ""}
                </p>
              )}
            </div>
            {!locked && (
              <span className="inline-block h-5 w-2 animate-pulse bg-cyan-glow/80" />
            )}
          </div>
          {activeSuggestion && (
            <p className="text-lg text-slate-500">
              suggest{" "}
              <span className="text-cyan-accent/80">{activeSuggestion}</span>
              <span className="text-slate-600"> — Tab or Enter</span>
            </p>
          )}
          <div className="flex flex-wrap items-center justify-between gap-2 rounded bg-[#050a14] px-2 py-1 text-lg text-slate-400">
            <span>
              {locked
                ? "LOCKED — waiting for opponent"
                : opponentLocked
                  ? "Opponent locked"
                  : "Opponent drafting…"}
            </span>
            <button
              type="button"
              disabled={locked}
              onClick={lockAnswer}
              className="min-h-11 rounded border border-cyan-accent/40 px-4 text-cyan-glow disabled:opacity-40"
            >
              LOCK (Enter)
            </button>
          </div>
          {fieldError && (
            <p className="text-xl text-rose-300" role="alert">
              {fieldError}
            </p>
          )}
        </div>
      )}

      {room?.phase === "reveal" && result && (
        <div className="kb-phase space-y-4 rounded border border-slate-800/80 bg-[#02060d] p-4">
          <p className="text-sm tracking-widest text-cyan-accent/50">
            REVEAL LOG
          </p>
          {revealLines.map((line, i) => (
            <div key={i} className="text-xl text-slate-200">
              {line}
            </div>
          ))}
          {motion.niche && (
            <p className="text-2xl tracking-widest text-amber-100">NICHE</p>
          )}
          <div className="kb-badge-in flex flex-wrap items-center gap-3">
            {result.sides
              .filter((a) => a.valid && a.tier)
              .map((a) => (
                <TierBadge key={a.seat} tier={a.tier as TierName} />
              ))}
          </div>
          {result.sides.find((a) => a.valid) && (
            <DepthMeter
              rarity={
                result.sides.find((a) => a.seat === yourSeat)?.rarity ??
                result.sides.find((a) => a.valid)?.rarity ??
                0
              }
            />
          )}
          <p className="text-2xl text-amber-200">
            {result.tie
              ? "TIE — both multipliers +0.5"
              : `HIT ${result.damage} on side ${result.winner === 0 ? "B" : "A"}`}
          </p>
        </div>
      )}
    </div>
    </>
  );
}

function HPBlock({
  label,
  hp,
  mult,
  highlight,
}: {
  label: string;
  hp: number;
  mult: number;
  highlight: boolean;
}) {
  const pct = Math.max(0, Math.min(100, (hp / 10000) * 100));
  return (
    <div
      className={`rounded border bg-[#070d16] px-3 py-3 ${
        highlight ? "border-cyan-accent/50" : "border-slate-800"
      }`}
    >
      <div className="mb-1 flex justify-between text-lg">
        <span className="text-slate-400">{label}</span>
        <span className="text-cyan-glow">{mult.toFixed(1)}x</span>
      </div>
      <div className="h-3 overflow-hidden rounded-sm bg-slate-950 ring-1 ring-slate-800">
        <div
          className="h-full bg-gradient-to-r from-teal-600 to-cyan-400 transition-all duration-500"
          style={{ width: `${pct}%` }}
        />
      </div>
      <div className="mt-1 text-right text-lg text-slate-300">{hp} HP</div>
    </div>
  );
}

"use client";

import { useCallback, useMemo, useRef, useState } from "react";
import { ClimbingScene } from "@/components/ClimbingScene";
import { KrillionMark } from "@/components/KrillionMark";
import { MatchSummary } from "@/components/MatchSummary";
import { MuteToggle } from "@/components/MuteToggle";
import {
  preDamageHp,
  RevealPanel,
  type RevealStage,
} from "@/components/RevealPanel";
import { RoundTimer } from "@/components/RoundTimer";
import { playSfx } from "@/lib/audio";
import { hitMotion, useReducedMotion } from "@/lib/hit";
import { useMatchSfx } from "@/lib/useMatchSfx";
import type {
  MatchEndData,
  RoundResultData,
  RoundStartData,
  RoomStateData,
} from "@/lib/protocol";

function sideOf(seat?: string | null): 0 | 1 | null {
  if (!seat) return null;
  return seat.startsWith("A") ? 0 : 1;
}

function playerName(
  room: RoomStateData | null,
  side: 0 | 1,
): string | undefined {
  const prefix = side === 0 ? "A" : "B";
  return room?.players?.find((p) => p.seat.startsWith(prefix))?.display_name;
}

export function MatchScreen({
  room,
  round,
  result,
  matchEnd,
  yourSeat,
  draft,
  locked,
  opponentLocked,
  fieldError,
  onDraft,
  onLock,
  onRematch,
}: {
  room: RoomStateData | null;
  round: RoundStartData | null;
  result: RoundResultData | null;
  matchEnd: MatchEndData | null;
  yourSeat?: string | null;
  draft: string;
  locked: boolean;
  opponentLocked: boolean;
  fieldError?: string | null;
  onDraft: (text: string) => void;
  onLock: () => void;
  onRematch: () => void;
}) {
  const draftRef = useRef(draft);
  const lockedRef = useRef(locked);
  const onLockRef = useRef(onLock);
  draftRef.current = draft;
  lockedRef.current = locked;
  onLockRef.current = onLock;
  const mySide = sideOf(yourSeat);
  const paused = room?.phase === "paused";
  const [revealStage, setRevealStage] = useState<RevealStage>("idle");

  const onRevealStage = useCallback((stage: RevealStage) => {
    setRevealStage(stage);
  }, []);

  useMatchSfx({
    round,
    phase: room?.phase,
    matchEnd,
    yourSeat,
  });
  const reduceMotion = useReducedMotion();
  const hitReady =
    room?.phase === "reveal" &&
    result &&
    (revealStage === "hit" || reduceMotion);
  const motion = hitMotion(hitReady ? result : null, room?.phase, mySide);
  const hudShake =
    !reduceMotion && motion.shake
      ? motion.hard
        ? "kb-shake-hard"
        : "kb-shake"
      : "";

  const displayHp = useMemo((): [number, number] => {
    if (room?.phase === "reveal" && result && revealStage !== "hit") {
      return preDamageHp(result);
    }
    return [
      room?.hp?.[0] ?? 10000,
      room?.hp?.[1] ?? 10000,
    ];
  }, [room?.phase, room?.hp, result, revealStage]);

  const damageTarget =
    hitReady && result && !result.tie && result.damage > 0
      ? result.winner === 0
        ? 1
        : result.winner === 1
          ? 0
          : null
      : null;

  const multA = room?.multipliers?.[0] ?? 1;
  const multB = room?.multipliers?.[1] ?? 1;

  const lockAnswer = () => {
    if (locked) return;
    playSfx("submit");
    onLock();
  };

  const sceneResult =
    result ??
    (matchEnd?.recap && matchEnd.recap.length > 0
      ? matchEnd.recap[matchEnd.recap.length - 1]
      : null);
  const sceneRound = sceneResult?.round ?? round?.round ?? room?.round ?? 1;

  const names = {
    a: playerName(room, 0),
    b: playerName(room, 1),
  };

  if (matchEnd) {
    return (
      <MatchSummary
        matchEnd={matchEnd}
        room={room}
        yourSeat={yourSeat}
        onRematch={onRematch}
        onBack={() => {
          if (typeof window !== "undefined" && window.history.length > 1) {
            window.history.back();
            return;
          }
          onRematch();
        }}
      />
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
      <div
        className={`relative z-10 mx-auto w-full max-w-3xl space-y-6 ${hudShake}`}
      >
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
          <HPBlock
            label={names.a?.trim() || "SIDE A"}
            hp={displayHp[0]}
            mult={multA}
            highlight={mySide === 0}
            drain={damageTarget === 0}
            damagePop={
              damageTarget === 0 && result ? result.damage : undefined
            }
            reduceMotion={reduceMotion}
          />
          <HPBlock
            label={names.b?.trim() || "SIDE B"}
            hp={displayHp[1]}
            mult={multB}
            highlight={mySide === 1}
            drain={damageTarget === 1}
            damagePop={
              damageTarget === 1 && result ? result.damage : undefined
            }
            reduceMotion={reduceMotion}
          />
        </div>

        <div className="rounded bg-[#050a14] px-3 py-2 text-center">
          <p className="text-sm tracking-[0.3em] text-cyan-accent/40">
            ROUND {room?.round ?? round?.round ?? 0}
          </p>
          <RoundTimer
            deadlineTs={round?.deadline_ts}
            phase={room?.phase}
            onTimeout={() => {
              if (lockedRef.current) return;
              if (!draftRef.current.trim()) return;
              playSfx("submit");
              onLockRef.current();
            }}
          />
          <h2 className="mt-2 text-4xl text-cyan-glow sm:text-5xl">
            {round?.prompt.text ?? "Waiting for prompt…"}
          </h2>
          {round?.prompt.category && (
            <p className="mt-1 text-lg text-slate-500">
              {round.prompt.category}
            </p>
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
              <input
                value={draft}
                disabled={locked}
                onChange={(e) => onDraft(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    lockAnswer();
                  }
                }}
                placeholder="type answer…"
                className="w-full bg-transparent text-cyan-glow outline-none caret-cyan-glow placeholder:text-slate-600 disabled:opacity-60"
                autoFocus
                autoComplete="off"
                spellCheck={false}
              />
              {!locked && (
                <span className="inline-block h-5 w-2 animate-pulse bg-cyan-glow/80" />
              )}
            </div>
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
          <RevealPanel
            result={
              result.prompt
                ? result
                : {
                    ...result,
                    prompt: round?.prompt,
                  }
            }
            yourSeat={yourSeat}
            names={names}
            onStage={onRevealStage}
          />
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
  drain,
  damagePop,
  reduceMotion,
}: {
  label: string;
  hp: number;
  mult: number;
  highlight: boolean;
  drain?: boolean;
  damagePop?: number;
  reduceMotion?: boolean;
}) {
  const pct = Math.max(0, Math.min(100, (hp / 10000) * 100));
  return (
    <div
      className={`relative rounded border bg-[#070d16] px-3 py-3 ${
        highlight ? "border-cyan-accent/50" : "border-slate-800"
      } ${drain && !reduceMotion ? "kb-hp-hit" : ""}`}
    >
      <div className="mb-1 flex justify-between text-lg">
        <span className="text-slate-400">{label}</span>
        <span className="text-cyan-glow">{mult.toFixed(1)}x</span>
      </div>
      <div className="h-3 overflow-hidden rounded-sm bg-slate-950 ring-1 ring-slate-800">
        <div
          className={`h-full bg-gradient-to-r from-teal-600 to-cyan-400 ${
            reduceMotion ? "" : "transition-all duration-700 ease-out"
          }`}
          style={{ width: `${pct}%` }}
        />
      </div>
      <div className="mt-1 flex items-center justify-end gap-2 text-lg text-slate-300">
        {typeof damagePop === "number" && damagePop > 0 && (
          <span
            className={`text-rose-300 ${reduceMotion ? "" : "kb-dmg-pop"}`}
          >
            −{damagePop}
          </span>
        )}
        <span>{hp} HP</span>
      </div>
    </div>
  );
}

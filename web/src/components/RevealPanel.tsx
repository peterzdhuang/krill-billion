"use client";

import { useEffect, useRef, useState } from "react";
import { TierBadge } from "@/components/TierBadge";
import { DepthMeter } from "@/components/DepthMeter";
import { playSfx } from "@/lib/audio";
import { useReducedMotion } from "@/lib/hit";
import type {
  RoundResultData,
  SideAnswerData,
  TierName,
} from "@/lib/protocol";

/** Reveal stages: both answers → compare → damage. */
export type RevealStage = "idle" | "a" | "b" | "hit";

const STAGE_MS = {
  a: 350,
  b: 1200,
  hit: 2300,
} as const;

function sideLabel(seat: string, names?: { a?: string; b?: string }) {
  if (seat.startsWith("A")) return names?.a?.trim() || "SIDE A";
  return names?.b?.trim() || "SIDE B";
}

/** True when the typed guess was fuzzy/prefix-corrected to a different label. */
export function wasCorrected(side: SideAnswerData): boolean {
  if (!side.valid) return false;
  const typed = side.text?.trim() ?? "";
  const canon = side.matched_label?.trim() ?? "";
  if (!typed || !canon) return false;
  return typed.toLowerCase() !== canon.toLowerCase();
}

function AnswerCard({
  side,
  label,
  show,
  winner,
  reduceMotion,
}: {
  side: SideAnswerData;
  label: string;
  show: boolean;
  winner: boolean;
  reduceMotion: boolean;
}) {
  if (!show) {
    return (
      <div className="rounded border border-dashed border-slate-800/80 bg-[#02060d]/60 px-4 py-5 text-center text-slate-600">
        …
      </div>
    );
  }
  const corrected = wasCorrected(side);
  return (
    <div
      className={`kb-reveal-card rounded border px-4 py-4 ${
        winner
          ? "border-cyan-accent/50 bg-cyan-950/30"
          : "border-slate-800 bg-[#02060d]"
      } ${reduceMotion ? "" : "kb-reveal-in"}`}
    >
      <p className="text-sm tracking-[0.25em] text-cyan-accent/45">{label}</p>
      {!side.valid ? (
        <>
          <p className="mt-2 text-2xl text-rose-300/90">
            {side.text?.trim() || "(empty)"}
          </p>
          <p className="mt-1 text-lg text-slate-500">miss · not in bank</p>
        </>
      ) : (
        <>
          {corrected ? (
            <p className="mt-2 text-3xl text-cyan-glow">
              <span className="text-slate-500 line-through decoration-slate-600">
                {side.text.trim()}
              </span>
              <span className="mx-2 text-slate-500" aria-hidden>
                →
              </span>
              <span>{side.matched_label}</span>
            </p>
          ) : (
            <p className="mt-2 text-3xl text-cyan-glow">{side.matched_label}</p>
          )}
          <div className="mt-3 flex flex-wrap items-center gap-2">
            <TierBadge tier={side.tier as TierName} />
            <span className="font-mono text-lg text-slate-400">
              rarity {side.rarity} · {side.score}
            </span>
            {side.streak > 1 && (
              <span className="text-lg text-amber-200/80">
                streak ×{side.streak}
              </span>
            )}
          </div>
        </>
      )}
    </div>
  );
}

export function RevealPanel({
  result,
  yourSeat,
  names,
  onStage,
}: {
  result: RoundResultData;
  yourSeat?: string | null;
  names?: { a?: string; b?: string };
  onStage?: (stage: RevealStage) => void;
}) {
  const reduceMotion = useReducedMotion();
  const [stage, setStage] = useState<RevealStage>(
    reduceMotion ? "hit" : "idle",
  );
  const played = useRef<Set<string>>(new Set());
  const resultKey = `${result.round}:${result.sides[0]?.text}:${result.sides[1]?.text}:${result.damage}`;

  const onStageRef = useRef(onStage);
  onStageRef.current = onStage;

  useEffect(() => {
    played.current = new Set();
    if (reduceMotion) {
      setStage("hit");
      onStageRef.current?.("hit");
      return;
    }
    setStage("idle");
    onStageRef.current?.("idle");
    const t1 = window.setTimeout(() => setStage("a"), STAGE_MS.a);
    const t2 = window.setTimeout(() => setStage("b"), STAGE_MS.b);
    const t3 = window.setTimeout(() => setStage("hit"), STAGE_MS.hit);
    return () => {
      window.clearTimeout(t1);
      window.clearTimeout(t2);
      window.clearTimeout(t3);
    };
  }, [resultKey, reduceMotion]);

  useEffect(() => {
    onStageRef.current?.(stage);
    const key = `${resultKey}:${stage}`;
    if (played.current.has(key)) return;
    played.current.add(key);
    if (stage === "a" || stage === "b") {
      playSfx("reveal");
      const side = result.sides[stage === "a" ? 0 : 1];
      if (yourSeat && side.seat === yourSeat) {
        if (side.valid && side.rarity >= 80) playSfx("flourish");
        if (!side.valid) playSfx("miss");
      }
      return;
    }
    if (stage === "hit") {
      if (result.tie || result.sudden_death) {
        playSfx(result.tie ? "flourish" : "stinger");
      } else if (result.damage > 0) {
        playSfx(result.damage >= 2500 ? "land_big" : "hit");
      }
    }
  }, [stage, resultKey, result, yourSeat]);

  const showA = stage === "a" || stage === "b" || stage === "hit";
  const showB = stage === "b" || stage === "hit";
  const showHit = stage === "hit";
  const winnerSide =
    !result.tie && (result.winner === 0 || result.winner === 1)
      ? result.winner
      : null;

  const mySide = yourSeat?.startsWith("A")
    ? 0
    : yourSeat?.startsWith("B")
      ? 1
      : null;
  const depthRarity =
    (mySide !== null ? result.sides[mySide] : undefined)?.rarity ??
    result.sides.find((s) => s.valid)?.rarity ??
    0;

  return (
    <div className="kb-phase space-y-4 rounded border border-slate-800/80 bg-[#02060d]/90 p-4">
      <p className="text-sm tracking-[0.3em] text-cyan-accent/50">
        ROUND {result.round} REVEAL
        {result.sudden_death ? " · SUDDEN DEATH" : ""}
      </p>
      {result.prompt?.text && (
        <p className="text-xl text-slate-400">{result.prompt.text}</p>
      )}

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <AnswerCard
          side={result.sides[0]}
          label={sideLabel("A1", names)}
          show={showA}
          winner={winnerSide === 0}
          reduceMotion={reduceMotion}
        />
        <AnswerCard
          side={result.sides[1]}
          label={sideLabel("B1", names)}
          show={showB}
          winner={winnerSide === 1}
          reduceMotion={reduceMotion}
        />
      </div>

      {showHit && (
        <div className={`space-y-3 ${reduceMotion ? "" : "kb-reveal-in"}`}>
          {result.sides.some((s) => s.valid && s.rarity >= 80) && (
            <p className="text-2xl tracking-widest text-amber-100">NICHE</p>
          )}
          {result.sides.some((s) => s.valid) && (
            <DepthMeter rarity={Math.max(0, depthRarity)} />
          )}
          <div className="relative overflow-hidden rounded border border-amber-500/30 bg-amber-950/20 px-4 py-3 text-center">
            {result.tie ? (
              <p className="text-2xl text-amber-100">
                TIE — both multipliers +0.5
              </p>
            ) : result.sudden_death ? (
              <p className="text-2xl text-amber-100">
                {winnerSide === null
                  ? "SUDDEN DEATH DRAW"
                  : `${sideLabel(winnerSide === 0 ? "A1" : "B1", names)} WINS`}
              </p>
            ) : (
              <>
                <p className="text-sm tracking-[0.25em] text-amber-200/60">
                  {sideLabel(winnerSide === 0 ? "A1" : "B1", names)} HITS
                </p>
                <p
                  className={`mt-1 text-4xl font-semibold text-amber-100 ${
                    reduceMotion ? "" : "kb-dmg-pop"
                  }`}
                >
                  −{result.damage}
                </p>
                <p className="mt-1 text-lg text-slate-400">
                  on {sideLabel(winnerSide === 0 ? "B1" : "A1", names)}
                </p>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

/** HP before damage lands, derived from post-damage hp + damage. */
export function preDamageHp(result: RoundResultData): [number, number] {
  const hp: [number, number] = [result.hp[0], result.hp[1]];
  if (result.tie || result.sudden_death || result.damage <= 0) return hp;
  if (result.winner === 0) hp[1] = result.hp[1] + result.damage;
  else if (result.winner === 1) hp[0] = result.hp[0] + result.damage;
  return hp;
}

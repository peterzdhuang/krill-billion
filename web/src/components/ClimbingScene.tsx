"use client";

import Image from "next/image";
import type { RoundResultData } from "@/lib/protocol";
import { PUFFER, WATER_BG } from "@/lib/krillionAssets";

/** Rounds 1–15 span the descent. (round-1)/14 is 0 on round 1 and 1 on round 15. */
const ROUND_SPAN = 14;
/** Staff and above. Matches internal/data/tier.go (Staff starts at rarity 65). */
const STAFF_RARITY = 65;
const STAFF_BOOST = 0.06;

const STAFF_OR_ABOVE = new Set([
  "Staff",
  "Principal",
  "Distinguished",
  "CTO",
  "Jeffy B",
]);

const BUBBLES: ReadonlyArray<readonly [number, number, number, number]> = [
  [12, 18, 0.9, 0],
  [28, 42, 1.2, 1.4],
  [48, 22, 0.7, 0.6],
  [64, 55, 1.1, 2.1],
  [78, 30, 0.8, 1.1],
  [18, 70, 1.0, 2.8],
  [88, 64, 0.6, 0.3],
  [36, 80, 1.3, 1.8],
];

function clamp01(n: number): number {
  if (n < 0) return 0;
  if (n > 1) return 1;
  return n;
}

function revealIsStaffOrAbove(result: RoundResultData | null): boolean {
  if (!result?.sides?.length) return false;
  let bestRarity = -1;
  for (const side of result.sides) {
    if (!side.valid) continue;
    if (STAFF_OR_ABOVE.has(side.tier)) return true;
    if (side.rarity > bestRarity) bestRarity = side.rarity;
  }
  return bestRarity >= STAFF_RARITY;
}

/**
 * Depth of the dive, 0 near the surface and 1 in the trench.
 * progress = clamp((round-1)/14, 0, 1), plus 0.06 when the current reveal
 * (or the best rarity on it) is Staff or above.
 */
export function climberProgress(
  round: number,
  result: RoundResultData | null,
): number {
  const safeRound = Number.isFinite(round) ? round : 1;
  let progress = clamp01((safeRound - 1) / ROUND_SPAN);
  if (revealIsStaffOrAbove(result)) {
    progress = clamp01(progress + STAFF_BOOST);
  }
  return progress;
}

export function ClimbingScene({
  round,
  result,
  spark = false,
  rung = false,
}: {
  round: number;
  result: RoundResultData | null;
  spark?: boolean;
  rung?: boolean;
}) {
  const progress = climberProgress(round, result);
  const depthPct = (8 + progress * 72).toFixed(2);
  const puffer = PUFFER[round % 2];

  return (
    <div
      aria-hidden
      className="pointer-events-none fixed inset-0 -z-10 overflow-x-hidden overflow-y-hidden"
    >
      <div
        className="absolute inset-0 bg-[#020814] bg-cover bg-center transition-[filter] duration-700"
        style={{
          backgroundImage: `url(${WATER_BG})`,
          filter: `brightness(${(0.95 - progress * 0.35).toFixed(2)}) saturate(${(1.05 + progress * 0.35).toFixed(2)})`,
        }}
      />
      <div
        className="absolute inset-0"
        style={{
          background: `linear-gradient(180deg,
            rgba(8, 40, 72, 0.15) 0%,
            rgba(4, 20, 48, 0.35) 40%,
            rgba(2, 8, 24, ${0.45 + progress * 0.4}) 100%)`,
        }}
      />

      {BUBBLES.map(([left, delay, scale, drift], i) => (
        <span
          key={i}
          className="kb-bubble absolute rounded-full bg-cyan-100/40"
          style={{
            left: `${left}%`,
            width: `${6 * scale}px`,
            height: `${6 * scale}px`,
            animationDelay: `${delay}s`,
            ["--kb-drift" as string]: `${drift * 8}px`,
          }}
        />
      ))}

      <div
        className="absolute left-1/2 w-24 -translate-x-1/2 transition-[top] duration-700 ease-out motion-reduce:transition-none sm:w-28"
        style={{ top: `${depthPct}%` }}
      >
        <div className={`relative ${spark ? "kb-mark-pop" : ""}`}>
          <Image
            src={puffer}
            alt=""
            width={112}
            height={112}
            className="kb-sway h-auto w-full drop-shadow-[0_8px_24px_rgba(34,211,238,0.35)]"
            priority
          />
          {spark && <div className="kb-spark absolute left-1/2 top-1/2 h-10 w-16 -translate-x-1/2 -translate-y-1/2" />}
        </div>
      </div>

      {rung && (
        <div className="kb-rung absolute inset-x-[15%] top-[42%] h-1.5 rounded-full bg-cyan-200/80" />
      )}

      <div className="absolute inset-x-0 bottom-0 h-24 bg-gradient-to-t from-[#02060d] to-transparent" />
      <div className="absolute inset-x-0 top-0 h-16 bg-gradient-to-b from-[#04101c]/80 to-transparent" />
    </div>
  );
}

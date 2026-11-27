"use client";

import type { RoundResultData } from "@/lib/protocol";

/** Rounds 1–15 span the facade. (round-1)/14 is 0 on round 1 and 1 on round 15. */
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

const WINDOW_COLS = 5;
const WINDOW_ROWS = 16;

const STARS: ReadonlyArray<readonly [number, number, number, number]> = [
  [28, 42, 1.1, 0.85],
  [64, 96, 0.8, 0.55],
  [110, 28, 1.2, 0.7],
  [148, 120, 0.7, 0.45],
  [186, 54, 1, 0.8],
  [214, 160, 0.6, 0.4],
  [250, 36, 1.3, 0.9],
  [286, 110, 0.8, 0.5],
  [40, 180, 0.7, 0.35],
  [96, 210, 0.9, 0.5],
  [170, 200, 0.6, 0.3],
  [300, 190, 1, 0.55],
  [340, 48, 0.8, 0.6],
  [360, 140, 0.7, 0.4],
  [18, 250, 0.6, 0.3],
  [380, 220, 0.9, 0.45],
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
 * Height of the climber on the facade, 0 at the lower floors and 1 at the roof.
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

function windowTone(index: number): "amber" | "cool" | "dark" {
  const band = index % 19;
  if (band === 2 || band === 11) return "amber";
  if (band === 7) return "cool";
  return "dark";
}

function Climber() {
  return (
    <svg viewBox="0 0 64 88" className="h-full w-full overflow-visible">
      <circle cx="24" cy="12" r="8" fill="#f3d0b8" />
      <path
        d="M24 20 L24 48"
        stroke="#f8fafc"
        strokeWidth="4.5"
        strokeLinecap="round"
      />
      <path
        d="M24 30 L50 22"
        stroke="#f8fafc"
        strokeWidth="3.5"
        strokeLinecap="round"
      />
      <circle cx="50" cy="22" r="2.4" fill="#f3d0b8" />
      <path
        d="M24 32 L12 42"
        stroke="#e2e8f0"
        strokeWidth="3"
        strokeLinecap="round"
      />
      <path
        d="M24 48 L14 74"
        stroke="#f8fafc"
        strokeWidth="3.5"
        strokeLinecap="round"
      />
      <path
        d="M24 48 L36 70"
        stroke="#e2e8f0"
        strokeWidth="3.5"
        strokeLinecap="round"
      />
    </svg>
  );
}

export function ClimbingScene({
  round,
  result,
}: {
  round: number;
  result: RoundResultData | null;
}) {
  const progress = climberProgress(round, result);
  const windows = WINDOW_COLS * WINDOW_ROWS;

  return (
    <div
      aria-hidden
      className="pointer-events-none fixed inset-0 -z-10 overflow-x-hidden overflow-y-hidden"
    >
      <div
        className="absolute inset-0"
        style={{
          background:
            "linear-gradient(180deg, #05060f 0%, #0a1020 46%, #141b2a 78%, #1a140e 100%)",
        }}
      />
      <svg
        className="absolute inset-0 h-full w-full"
        viewBox="0 0 400 800"
        preserveAspectRatio="xMidYMid slice"
      >
        {STARS.map(([x, y, r, opacity], i) => (
          <circle key={i} cx={x} cy={y} r={r} fill="#e2e8f0" opacity={opacity} />
        ))}
        <circle cx="332" cy="74" r="16" fill="#f4efe4" opacity="0.88" />
        <circle cx="340" cy="68" r="14" fill="#05060f" opacity="0.28" />
      </svg>

      <div className="absolute bottom-0 left-[5%] h-[40%] w-[14%] bg-[#0c121c]" />
      <div className="absolute bottom-0 right-[4%] h-[32%] w-[11%] bg-[#0b1018]" />

      <div className="absolute bottom-0 left-0 right-0 mx-auto h-[92%] w-[min(230px,56vw)] max-w-full">
        <div className="absolute left-1/2 top-0 h-[3%] w-px -translate-x-1/2 bg-slate-400/80" />
        <div className="absolute left-[40%] right-[40%] top-[3%] h-[1.4%] bg-[#2a3548]" />
        <div className="absolute left-[10%] right-[10%] top-[4.4%] h-[2%] bg-[#1a2330]" />
        <div className="absolute inset-x-0 bottom-0 top-[6.5%] bg-gradient-to-r from-[#10161f] via-[#1d293a] to-[#0c1118] shadow-[0_0_48px_rgba(0,0,0,0.45)]">
          <div
            className="grid h-full min-w-0 grid-cols-5 gap-1.5 px-3 py-3"
            style={{ gridTemplateRows: `repeat(${WINDOW_ROWS}, minmax(0, 1fr))` }}
          >
            {Array.from({ length: windows }, (_, i) => {
              const tone = windowTone(i);
              const lit =
                tone === "amber"
                  ? "bg-[#f6d48a] shadow-[0_0_8px_rgba(246,212,138,0.7)]"
                  : tone === "cool"
                    ? "bg-[#e2e8f0] shadow-[0_0_6px_rgba(226,232,240,0.45)]"
                    : "bg-[#070c14]";
              return <div key={i} className={`min-h-0 min-w-0 ${lit}`} />;
            })}
          </div>
        </div>

        <div
          className="absolute left-[5%] h-16 w-12 transition-[bottom] duration-700 ease-out motion-reduce:transition-none"
          style={{ bottom: `${(8 + progress * 70).toFixed(2)}%` }}
        >
          <Climber />
        </div>
      </div>

      <div className="absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t from-[#140e0a] to-transparent" />
    </div>
  );
}

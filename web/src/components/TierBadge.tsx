"use client";

import type { TierName } from "@/lib/protocol";
import { KrillionMark } from "@/components/KrillionMark";

const TIER_STYLES: Record<
  TierName,
  { bg: string; glow: string; text: string }
> = {
  Intern: {
    bg: "from-slate-600/40 to-slate-800/40",
    glow: "shadow-none",
    text: "text-slate-300",
  },
  Junior: {
    bg: "from-emerald-700/40 to-emerald-900/40",
    glow: "shadow-[0_0_8px_rgba(16,185,129,0.25)]",
    text: "text-emerald-300",
  },
  Senior: {
    bg: "from-cyan-700/40 to-cyan-900/50",
    glow: "shadow-[0_0_10px_rgba(34,211,238,0.3)]",
    text: "text-cyan-300",
  },
  Staff: {
    bg: "from-sky-600/40 to-indigo-900/50",
    glow: "shadow-[0_0_12px_rgba(56,189,248,0.35)]",
    text: "text-sky-300",
  },
  Principal: {
    bg: "from-violet-600/40 to-violet-950/50",
    glow: "shadow-[0_0_14px_rgba(167,139,250,0.4)]",
    text: "text-violet-300",
  },
  Distinguished: {
    bg: "from-amber-600/40 to-orange-950/50",
    glow: "shadow-[0_0_16px_rgba(251,191,36,0.45)]",
    text: "text-amber-300",
  },
  CTO: {
    bg: "from-rose-600/50 to-fuchsia-950/60",
    glow: "shadow-[0_0_18px_rgba(244,63,94,0.5)]",
    text: "text-rose-300",
  },
  "Jeffy B": {
    bg: "from-yellow-400/50 via-amber-500/40 to-orange-700/60",
    glow: "shadow-[0_0_24px_rgba(250,204,21,0.7)] animate-pulse",
    text: "text-yellow-100",
  },
};

export const ALL_TIERS: TierName[] = [
  "Intern",
  "Junior",
  "Senior",
  "Staff",
  "Principal",
  "Distinguished",
  "CTO",
  "Jeffy B",
];

export function TierBadge({
  tier,
  className = "",
}: {
  tier: TierName;
  className?: string;
}) {
  const s = TIER_STYLES[tier] ?? TIER_STYLES.Intern;
  const showAura =
    tier === "Principal" ||
    tier === "Distinguished" ||
    tier === "CTO" ||
    tier === "Jeffy B";
  return (
    <span
      className={`kb-badge-in inline-flex items-center gap-2 rounded px-2 py-1 text-base tracking-wide bg-gradient-to-r ${s.bg} ${s.glow} ${s.text} ${className}`}
    >
      <KrillionMark tier={tier} size={28} aura={showAura} />
      {tier === "Jeffy B" ? "★ JEFFY B ★" : tier}
    </span>
  );
}

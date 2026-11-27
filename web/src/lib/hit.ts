"use client";

import { useEffect, useState } from "react";
import type { RoundResultData } from "@/lib/protocol";

export function useReducedMotion(): boolean {
  const [reduce, setReduce] = useState(false);
  useEffect(() => {
    const mq = window.matchMedia("(prefers-reduced-motion: reduce)");
    const apply = () => setReduce(mq.matches);
    apply();
    mq.addEventListener("change", apply);
    return () => mq.removeEventListener("change", apply);
  }, []);
  return reduce;
}

export type HitMotion = {
  flash: boolean;
  shake: boolean;
  hard: boolean;
  gold: boolean;
  spark: boolean;
  rung: boolean;
  niche: boolean;
};

const none: HitMotion = {
  flash: false,
  shake: false,
  hard: false,
  gold: false,
  spark: false,
  rung: false,
  niche: false,
};

/** Visual hit for the side that took damage, plus a niche mark for rarity >= 80. */
export function hitMotion(
  result: RoundResultData | null,
  phase: string | undefined,
  mySide: 0 | 1 | null,
): HitMotion {
  if (phase !== "reveal" || !result || mySide === null) return none;
  const mine = result.sides[mySide];
  const niche = Boolean(mine?.valid && mine.rarity >= 80);
  const winner = result.winner;
  const tookHit =
    !result.tie && winner !== mySide && (winner === 0 || winner === 1);
  const winnerSide = winner === 0 || winner === 1 ? result.sides[winner] : undefined;
  const tier = winnerSide?.tier ?? "";
  const rung = Boolean(winnerSide && winnerSide.streak >= 3 && !result.tie);
  if (!tookHit) return { ...none, niche, rung };
  return {
    flash: tier === "Intern" || tier === "Distinguished" || tier === "CTO",
    shake:
      tier === "Junior" ||
      tier === "Senior" ||
      tier === "Staff" ||
      tier === "Principal",
    hard: tier === "Staff" || tier === "Principal",
    gold: tier === "Jeffy B",
    spark: tier === "Staff" || tier === "Principal",
    rung,
    niche,
  };
}

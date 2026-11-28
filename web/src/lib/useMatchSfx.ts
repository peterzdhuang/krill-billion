"use client";

import { useEffect, useRef } from "react";
import { playSfx } from "@/lib/audio";
import type { MatchEndData, RoundStartData } from "@/lib/protocol";

function sideOf(seat?: string | null): 0 | 1 | null {
  if (!seat) return null;
  return seat.startsWith("A") ? 0 : 1;
}

/**
 * Round-start and match-end cues. Reveal / hit SFX are sequenced by
 * RevealPanel so they line up with the GeoGuessr-style stages.
 */
export function useMatchSfx({
  round,
  phase,
  matchEnd,
  yourSeat,
}: {
  round: RoundStartData | null;
  phase?: string;
  matchEnd: MatchEndData | null;
  yourSeat?: string | null;
}) {
  const beganFor = useRef<RoundStartData | null>(null);
  const ended = useRef<MatchEndData | null>(null);

  useEffect(() => {
    if (!round || phase !== "answering" || matchEnd) return;
    if (beganFor.current === round) return;
    beganFor.current = round;
    playSfx("begin");
  }, [round, phase, matchEnd]);

  useEffect(() => {
    if (!matchEnd || ended.current === matchEnd) return;
    ended.current = matchEnd;
    const mySide = sideOf(yourSeat);
    const youWin =
      !matchEnd.draw && mySide !== null && matchEnd.winner === mySide;
    playSfx(youWin ? "win" : "miss");
  }, [matchEnd, yourSeat]);
}

"use client";

import { useEffect, useRef } from "react";
import { playSfx } from "@/lib/audio";
import type { MatchEndData, RoundResultData, RoundStartData } from "@/lib/protocol";

function sideOf(seat?: string | null): 0 | 1 | null {
  if (!seat) return null;
  return seat.startsWith("A") ? 0 : 1;
}

/** Round start, reveal, and match-end cues. Lock / reject stay at their call sites. */
export function useMatchSfx({
  round,
  phase,
  result,
  matchEnd,
  yourSeat,
}: {
  round: RoundStartData | null;
  phase?: string;
  result: RoundResultData | null;
  matchEnd: MatchEndData | null;
  yourSeat?: string | null;
}) {
  const beganFor = useRef<RoundStartData | null>(null);
  const revealed = useRef<RoundResultData | null>(null);
  const ended = useRef<MatchEndData | null>(null);

  useEffect(() => {
    if (!round || phase !== "answering" || matchEnd) return;
    if (beganFor.current === round) return;
    beganFor.current = round;
    playSfx("begin");
  }, [round, phase, matchEnd]);

  useEffect(() => {
    if (!result || revealed.current === result) return;
    revealed.current = result;
    const mine = yourSeat ? result.sides.find((a) => a.seat === yourSeat) : undefined;
    if (mine && !mine.valid) {
      playSfx("miss");
      return;
    }
    playSfx(result.damage < 1500 ? "land_small" : "land_big");
  }, [result, yourSeat]);

  useEffect(() => {
    if (!matchEnd || ended.current === matchEnd) return;
    ended.current = matchEnd;
    const mySide = sideOf(yourSeat);
    const youWin = !matchEnd.draw && mySide !== null && matchEnd.winner === mySide;
    playSfx(youWin ? "krillion" : "miss");
  }, [matchEnd, yourSeat]);
}

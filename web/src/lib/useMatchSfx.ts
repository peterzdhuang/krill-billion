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
    const mySide = sideOf(yourSeat);
    const mine = yourSeat ? result.sides.find((a) => a.seat === yourSeat) : undefined;
    if (mine?.valid && mine.rarity >= 80) playSfx("flourish");
    if (mine && !mine.valid) playSfx("miss");
    if (
      mySide === null ||
      result.tie ||
      result.winner === mySide ||
      result.winner < 0
    ) {
      return;
    }
    const winner = result.sides[result.winner];
    if (winner?.tier === "Jeffy B") playSfx("krillion");
    else if (winner?.tier === "Distinguished" || winner?.tier === "CTO") {
      playSfx("stinger");
    }
  }, [result, yourSeat]);

  useEffect(() => {
    if (!matchEnd || ended.current === matchEnd) return;
    ended.current = matchEnd;
    const mySide = sideOf(yourSeat);
    const youWin = !matchEnd.draw && mySide !== null && matchEnd.winner === mySide;
    playSfx(youWin ? "krillion" : "miss");
  }, [matchEnd, yourSeat]);
}

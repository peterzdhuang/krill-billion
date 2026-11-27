"use client";

import { useEffect, useRef, useState } from "react";
import { playSfx } from "@/lib/audio";

/** Ring full-scale. Matches match.AnsweringDur; the server still owns the deadline. */
const RING_SCALE_S = 25;
const TICK_MS = 200;
const WARN_AT_S = 10;
const CRITICAL_AT_S = 5;
const DOUBLE_TICK_GAP_MS = 140;

function useRemaining(deadlineTs?: number): number | null {
  const [now, setNow] = useState<number | null>(null);
  const [tracked, setTracked] = useState(deadlineTs);
  if (deadlineTs !== tracked) {
    setTracked(deadlineTs);
    if (now !== null) setNow(Date.now());
  }

  useEffect(() => {
    if (deadlineTs === undefined) return;
    setNow(Date.now());
    const id = window.setInterval(() => setNow(Date.now()), TICK_MS);
    return () => window.clearInterval(id);
  }, [deadlineTs]);

  if (deadlineTs === undefined || now === null) return null;
  return Math.max(0, (deadlineTs - now) / 1000);
}

/**
 * Answering-phase countdown. Whole seconds use ceil, same as the old caption.
 * Ticks on seconds 10–1 only, and only while phase is answering.
 * playSfx("tick") returns immediately when muted.
 */
export function RoundTimer({
  deadlineTs,
  phase,
}: {
  deadlineTs?: number;
  phase?: string;
}) {
  const remaining = useRemaining(deadlineTs);
  const answering = phase === "answering";
  const lastTickedWhole = useRef<number | null>(null);
  const followUp = useRef<number | null>(null);
  const followUpDone = useRef(false);
  const phaseRef = useRef(phase);
  const shownWhole = useRef(0);
  const clockGen = useRef(0);
  phaseRef.current = phase;
  if (remaining !== null) shownWhole.current = Math.ceil(remaining);

  const seenDeadline = useRef(deadlineTs);
  if (seenDeadline.current !== deadlineTs) {
    seenDeadline.current = deadlineTs;
    clockGen.current += 1;
    lastTickedWhole.current = null;
    followUpDone.current = false;
  }

  useEffect(() => {
    const cancelFollowUp = () => {
      if (followUp.current !== null) {
        window.clearTimeout(followUp.current);
        followUp.current = null;
      }
    };

    if (!answering || remaining === null) {
      cancelFollowUp();
      return;
    }

    const whole = Math.ceil(remaining);
    if (whole < 1 || whole > WARN_AT_S) {
      cancelFollowUp();
      return;
    }

    const gen = clockGen.current;
    const scheduleHarsh = () => {
      cancelFollowUp();
      const id = window.setTimeout(() => {
        if (followUp.current === id) followUp.current = null;
        if (clockGen.current !== gen) return;
        followUpDone.current = true;
        const at = shownWhole.current;
        if (phaseRef.current !== "answering" || at < 1 || at > CRITICAL_AT_S) return;
        playSfx("tick");
      }, DOUBLE_TICK_GAP_MS);
      followUp.current = id;
    };

    if (lastTickedWhole.current !== whole) {
      lastTickedWhole.current = whole;
      followUpDone.current = whole > CRITICAL_AT_S;
      playSfx("tick");
      if (whole <= CRITICAL_AT_S) scheduleHarsh();
    } else if (
      whole <= CRITICAL_AT_S &&
      followUp.current === null &&
      !followUpDone.current
    ) {
      // A strict-mode remount clears the harsh hit. Arm it again for this
      // same whole second without replaying the first tick.
      scheduleHarsh();
    }

    return () => {
      const sameSecond =
        phaseRef.current === "answering" &&
        shownWhole.current === whole &&
        clockGen.current === gen;
      if (!sameSecond) cancelFollowUp();
    };
  }, [answering, remaining]);

  useEffect(() => {
    const pending = followUp;
    return () => {
      if (pending.current !== null) {
        window.clearTimeout(pending.current);
        pending.current = null;
      }
    };
  }, []);

  if (!answering || remaining === null) return null;

  const whole = Math.ceil(remaining);
  const critical = whole <= CRITICAL_AT_S;
  const warn = whole <= WARN_AT_S;
  const tone = critical
    ? "text-rose-300"
    : warn
      ? "text-amber-300"
      : "text-cyan-glow";
  const radius = 52;
  const circumference = 2 * Math.PI * radius;
  const fraction = Math.max(0, Math.min(1, remaining / RING_SCALE_S));
  const offset = circumference * (1 - fraction);

  return (
    <div
      role="timer"
      aria-label={`${whole} ${whole === 1 ? "second" : "seconds"} remaining`}
      className={`relative mx-auto mt-3 h-32 w-32 sm:h-48 sm:w-48 ${tone} ${
        critical ? "motion-safe:animate-pulse" : ""
      }`}
    >
      <svg viewBox="0 0 120 120" className="h-full w-full -rotate-90" aria-hidden>
        <circle
          cx="60"
          cy="60"
          r={radius}
          fill="none"
          strokeWidth="8"
          className="stroke-slate-800"
        />
        {fraction > 0 && (
          <circle
            cx="60"
            cy="60"
            r={radius}
            fill="none"
            strokeWidth="8"
            strokeLinecap="round"
            className="stroke-current transition-[stroke-dashoffset] duration-200 ease-linear motion-reduce:transition-none"
            style={{
              strokeDasharray: `${circumference} ${circumference}`,
              strokeDashoffset: offset,
            }}
          />
        )}
      </svg>
      <div className="absolute inset-0 flex items-center justify-center">
        <span className="font-mono text-6xl leading-none sm:text-8xl">{whole}</span>
      </div>
    </div>
  );
}

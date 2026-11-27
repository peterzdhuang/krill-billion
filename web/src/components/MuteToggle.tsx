"use client";

import { useSyncExternalStore } from "react";
import { isMuted, subscribeMute, toggleMuted } from "@/lib/audio";

function subscribe(onChange: () => void) {
  return subscribeMute(() => onChange());
}

function mutedOnServer() {
  return false;
}

export function MuteToggle({ className = "" }: { className?: string }) {
  const muted = useSyncExternalStore(subscribe, isMuted, mutedOnServer);
  return (
    <button
      type="button"
      aria-pressed={muted}
      aria-label={muted ? "turn sound on" : "turn sound off"}
      onClick={() => {
        toggleMuted();
      }}
      className={`inline-flex min-h-11 items-center rounded border border-cyan-accent/40 px-3 text-xl text-cyan-glow hover:border-cyan-glow/70 ${className}`}
    >
      {muted ? "Sound off" : "Sound on"}
    </button>
  );
}

"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { ALL_TIERS, TierBadge } from "@/components/TierBadge";
import { DepthMeter } from "@/components/DepthMeter";
import type { Mode } from "@/lib/protocol";

const NAME_KEY = "kb_display_name";

export default function Home() {
  const router = useRouter();
  const [name, setName] = useState(() => {
    if (typeof window === "undefined") return "";
    return localStorage.getItem(NAME_KEY) ?? "";
  });
  const [mode, setMode] = useState<Mode>("1v1");
  const [error, setError] = useState<string | null>(null);
  const [pending, startTransition] = useTransition();

  const createLobby = () => {
    const display = name.trim();
    if (!display) {
      setError("Enter a display name");
      return;
    }
    localStorage.setItem(NAME_KEY, display);
    setError(null);
    startTransition(async () => {
      try {
        const res = await fetch("/api/lobbies", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ mode, display_name: display }),
        });
        if (!res.ok) {
          const t = await res.text();
          setError(t || `Create failed (${res.status})`);
          return;
        }
        const data = (await res.json()) as {
          lobby_id: string;
          link: string;
          host_token: string;
        };
        localStorage.setItem(`kb_host_${data.lobby_id}`, data.host_token);
        router.push(`/play/${data.lobby_id}`);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Network error");
      }
    });
  };

  return (
    <main className="relative flex min-h-screen flex-col items-center justify-center overflow-hidden px-6 py-16">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0"
        style={{
          backgroundImage:
            "radial-gradient(ellipse at 50% 0%, rgba(14,116,144,0.25), transparent 50%), radial-gradient(circle at 15% 80%, rgba(6,78,59,0.2), transparent 40%), radial-gradient(circle at 85% 60%, rgba(8,47,73,0.35), transparent 45%)",
        }}
      />
      <div
        aria-hidden
        className="pointer-events-none absolute inset-x-0 bottom-0 h-1/3 opacity-30"
        style={{
          background:
            "repeating-linear-gradient(0deg, transparent, transparent 12px, rgba(45,212,191,0.04) 12px, rgba(45,212,191,0.04) 13px)",
        }}
      />

      <div className="relative z-10 w-full max-w-xl text-center">
        <p className="mb-3 text-sm tracking-[0.4em] text-cyan-accent/60">
          DEPTH PROTOCOL // COMMIT-BASED DUELS
        </p>
        <h1 className="text-6xl leading-none tracking-wide text-cyan-glow sm:text-8xl">
          KRILL BILLION
        </h1>
        <p className="mt-5 text-xl text-slate-300/90 sm:text-2xl">
          Guess rarer. Sink deeper. Drain their HP.
        </p>

        <div className="mx-auto mt-12 w-full max-w-md space-y-5 text-left">
          <label className="block">
            <span className="mb-1 block text-sm tracking-widest text-cyan-accent/50">
              CALLSIGN
            </span>
            <div className="flex items-center gap-2 rounded border border-cyan-900/60 bg-[#02060d]/80 px-3 py-2 text-2xl text-cyan-glow focus-within:border-cyan-accent/50">
              <span className="text-cyan-accent/70">$</span>
              <input
                value={name}
                onChange={(e) => setName(e.target.value)}
                maxLength={24}
                placeholder="display_name"
                className="w-full bg-transparent outline-none caret-cyan-glow placeholder:text-slate-600"
                onKeyDown={(e) => e.key === "Enter" && createLobby()}
              />
              <span className="inline-block h-5 w-2 animate-pulse bg-cyan-glow/80" />
            </div>
          </label>

          <div>
            <span className="mb-1 block text-sm tracking-widest text-cyan-accent/50">
              MODE
            </span>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setMode("1v1")}
                className={`flex-1 rounded border px-3 py-2 text-xl transition ${
                  mode === "1v1"
                    ? "border-cyan-accent/60 bg-cyan-950/40 text-cyan-glow"
                    : "border-slate-700/60 text-slate-500"
                }`}
              >
                1v1
              </button>
              <button
                type="button"
                disabled
                title="2v2 unlocks in a later milestone"
                className="flex-1 cursor-not-allowed rounded border border-slate-800/80 px-3 py-2 text-xl text-slate-600"
              >
                2v2 (soon)
              </button>
            </div>
          </div>

          <button
            type="button"
            disabled={pending}
            onClick={createLobby}
            className="w-full rounded border border-cyan-accent/40 bg-gradient-to-b from-cyan-900/50 to-slate-950/80 py-3 text-2xl tracking-wide text-cyan-glow transition hover:border-cyan-glow/70 hover:shadow-[0_0_20px_rgba(45,212,191,0.2)] disabled:opacity-50"
          >
            {pending ? "SPINNING UP LOBBY…" : "CREATE LOBBY"}
          </button>
          {error && <p className="text-center text-lg text-rose-400">{error}</p>}
        </div>

        <div className="mx-auto mt-14 max-w-md space-y-4 opacity-90">
          <DepthMeter rarity={72} />
          <div className="flex flex-wrap justify-center gap-2">
            {ALL_TIERS.map((t) => (
              <TierBadge key={t} tier={t} />
            ))}
          </div>
        </div>
      </div>
    </main>
  );
}

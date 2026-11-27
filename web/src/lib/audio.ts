/**
 * Synthesized cues ported from Krillion (https://krillion.io).
 * OscillatorNode + noise buffer + gain envelopes. No sample files.
 */

const MUTE_KEY = "kb-muted";
const MASTER = 0.75;

export type SfxName =
  | "submit"
  | "reject"
  | "tick"
  | "land_small"
  | "land_big"
  | "begin"
  | "krillion"
  | "stinger"
  | "flourish"
  | "miss";

type MuteListener = (muted: boolean) => void;

type ToneOpts = {
  type?: OscillatorType;
  f0: number;
  f1?: number;
  dur?: number;
  a?: number;
  peak?: number;
  lp?: number;
  delay?: number;
};

type NoiseOpts = {
  dur?: number;
  lp?: number;
  hp?: number;
  peak?: number;
  a?: number;
  delay?: number;
  sweepTo?: number;
};

let ctx: AudioContext | null = null;
let master: GainNode | null = null;
let muted = false;
let booted = false;
const listeners = new Set<MuteListener>();

function audioCtor(): typeof AudioContext | null {
  if (typeof window === "undefined") return null;
  const w = window as Window & { webkitAudioContext?: typeof AudioContext };
  return window.AudioContext ?? w.webkitAudioContext ?? null;
}

function boot() {
  if (booted || typeof window === "undefined") return;
  booted = true;
  try {
    muted = localStorage.getItem(MUTE_KEY) === "1";
  } catch {
    muted = false;
  }
  const unlock = () => {
    ensureContext();
  };
  document.addEventListener("pointerdown", unlock, { capture: true, passive: true });
  document.addEventListener("keydown", unlock, { capture: true });
}

function applyMuteGain() {
  if (!master) return;
  master.gain.value = muted ? 0 : MASTER;
}

function ensureContext(): AudioContext | null {
  const Ctor = audioCtor();
  if (!Ctor) return null;
  if (ctx && ctx.state === "closed") {
    ctx = null;
    master = null;
  }
  if (!ctx) {
    try {
      ctx = new Ctor();
      master = ctx.createGain();
      applyMuteGain();
      master.connect(ctx.destination);
    } catch {
      ctx = null;
      master = null;
      return null;
    }
  }
  if (ctx.state === "suspended") {
    ctx.resume().catch(() => {});
  }
  return ctx.state === "closed" ? null : ctx;
}

function env(gain: GainNode, t: number, attack: number, peak: number, dur: number) {
  const peakAt = Math.max(peak, 1e-4);
  gain.gain.setValueAtTime(0, t);
  gain.gain.linearRampToValueAtTime(peakAt, t + attack);
  gain.gain.exponentialRampToValueAtTime(1e-4, t + attack + dur);
}

function tone(ac: AudioContext, out: GainNode, opts: ToneOpts) {
  const dur = opts.dur ?? 0.2;
  const attack = opts.a ?? 0.005;
  const peak = opts.peak ?? 0.3;
  const delay = opts.delay ?? 0;
  const t = ac.currentTime + delay;
  const osc = ac.createOscillator();
  osc.type = opts.type ?? "sine";
  osc.frequency.setValueAtTime(opts.f0, t);
  if (opts.f1 !== undefined) {
    osc.frequency.exponentialRampToValueAtTime(Math.max(opts.f1, 1), t + dur);
  }
  const gain = ac.createGain();
  env(gain, t, attack, peak, dur);
  let node: AudioNode = osc;
  if (opts.lp) {
    const filter = ac.createBiquadFilter();
    filter.type = "lowpass";
    filter.frequency.value = opts.lp;
    osc.connect(filter);
    node = filter;
  }
  node.connect(gain);
  gain.connect(out);
  osc.start(t);
  osc.stop(t + dur + attack + 0.05);
}

function noise(ac: AudioContext, out: GainNode, opts: NoiseOpts) {
  const dur = opts.dur ?? 0.3;
  const lp = opts.lp ?? 1200;
  const peak = opts.peak ?? 0.25;
  const attack = opts.a ?? 0.003;
  const delay = opts.delay ?? 0;
  const t = ac.currentTime + delay;
  const length = Math.ceil(ac.sampleRate * (dur + 0.1));
  const buffer = ac.createBuffer(1, length, ac.sampleRate);
  const data = buffer.getChannelData(0);
  for (let i = 0; i < length; i++) data[i] = Math.random() * 2 - 1;
  const source = ac.createBufferSource();
  source.buffer = buffer;
  const lowpass = ac.createBiquadFilter();
  lowpass.type = "lowpass";
  lowpass.frequency.setValueAtTime(lp, t);
  if (opts.sweepTo) {
    lowpass.frequency.exponentialRampToValueAtTime(opts.sweepTo, t + dur);
  }
  source.connect(lowpass);
  let node: AudioNode = lowpass;
  if (opts.hp) {
    const highpass = ac.createBiquadFilter();
    highpass.type = "highpass";
    highpass.frequency.value = opts.hp;
    lowpass.connect(highpass);
    node = highpass;
  }
  const gain = ac.createGain();
  env(gain, t, attack, peak, dur);
  node.connect(gain);
  gain.connect(out);
  source.start(t);
  source.stop(t + dur + 0.1);
}

function arp(
  ac: AudioContext,
  out: GainNode,
  notes: number[],
  opts: { type?: OscillatorType; step?: number; dur?: number; peak?: number; lp?: number },
) {
  const step = opts.step ?? 0.12;
  notes.forEach((f0, i) => {
    tone(ac, out, {
      type: opts.type ?? "triangle",
      f0,
      dur: opts.dur ?? 0.5,
      peak: opts.peak ?? 0.16,
      lp: opts.lp,
      delay: i * step,
    });
  });
}

function emit(ac: AudioContext, out: GainNode, name: SfxName) {
  switch (name) {
    case "submit":
      // Soft bubble lock — same silhouette as Krillion's submit ping.
      tone(ac, out, { f0: 1180, dur: 0.12, a: 0.002, peak: 0.14 });
      tone(ac, out, { type: "sine", f0: 1560, dur: 0.1, peak: 0.08, delay: 0.04 });
      noise(ac, out, { dur: 0.08, lp: 2200, hp: 900, peak: 0.04 });
      break;
    case "reject":
      noise(ac, out, { dur: 0.22, lp: 420, peak: 0.2 });
      tone(ac, out, { f0: 150, f1: 62, dur: 0.36, peak: 0.22, lp: 700 });
      break;
    case "tick":
      tone(ac, out, { f0: 920, dur: 0.055, peak: 0.14 });
      tone(ac, out, { f0: 920, dur: 0.04, peak: 0.05, delay: 0.11 });
      break;
    case "land_small":
      tone(ac, out, { f0: 140, f1: 70, dur: 0.25, peak: 0.28, lp: 600 });
      noise(ac, out, { dur: 0.16, lp: 700, hp: 200, peak: 0.09 });
      break;
    case "land_big":
      tone(ac, out, { f0: 72, f1: 38, dur: 0.7, peak: 0.4 });
      noise(ac, out, { dur: 0.85, lp: 1100, sweepTo: 180, peak: 0.2, delay: 0.02 });
      tone(ac, out, { type: "triangle", f0: 523, dur: 0.45, peak: 0.1, delay: 0.16 });
      noise(ac, out, { dur: 0.3, lp: 2800, hp: 1400, peak: 0.05, delay: 0.25 });
      break;
    case "begin":
      noise(ac, out, { dur: 1.25, lp: 2400, sweepTo: 220, peak: 0.16 });
      tone(ac, out, { f0: 98, f1: 42, dur: 1.3, peak: 0.34 });
      tone(ac, out, { type: "triangle", f0: 784, dur: 0.55, peak: 0.12, lp: 3000 });
      tone(ac, out, { type: "triangle", f0: 523, dur: 0.8, peak: 0.12, lp: 3000, delay: 0.24 });
      break;
    case "krillion":
      tone(ac, out, { f0: 55, f1: 36, dur: 1.35, peak: 0.36 });
      arp(ac, out, [784, 988, 1175, 1568, 1976], { step: 0.12, dur: 0.75, peak: 0.15, lp: 3200 });
      noise(ac, out, { dur: 1.5, lp: 2600, hp: 1100, peak: 0.06, delay: 0.28 });
      break;
    case "stinger":
      tone(ac, out, { f0: 90, f1: 50, dur: 0.7, peak: 0.3 });
      arp(ac, out, [659, 880, 1175], { step: 0.1, dur: 0.4, peak: 0.14, lp: 2800 });
      break;
    case "flourish":
      arp(ac, out, [1046, 1318, 1568], { step: 0.07, dur: 0.28, peak: 0.12, lp: 3200 });
      break;
    case "miss":
      tone(ac, out, { f0: 220, f1: 52, dur: 0.7, peak: 0.24 });
      noise(ac, out, { dur: 0.5, lp: 400, peak: 0.14, delay: 0.05 });
      break;
    default:
      break;
  }
}

export function isMuted(): boolean {
  boot();
  return muted;
}

export function toggleMuted(): boolean {
  boot();
  muted = !muted;
  try {
    localStorage.setItem(MUTE_KEY, muted ? "1" : "0");
  } catch {
    /* storage unavailable; the in-memory flag still applies */
  }
  applyMuteGain();
  for (const cb of listeners) cb(muted);
  return muted;
}

export function subscribeMute(cb: MuteListener): () => void {
  boot();
  listeners.add(cb);
  return () => {
    listeners.delete(cb);
  };
}

export function playSfx(name: SfxName): void {
  boot();
  if (muted) return;
  const ac = ensureContext();
  if (!ac || !master) return;
  const out = master;
  const run = () => {
    if (muted || ac.state !== "running") return;
    try {
      emit(ac, out, name);
    } catch {
      /* context could not schedule this cue */
    }
  };
  if (ac.state === "running") {
    run();
    return;
  }
  ac.resume().then(run).catch(() => {});
}

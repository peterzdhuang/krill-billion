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
      tone(ac, out, { f0: 1150, dur: 0.16, a: 0.003, peak: 0.16 });
      break;
    case "reject":
      noise(ac, out, { dur: 0.22, lp: 420, peak: 0.2 });
      tone(ac, out, { f0: 150, f1: 62, dur: 0.36, peak: 0.22, lp: 700 });
      break;
    case "tick":
      tone(ac, out, { f0: 880, dur: 0.07, peak: 0.16 });
      tone(ac, out, { f0: 880, dur: 0.05, peak: 0.06, delay: 0.12 });
      break;
    case "land_small":
      tone(ac, out, { f0: 140, f1: 70, dur: 0.25, peak: 0.28, lp: 600 });
      noise(ac, out, { dur: 0.14, lp: 500, peak: 0.1 });
      break;
    case "land_big":
      tone(ac, out, { f0: 80, f1: 45, dur: 0.6, peak: 0.38 });
      noise(ac, out, { dur: 0.7, lp: 900, sweepTo: 200, peak: 0.18, delay: 0.02 });
      tone(ac, out, { type: "triangle", f0: 523, dur: 0.4, peak: 0.09, delay: 0.15 });
      break;
    case "begin":
      noise(ac, out, { dur: 1.1, lp: 2000, sweepTo: 240, peak: 0.15 });
      tone(ac, out, { f0: 110, f1: 48, dur: 1.2, peak: 0.32 });
      tone(ac, out, { type: "triangle", f0: 784, dur: 0.5, peak: 0.12, lp: 3000 });
      tone(ac, out, { type: "triangle", f0: 523, dur: 0.75, peak: 0.12, lp: 3000, delay: 0.22 });
      break;
    case "krillion":
      tone(ac, out, { f0: 60, f1: 40, dur: 1.2, peak: 0.34 });
      arp(ac, out, [784, 988, 1175, 1568], { step: 0.13, dur: 0.7, peak: 0.15, lp: 3000 });
      noise(ac, out, { dur: 1.4, lp: 2500, hp: 1200, peak: 0.05, delay: 0.3 });
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

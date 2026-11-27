/** Mirrored from internal/ws/protocol.go — keep in sync. */

export type Mode = "1v1" | "2v2";
export type Seat = "A1" | "A2" | "B1" | "B2";
export type Phase = "lobby" | "answering" | "reveal" | "paused" | "ended";
export type MatchEndReason = "ko" | "forfeit" | "cap" | "draw";
export type ErrorCode =
  | "seat_taken"
  | "not_host"
  | "not_ready"
  | "bad_phase"
  | "duplicate_answer"
  | "rate_limited"
  | "not_found"
  | "bad_request";

export type TierName =
  | "Intern"
  | "Junior"
  | "Senior"
  | "Staff"
  | "Principal"
  | "Distinguished"
  | "CTO"
  | "Jeffy B";

export interface Envelope<T = unknown> {
  type: string;
  data: T;
}

// —— Client → Server ——

export interface JoinMsg {
  lobby_id: string;
  display_name: string;
  seat?: Seat;
  reconnect_token?: string;
}

export interface ClaimSeatMsg {
  seat: Seat;
}

export interface DraftMsg {
  text: string;
}

export type ClientMessage =
  | { type: "join"; data: JoinMsg }
  | { type: "claim_seat"; data: ClaimSeatMsg }
  | { type: "ready"; data: Record<string, never> }
  | { type: "start"; data: Record<string, never> }
  | { type: "draft"; data: DraftMsg }
  | { type: "lock"; data: Record<string, never> }
  | { type: "ping"; data: Record<string, never> };

// —— Server → Client ——

export interface SeatInfo {
  seat: Seat;
  player_id?: string;
  display_name?: string;
  ready: boolean;
  empty: boolean;
}

export interface LobbyStateMsg {
  lobby_id: string;
  seats: SeatInfo[];
  mode: Mode;
  link: string;
  host_player_id: string;
  you_are_host?: boolean;
  your_seat?: Seat | null;
  your_player_id?: string;
}

export interface PlayerPublic {
  seat: Seat;
  display_name: string;
  player_id: string;
  connected: boolean;
}

export interface RoomStateMsg {
  players: PlayerPublic[];
  hp: { A: number; B: number };
  multipliers: { A: number; B: number };
  phase: Phase;
  clock_ms?: number;
  round: number;
  match_deadline_ts?: number;
  pause_until_ts?: number;
  paused_for_seat?: Seat;
}

export interface RoundStartMsg {
  prompt: { id: string; text: string; category: string };
  deadline_ts: number;
  round: number;
  sudden_death?: boolean;
}

export interface TeammateDraftMsg {
  seat: Seat;
  text: string;
}

export interface LockedMsg {
  seat: Seat;
}

export interface AnswerReveal {
  seat: Seat;
  text: string;
  matched: boolean;
  label?: string;
  rarity?: number;
  tier?: TierName;
  score: number;
  entry_id?: string;
}

export interface RoundResultMsg {
  round: number;
  answers: AnswerReveal[];
  gap: number;
  damage: number;
  winner_side?: "A" | "B" | null;
  tie: boolean;
  hp: { A: number; B: number };
  multipliers: { A: number; B: number };
}

export interface RoundRecap {
  round: number;
  answers: AnswerReveal[];
  damage: number;
  winner_side?: "A" | "B" | null;
}

export interface MatchEndMsg {
  winner_side?: "A" | "B" | null;
  draw: boolean;
  reason: MatchEndReason;
  recap: RoundRecap[];
  hp: { A: number; B: number };
}

export interface ErrorMsg {
  code: ErrorCode | string;
  message: string;
}

export interface JoinAckMsg {
  player_id: string;
  reconnect_token: string;
  seat?: Seat | null;
  is_host: boolean;
}

/** Mirrored from internal/ws/protocol.go — keep in sync. */

export type ClientMessageType =
  | "join"
  | "claim_seat"
  | "ready"
  | "start"
  | "set_mode"
  | "draft"
  | "lock"
  | "ping";

export type ServerMessageType =
  | "lobby_state"
  | "room_state"
  | "round_start"
  | "teammate_draft"
  | "locked"
  | "round_result"
  | "match_end"
  | "error"
  | "pong";

export type Envelope<T = unknown> = {
  type: ClientMessageType | ServerMessageType;
  data: T;
};

export type JoinData = {
  lobby_id: string;
  display_name: string;
  seat?: string;
  reconnect_token?: string;
  host_token?: string;
};

export type ClaimSeatData = { seat: string };
export type ReadyData = Record<string, never>;
export type StartData = Record<string, never>;
export type SetModeData = { mode: "1v1" | "2v2" };
export type DraftData = { text: string };
export type LockData = Record<string, never>;
export type PingData = Record<string, never>;

export type SeatState = {
  seat: string;
  player_id?: string;
  display_name?: string;
  ready: boolean;
  connected: boolean;
};

export type LobbyStateData = {
  seats: SeatState[];
  mode: string;
  link: string;
  host: string;
  host_player_id?: string;
  started: boolean;
  player_id?: string;
  reconnect_token?: string;
  your_seat?: string;
};

export type RoomPlayer = {
  player_id: string;
  display_name: string;
  seat: string;
  connected: boolean;
};

export type RoomStateData = {
  players: RoomPlayer[];
  hp: [number, number];
  multipliers: [number, number];
  phase: string;
  clock: number;
  round: number;
  sudden_death: boolean;
};

export type PromptInfo = {
  id: string;
  text: string;
  category: string;
};

export type RoundStartData = {
  prompt: PromptInfo;
  deadline_ts: number;
  round: number;
  sudden_death: boolean;
};

export type TeammateDraftData = { seat: string; text: string };
export type LockedData = { seat: string };

export type SideAnswerData = {
  seat: string;
  text: string;
  matched_label: string;
  rarity: number;
  tier: string;
  score: number;
  valid: boolean;
  streak: number;
};

export type RoundResultData = {
  round: number;
  sudden_death: boolean;
  sides: [SideAnswerData, SideAnswerData];
  damage: number;
  gap: number;
  hp: [number, number];
  multipliers: [number, number];
  winner: number;
  tie: boolean;
};

export type MatchEndData = {
  winner: 0 | 1 | null;
  draw: boolean;
  reason: "ko" | "forfeit" | "cap" | "draw";
  recap: RoundResultData[];
};

export type ErrorData = {
  code:
    | "seat_taken"
    | "not_host"
    | "not_ready"
    | "bad_phase"
    | "duplicate_answer"
    | "rate_limited"
    | string;
  message: string;
};

export const ErrorCodes = {
  seat_taken: "seat_taken",
  not_host: "not_host",
  not_ready: "not_ready",
  bad_phase: "bad_phase",
  duplicate_answer: "duplicate_answer",
  rate_limited: "rate_limited",
} as const;

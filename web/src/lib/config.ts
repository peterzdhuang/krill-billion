/** Frontend → server URL helpers. */

export function serverHTTPBase(): string {
  const raw = process.env.NEXT_PUBLIC_SERVER_URL?.trim();
  if (!raw) return "";
  return raw.replace(/\/$/, "");
}

export function apiURL(path: string): string {
  const base = serverHTTPBase();
  if (!base) return path; // same-origin via Next rewrite
  return `${base}${path.startsWith("/") ? path : `/${path}`}`;
}

export function wsURL(lobbyId: string): string {
  const base = serverHTTPBase();
  if (base) {
    const u = new URL(base);
    u.protocol = u.protocol === "https:" ? "wss:" : "ws:";
    u.pathname = "/ws";
    u.search = `?lobby=${encodeURIComponent(lobbyId)}`;
    return u.toString();
  }
  const proto = typeof window !== "undefined" && window.location.protocol === "https:" ? "wss" : "ws";
  const host = typeof window !== "undefined" ? window.location.host : "localhost:3000";
  return `${proto}://${host}/ws?lobby=${encodeURIComponent(lobbyId)}`;
}

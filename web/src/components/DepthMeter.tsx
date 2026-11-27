"use client";

/** Maps rarity 0–100 onto tower floors (higher rarity = a higher floor). */
export function DepthMeter({
  rarity,
  className = "",
}: {
  rarity: number;
  className?: string;
}) {
  const clamped = Math.max(0, Math.min(100, rarity));
  const floor = Math.round(clamped);
  const floorLabel = floor >= 100 ? "ROOF" : floor <= 0 ? "LOBBY" : `FL ${floor}`;

  return (
    <div className={`flex flex-col gap-1 ${className}`}>
      <div className="flex justify-between text-sm text-cyan-accent/70">
        <span>LOBBY</span>
        <span>{floorLabel}</span>
        <span>ROOF</span>
      </div>
      <div className="relative h-3 w-full overflow-hidden rounded-sm bg-[#121820] ring-1 ring-slate-700/80">
        <div
          className="absolute inset-y-0 left-0 transition-all duration-700 ease-out motion-reduce:transition-none"
          style={{
            width: `${clamped}%`,
            background:
              "linear-gradient(90deg, #1e293b 0%, #334155 38%, #b45309 72%, #fde68a 100%)",
          }}
        />
        {Array.from({ length: 14 }, (_, i) => (
          <div
            key={i}
            className="absolute inset-y-0 w-px bg-black/45"
            style={{ left: `${((i + 1) / 15) * 100}%` }}
          />
        ))}
        <div
          className="absolute top-1/2 h-2 w-2 -translate-y-1/2 rounded-full bg-amber-100 shadow-[0_0_8px_#fde68a] transition-all duration-700 motion-reduce:transition-none"
          style={{ left: `calc(${clamped}% - 4px)` }}
        />
      </div>
    </div>
  );
}

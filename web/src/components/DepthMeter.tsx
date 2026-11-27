"use client";

/** Maps rarity 0–100 to ocean depth visualization (rarer = deeper). */
export function DepthMeter({
  rarity,
  className = "",
}: {
  rarity: number;
  className?: string;
}) {
  const clamped = Math.max(0, Math.min(100, rarity));
  const depthM = Math.round(clamped * 110); // 0–11000m feel
  const fill = clamped;

  return (
    <div className={`flex flex-col gap-1 ${className}`}>
      <div className="flex justify-between text-sm text-cyan-accent/70">
        <span>SURFACE</span>
        <span>{depthM}m</span>
        <span>ABYSS</span>
      </div>
      <div className="relative h-3 w-full overflow-hidden rounded-sm bg-[#02060d] ring-1 ring-cyan-900/50">
        <div
          className="absolute inset-y-0 left-0 transition-all duration-700 ease-out"
          style={{
            width: `${fill}%`,
            background:
              "linear-gradient(90deg, #0e7490 0%, #155e75 40%, #0c4a6e 70%, #042f2e 100%)",
          }}
        />
        <div
          className="absolute top-1/2 h-2 w-2 -translate-y-1/2 rounded-full bg-cyan-glow shadow-[0_0_8px_#5eead4] transition-all duration-700"
          style={{ left: `calc(${fill}% - 4px)` }}
        />
      </div>
    </div>
  );
}

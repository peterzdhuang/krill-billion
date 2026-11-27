"use client";

/** Maps rarity 0–100 onto dive depth (higher rarity = deeper). */
export function DepthMeter({
  rarity,
  className = "",
}: {
  rarity: number;
  className?: string;
}) {
  const clamped = Math.max(0, Math.min(100, rarity));
  const meters = clamped * 10;
  const zone =
    meters >= 6000
      ? "HADAL"
      : meters >= 4000
        ? "ABYSSAL"
        : meters >= 1000
          ? "MIDNIGHT"
          : meters >= 200
            ? "TWILIGHT"
            : "SUNLIGHT";

  return (
    <div className={`flex flex-col gap-1 ${className}`}>
      <div className="flex justify-between text-sm text-cyan-accent/70">
        <span>SURFACE</span>
        <span>
          {meters.toLocaleString()} m · {zone}
        </span>
        <span>TRENCH</span>
      </div>
      <div className="relative h-3 w-full overflow-hidden rounded-sm bg-[#061018] ring-1 ring-cyan-900/60">
        <div
          className="absolute inset-y-0 left-0 transition-all duration-700 ease-out motion-reduce:transition-none"
          style={{
            width: `${clamped}%`,
            background:
              "linear-gradient(90deg, #0e7490 0%, #155e75 28%, #1e3a8a 62%, #312e81 88%, #fde68a 100%)",
          }}
        />
        {Array.from({ length: 14 }, (_, i) => (
          <div
            key={i}
            className="absolute inset-y-0 w-px bg-black/35"
            style={{ left: `${((i + 1) / 15) * 100}%` }}
          />
        ))}
        <div
          className="absolute top-1/2 h-2.5 w-2.5 -translate-y-1/2 rounded-full bg-cyan-100 shadow-[0_0_10px_#67e8f9] transition-all duration-700 motion-reduce:transition-none"
          style={{ left: `calc(${clamped}% - 5px)` }}
        />
      </div>
    </div>
  );
}

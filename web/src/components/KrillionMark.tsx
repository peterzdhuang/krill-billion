import Image from "next/image";
import type { TierName } from "@/lib/protocol";
import { TIER_AURA, TIER_MARK } from "@/lib/krillionAssets";

const LEGACY = {
  krillion: "/krillion/krillion.png",
  schooler: "/krillion/schooler.png",
  rare: "/krillion/rare.png",
} as const;

export function KrillionMark({
  name,
  tier,
  size = 40,
  aura = false,
}: {
  name?: keyof typeof LEGACY;
  tier?: TierName;
  size?: number;
  aura?: boolean;
}) {
  const src = tier ? TIER_MARK[tier] : name ? LEGACY[name] : TIER_MARK.Intern;
  const auraSrc = aura && tier ? TIER_AURA[tier] : undefined;
  return (
    <span
      className="relative inline-flex items-center justify-center"
      style={{ width: size, height: size }}
    >
      {auraSrc && (
        <Image
          src={auraSrc}
          alt=""
          width={Math.round(size * 1.7)}
          height={Math.round(size * 1.7)}
          className="kb-aura pointer-events-none absolute left-1/2 top-1/2 max-w-none -translate-x-1/2 -translate-y-1/2"
          style={{ width: size * 1.7, height: size * 1.7 }}
        />
      )}
      <Image
        src={src}
        alt=""
        width={size}
        height={size}
        className="relative z-10 [image-rendering:pixelated]"
        style={{ width: size, height: size }}
      />
    </span>
  );
}

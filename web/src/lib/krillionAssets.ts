import type { TierName } from "@/lib/protocol";

/** Official Krillion tier marks → career tiers used in scoring. */
export const TIER_MARK: Record<TierName, string> = {
  Intern: "/krillion/tiers/plankton.png",
  Junior: "/krillion/tiers/schooler.png",
  Senior: "/krillion/tiers/rare.png",
  Staff: "/krillion/tiers/deepcut.png",
  Principal: "/krillion/tiers/deepcut-gold.png",
  Distinguished: "/krillion/tiers/rare-gold.png",
  CTO: "/krillion/tiers/krillion.png",
  "Jeffy B": "/krillion/tiers/krillion-gold.png",
};

export const TIER_AURA: Partial<Record<TierName, string>> = {
  Principal: "/krillion/fx/mantis-aura.webp",
  Distinguished: "/krillion/fx/fossil-aura.webp",
  CTO: "/krillion/fx/leviathan-aura.webp",
  "Jeffy B": "/krillion/fx/leviathan-gold-aura.webp",
};

export const WATER_BG = "/krillion/bg/water-bg.webp";
export const PUFFER = ["/krillion/party/puffer-0.png", "/krillion/party/puffer-1.png"] as const;

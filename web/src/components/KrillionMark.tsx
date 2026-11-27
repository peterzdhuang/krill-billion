import Image from "next/image";

const SRC = {
  krillion: "/krillion/krillion.png",
  schooler: "/krillion/schooler.png",
  rare: "/krillion/rare.png",
} as const;

export function KrillionMark({
  name,
  size = 40,
}: {
  name: keyof typeof SRC;
  size?: number;
}) {
  return (
    <Image
      src={SRC[name]}
      alt=""
      width={size}
      height={size}
      className="[image-rendering:pixelated]"
      style={{ width: size, height: size }}
    />
  );
}

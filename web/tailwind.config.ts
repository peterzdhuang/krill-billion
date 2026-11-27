import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        abyss: "#050a14",
        deep: "#0a1628",
        midnavy: "#0f2744",
        cyan: {
          accent: "#2dd4bf",
          glow: "#5eead4",
        },
      },
      fontFamily: {
        mono: ["var(--font-vt323)", "ui-monospace", "monospace"],
      },
    },
  },
  plugins: [],
};

export default config;

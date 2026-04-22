import type { Config } from "tailwindcss";
import typography from "@tailwindcss/typography";

const config: Config = {
  content: ["./src/**/*.{js,ts,jsx,tsx,mdx}"],
  theme: {
    extend: {
      colors: {
        "molecule-bg": {
          950: "#060b14",
          900: "#0a0f1a",
          800: "#111827",
          700: "#1e293b",
        },
        "molecule-surface": {
          900: "#0f1629",
          800: "#151d30",
          700: "#1a2540",
          600: "#1f2d4d",
        },
        "molecule-accent": {
          mint: "#39e58c",
          cyan: "#22d1ee",
          "mint-dim": "#39e58c80",
          "cyan-dim": "#22d1ee80",
        },
        "molecule-border": {
          DEFAULT: "rgba(255, 255, 255, 0.06)",
          light: "rgba(255, 255, 255, 0.10)",
          warm: "rgba(148, 163, 184, 0.12)",
        },
      },
      boxShadow: {
        "glow-mint": "0 0 20px rgba(57, 229, 140, 0.15), 0 0 60px rgba(57, 229, 140, 0.05)",
        "glow-cyan": "0 0 20px rgba(34, 209, 238, 0.15), 0 0 60px rgba(34, 209, 238, 0.05)",
        "glow-status": "0 0 12px rgba(57, 229, 140, 0.3)",
        "glow-mint-lg": "0 0 40px rgba(57, 229, 140, 0.2), 0 0 80px rgba(57, 229, 140, 0.08)",
        "glow-cyan-lg": "0 0 40px rgba(34, 209, 238, 0.2), 0 0 80px rgba(34, 209, 238, 0.08)",
        "premium": "0 8px 32px rgba(0, 0, 0, 0.4), 0 2px 8px rgba(0, 0, 0, 0.2)",
        "premium-lg": "0 16px 48px rgba(0, 0, 0, 0.5), 0 4px 16px rgba(0, 0, 0, 0.3)",
      },
      backgroundImage: {
        "gradient-radial": "radial-gradient(var(--tw-gradient-stops))",
        "gradient-mint-cyan": "linear-gradient(135deg, #39e58c, #22d1ee)",
        "gradient-mint-cyan-subtle": "linear-gradient(135deg, rgba(57, 229, 140, 0.1), rgba(34, 209, 238, 0.1))",
      },
    },
  },
  plugins: [typography],
};

export default config;

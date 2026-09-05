/** @type {import('tailwindcss').Config} */
export default {
  content: ["./src/**/*.{html,js,svelte,ts}"],
  darkMode: "class",
  theme: {
    extend: {
      fontFamily: {
        sans: [
          "Manrope",
          "PingFang SC",
          "Microsoft YaHei",
          "ui-sans-serif",
          "system-ui",
          "sans-serif",
        ],
        mono: ["IBM Plex Mono", "ui-monospace", "SFMono-Regular", "monospace"],
      },
      colors: {
        ink: {
          50: "#f5f7f7",
          100: "#e3e8e9",
          200: "#c9d0d2",
          300: "#b0b9bc",
          400: "#9aa5aa",
          500: "#828d92",
          600: "#535c61",
          700: "#353b3e",
          800: "#222629",
          900: "#181b1d",
          950: "#111314",
        },
        accent: { 400: "#c6e1d1", 500: "#afd4bf", 600: "#719f87" },
        loss: { 400: "#efaaa5", 500: "#e58d89", 600: "#c36d69" },
      },
    },
  },
  plugins: [],
};

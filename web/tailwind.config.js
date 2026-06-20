/** @type {import('tailwindcss').Config} */
// Palette ported from the Claude design canvas: blue-tinted dark grays (hue 240),
// teal/emerald "pos" accent (hue 168), warm-red "neg" (hue 24). Colors carry the
// `/ <alpha-value>` placeholder so Tailwind opacity modifiers (bg-accent-500/10)
// still work with oklch.
export default {
  content: ['./src/**/*.{html,js,svelte,ts}'],
  darkMode: 'class',
  theme: {
    extend: {
      fontFamily: {
        sans: ['Manrope', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        mono: ['IBM Plex Mono', 'ui-monospace', 'SFMono-Regular', 'monospace']
      },
      colors: {
        ink: {
          50: 'oklch(0.97 0.004 240 / <alpha-value>)',
          100: 'oklch(0.92 0.005 240 / <alpha-value>)',
          200: 'oklch(0.85 0.006 240 / <alpha-value>)',
          300: 'oklch(0.73 0.008 240 / <alpha-value>)',
          400: 'oklch(0.62 0.01 240 / <alpha-value>)',
          500: 'oklch(0.54 0.01 240 / <alpha-value>)',
          600: 'oklch(0.40 0.012 240 / <alpha-value>)',
          700: 'oklch(0.30 0.01 240 / <alpha-value>)',
          800: 'oklch(0.235 0.01 240 / <alpha-value>)',
          900: 'oklch(0.205 0.008 240 / <alpha-value>)',
          950: 'oklch(0.165 0.006 240 / <alpha-value>)'
        },
        accent: {
          400: 'oklch(0.84 0.12 168 / <alpha-value>)',
          500: 'oklch(0.80 0.115 168 / <alpha-value>)',
          600: 'oklch(0.66 0.11 182 / <alpha-value>)'
        },
        loss: {
          400: 'oklch(0.74 0.16 24 / <alpha-value>)',
          500: 'oklch(0.70 0.155 24 / <alpha-value>)',
          600: 'oklch(0.62 0.15 24 / <alpha-value>)'
        }
      },
      boxShadow: {
        soft: '0 1px 2px rgba(0,0,0,0.04), 0 8px 16px rgba(0,0,0,0.06)',
        glow: '0 4px 14px -4px oklch(0.80 0.115 168 / 0.6)'
      }
    }
  },
  plugins: []
};

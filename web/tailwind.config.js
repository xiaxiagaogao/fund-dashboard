/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{html,js,svelte,ts}'],
  darkMode: 'class',
  theme: {
    extend: {
      fontFamily: {
        sans: ['Inter', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'ui-monospace', 'SFMono-Regular', 'monospace']
      },
      colors: {
        ink: {
          50: '#fafafa',
          100: '#f4f4f5',
          200: '#e4e4e7',
          300: '#a1a1aa',
          400: '#71717a',
          500: '#52525b',
          600: '#3f3f46',
          700: '#27272a',
          800: '#18181b',
          900: '#09090b',
          950: '#030305'
        },
        accent: {
          400: '#a3e635',
          500: '#84cc16',
          600: '#65a30d'
        },
        loss: {
          400: '#f87171',
          500: '#ef4444',
          600: '#dc2626'
        }
      },
      boxShadow: {
        soft: '0 1px 2px rgba(0,0,0,0.04), 0 8px 16px rgba(0,0,0,0.06)',
        ring: '0 0 0 1px rgba(255,255,255,0.04)'
      }
    }
  },
  plugins: []
};

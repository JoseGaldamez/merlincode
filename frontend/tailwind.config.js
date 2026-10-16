/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './index.html',
    './src/**/*.{js,ts,jsx,tsx}',
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        canvas: 'var(--bg-canvas)',
        surface: {
          DEFAULT: 'var(--surface-neutral)',
          dim: 'var(--surface-neutral)',
          bright: 'var(--surface-bright)',
          lowest: 'var(--surface-lowest)',
          low: 'var(--surface-card)',
          container: 'var(--surface-container)',
          high: 'var(--surface-elevated)',
          highest: 'var(--surface-strong)',
          neutral: 'var(--surface-neutral)',
          card: 'var(--surface-card)',
          modal: 'var(--surface-elevated)',
          base: 'var(--surface-container)',
          sidebar: 'var(--surface-sidebar)',
          hover: 'var(--surface-hover)',
          selected: 'var(--surface-selected)'
        },
        border: {
          subtle: 'var(--border-subtle)',
          petrol: 'var(--border-petrol)',
          strong: 'var(--border-strong)',
          hairline: 'var(--border-hairline)'
        },
        accent: {
          primary: 'var(--accent-primary)',
          'primary-hover': 'var(--accent-primary-hover)',
          mint: 'var(--accent-primary-hover)',
          petrol: 'var(--accent-petrol)',
          'petrol-hover': 'var(--accent-petrol-hover)',
          cyan: 'var(--accent-tertiary)'
        },
        brand: {
          primary: 'var(--accent-primary)',
          secondary: 'var(--accent-petrol)',
          tertiary: 'var(--accent-tertiary)'
        },
        content: {
          headline: 'var(--text-headline)',
          body: 'var(--text-body)',
          muted: 'var(--text-muted)',
          dim: 'var(--text-dim)'
        },
        status: {
          success: 'var(--status-success)',
          info: 'var(--status-thought)',
          warning: 'var(--status-warning)',
          error: 'var(--status-error)'
        }
      },
      fontFamily: {
        sans: ['Geist', 'Inter', 'Segoe UI', '-apple-system', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'Consolas', 'monospace'],
      },
      fontSize: {
        md: ['1rem', { lineHeight: '1.5rem' }],
      },
      borderRadius: {
        xs: '2px',
        sm: '4px',
        md: '8px',
        lg: '12px',
      },
      boxShadow: {
        glow: '0 0 20px -4px color-mix(in srgb, var(--accent-primary) 25%, transparent)',
        'glow-lg': '0 0 30px -6px color-mix(in srgb, var(--accent-primary) 40%, transparent)',
        modal: '0 16px 40px rgba(0, 0, 0, 0.7)',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.98)' },
          '100%': { opacity: '1', transform: 'scale(1)' },
        },
      },
      animation: {
        fadeIn: 'fadeIn 0.15s ease-out',
        scaleIn: 'scaleIn 0.15s ease-out',
      },
    },
  },
  plugins: [],
};

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
        canvas: '#0b0e10',
        surface: {
          DEFAULT: '#111416',
          dim: '#111416',
          bright: '#37393c',
          lowest: '#0c0f11',
          low: '#191c1e',
          container: '#1d2022',
          high: '#272a2c',
          highest: '#323537',
          neutral: '#111416',
          card: '#151b1e',
          modal: '#1a2328',
          base: '#182024',
          sidebar: '#101518',
        },
        border: {
          subtle: '#222d32',
          petrol: '#1f3d44',
          strong: '#3c4a46',
          hairline: 'rgba(255, 255, 255, 0.07)',
        },
        accent: {
          primary: '#2dd4bf',
          'primary-hover': '#57f1db',
          mint: '#57f1db',
          petrol: '#1f3d44',
          'petrol-hover': '#264b54',
          cyan: '#06b6d4',
        },
        brand: {
          primary: '#2dd4bf',
          secondary: '#1f3d44',
          tertiary: '#06b6d4',
        },
        content: {
          headline: '#f8fafc',
          body: '#e2e8f0',
          muted: '#7e8b91',
          dim: '#5a6569',
        },
        status: {
          success: '#10b981',
          info: '#38bdf8',
          warning: '#f59e0b',
          error: '#ef4444',
        },
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
        glow: '0 0 20px -4px rgba(45, 212, 191, 0.25)',
        'glow-lg': '0 0 30px -6px rgba(45, 212, 191, 0.4)',
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

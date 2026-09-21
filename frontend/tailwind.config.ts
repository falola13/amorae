import type { Config } from 'tailwindcss';

// Tokens mirror the Amorae design system board one to one.
const config: Config = {
  content: ['./src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        bg: '#F7F4EF',
        paper: '#F2EDE5',
        surface: '#FFFDFA',
        ink: '#24201F',
        stone: '#6B635C',
        line: '#E4DDD3',
        edge: '#8F857B',
        faint: '#DDD4C8',
        photo: '#EAE3D9',
        plum: { DEFAULT: '#5B2A4A', dark: '#47203A', tint: '#F1E8EE', soft: '#B59AAB' },
        green: { DEFAULT: '#3F6B52', tint: '#E4EDE6' },
        amber: { DEFAULT: '#7F5210', tint: '#F6EBD6' },
        red: { DEFAULT: '#9B3B32', tint: '#F6E7E4' },
      },
      fontFamily: {
        sans: ['"Manrope Variable"', 'Manrope', 'ui-sans-serif', 'system-ui', '-apple-system', '"Segoe UI"', 'sans-serif'],
      },
      fontSize: {
        micro: ['12px', { lineHeight: '1.4', letterSpacing: '0.08em', fontWeight: '700' }],
        support: ['14px', { lineHeight: '1.5' }],
        body: ['16px', { lineHeight: '1.55' }],
        bodylg: ['17px', { lineHeight: '1.55' }],
        reading: ['21px', { lineHeight: '1.6' }],
        section: ['20px', { lineHeight: '1.3', letterSpacing: '-0.015em', fontWeight: '600' }],
        title: ['28px', { lineHeight: '1.18', letterSpacing: '-0.022em', fontWeight: '600' }],
        display: ['34px', { lineHeight: '1.18', letterSpacing: '-0.025em', fontWeight: '600' }],
      },
      borderRadius: { input: '12px', btn: '14px', card: '16px', sheet: '24px' },
      keyframes: {
        breathe: { '0%,100%': { opacity: '0.25' }, '50%': { opacity: '1' } },
        rise: { from: { opacity: '0', transform: 'translateY(6px)' }, to: { opacity: '1', transform: 'none' } },
        sheet: { from: { opacity: '0', transform: 'translateY(24px)' }, to: { opacity: '1', transform: 'none' } },
        fade: { from: { opacity: '0' }, to: { opacity: '1' } },
        draw: { to: { strokeDashoffset: '0' } },
      },
      animation: {
        breathe: 'breathe 1.8s ease-in-out infinite',
        rise: 'rise 0.24s ease-out both',
        page: 'rise 0.2s ease-out both',
        sheet: 'sheet 0.32s cubic-bezier(0.2,0.8,0.2,1) both',
        fade: 'fade 0.24s ease-out both',
        draw: 'draw 0.32s 0.08s ease-out forwards',
      },
    },
  },
  plugins: [],
};
export default config;

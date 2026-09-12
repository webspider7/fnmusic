/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        dark: { 950: '#07080d', 900: '#0c0e17', 850: '#111422', 800: '#171b2d', 700: '#232842' },
        aurora: { cyan: '#00f2fe', magenta: '#f72585', violet: '#7209b7', purple: '#4cc9f0', emerald: '#00f5d4', neon: '#39ff14' },
      },
      animation: { 'spin-slow': 'spin 18s linear infinite' },
    },
  },
  plugins: [],
}

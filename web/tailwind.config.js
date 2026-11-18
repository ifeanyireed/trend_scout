/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Dropa'],
        serif: ['Dropa'],
        mono: ['Dropa'],
        display: ['Dropa'],
      },
      colors: {
        'nexa-brand': 'var(--nexa-brand)',
        'nexa-brand-light': 'var(--nexa-brand-light)',
        'nexa-brand-mid': 'var(--nexa-brand-mid)',
        'nexa-brand-glow': 'var(--nexa-brand-glow)',
        'nexa-accent': 'var(--nexa-accent)',
        'nexa-accent-light': 'var(--nexa-accent-light)',
        'nexa-amber': 'var(--nexa-amber)',
        'nexa-amber-light': 'var(--nexa-amber-light)',
        'nexa-coral': 'var(--nexa-coral)',
        'nexa-coral-light': 'var(--nexa-coral-light)',
        'nexa-bg-base': 'var(--nexa-bg-base)',
        'nexa-bg-surface': 'var(--nexa-bg-surface)',
        'nexa-bg-overlay': 'var(--nexa-bg-overlay)',
        'nexa-bg-glass': 'var(--nexa-bg-glass)',
        'nexa-text-primary': 'var(--nexa-text-primary)',
        'nexa-text-secondary': 'var(--nexa-text-secondary)',
        'nexa-text-muted': 'var(--nexa-text-muted)',
        'nexa-text-faint': 'var(--nexa-text-faint)',
        'nexa-border': 'var(--nexa-border)',
        'nexa-border-mid': 'var(--nexa-border-mid)',
      },
    },
  },
  plugins: [],
};

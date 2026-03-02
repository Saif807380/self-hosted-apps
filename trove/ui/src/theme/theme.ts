import { createSystem, defaultConfig, defineConfig } from '@chakra-ui/react'

const config = defineConfig({
  globalCss: {
    'html, body': {
      fontFamily: "'DM Sans', system-ui, sans-serif",
    },
  },
  theme: {
    tokens: {
      fonts: {
        heading: { value: "'Lora', Georgia, serif" },
        body:    { value: "'DM Sans', system-ui, sans-serif" },
      },
      colors: {
        brand: {
          50:  { value: '#eff6ff' },
          100: { value: '#dbeafe' },
          200: { value: '#bfdbfe' },
          300: { value: '#93c5fd' },
          400: { value: '#60a5fa' },
          500: { value: '#3b82f6' },
          600: { value: '#2563eb' },
          700: { value: '#1d4ed8' },
          800: { value: '#1e40af' },
          900: { value: '#1e3a8a' },
        },
      },
    },
    semanticTokens: {
      colors: {
        'bg.canvas': {
          value: { base: '#f7f6f3', _dark: '#0e0f12' },
        },
        'bg.surface': {
          value: { base: 'white', _dark: '#1a1b1f' },
        },
        'bg.subtle': {
          value: { base: '#eeecea', _dark: '#25262b' },
        },
        'border.default': {
          value: { base: '#e5e2dc', _dark: '#2e2f35' },
        },
        'text.primary': {
          value: { base: '#1c1c1e', _dark: '#f2f2f7' },
        },
        'text.secondary': {
          value: { base: '#6b6b6b', _dark: '#8e8e93' },
        },
        'text.muted': {
          value: { base: '#a3a3a3', _dark: '#636366' },
        },
        'accent': {
          value: { base: '{colors.brand.600}', _dark: '{colors.brand.400}' },
        },
        'accent.subtle': {
          value: { base: '{colors.brand.50}', _dark: '#172554' },
        },
      },
    },
  },
})

export const system = createSystem(defaultConfig, config)

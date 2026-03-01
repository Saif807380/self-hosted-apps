import { createSystem, defaultConfig, defineConfig } from '@chakra-ui/react'

const config = defineConfig({
  theme: {
    tokens: {
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
          value: { base: '{colors.gray.50}', _dark: '{colors.gray.950}' },
        },
        'bg.surface': {
          value: { base: 'white', _dark: '{colors.gray.900}' },
        },
        'bg.subtle': {
          value: { base: '{colors.gray.100}', _dark: '{colors.gray.800}' },
        },
        'border.default': {
          value: { base: '{colors.gray.200}', _dark: '{colors.gray.700}' },
        },
        'text.primary': {
          value: { base: '{colors.gray.900}', _dark: 'white' },
        },
        'text.secondary': {
          value: { base: '{colors.gray.600}', _dark: '{colors.gray.400}' },
        },
        'accent': {
          value: { base: '{colors.brand.500}', _dark: '{colors.brand.400}' },
        },
      },
    },
  },
})

export const system = createSystem(defaultConfig, config)

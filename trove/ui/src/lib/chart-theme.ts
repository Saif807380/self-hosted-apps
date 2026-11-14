// Theme colours resolved to hex for Recharts, whose SVG attributes can't use
// CSS var() references. Mirrors the semantic tokens in theme/theme.ts.
export function chartTheme(isDark: boolean) {
  return {
    muted: isDark ? '#636366' : '#a3a3a3',
    border: isDark ? '#2e2f35' : '#e5e2dc',
    surface: isDark ? '#1a1b1f' : '#ffffff',
    primary: isDark ? '#f2f2f7' : '#1c1c1e',
    secondary: isDark ? '#8e8e93' : '#6b6b6b',
  }
}

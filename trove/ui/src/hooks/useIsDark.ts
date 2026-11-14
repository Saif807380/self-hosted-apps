import { useEffect, useState } from 'react'

// Tracks dark mode by watching the `dark` class on <html>. Charts need it:
// Recharts SVG attributes don't support CSS var() references (they're not CSS
// properties), so colours have to be resolved to hex in JS.
export function useIsDark() {
  const [isDark, setIsDark] = useState(() => document.documentElement.classList.contains('dark'))
  useEffect(() => {
    const obs = new MutationObserver(() =>
      setIsDark(document.documentElement.classList.contains('dark')),
    )
    obs.observe(document.documentElement, { attributeFilter: ['class'] })
    return () => obs.disconnect()
  }, [])
  return isDark
}

import { IconButton } from '@chakra-ui/react'
import { useEffect, useState } from 'react'

export default function ThemeToggle() {
  const [isDark, setIsDark] = useState(() =>
    document.documentElement.getAttribute('data-theme') === 'dark',
  )

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', isDark ? 'dark' : 'light')
  }, [isDark])

  return (
    <IconButton
      aria-label="Toggle color mode"
      onClick={() => setIsDark((d) => !d)}
      variant="ghost"
      size="sm"
    >
      {isDark ? '☀️' : '🌙'}
    </IconButton>
  )
}

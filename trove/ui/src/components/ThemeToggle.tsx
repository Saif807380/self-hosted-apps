import { useState } from 'react'
import { IconButton } from '@chakra-ui/react'

export default function ThemeToggle() {
  const [dark, setDark] = useState(() => document.documentElement.classList.contains('dark'))

  function handleToggle() {
    const isDark = !dark
    document.documentElement.classList.toggle('dark', isDark)
    localStorage.setItem('color-mode', isDark ? 'dark' : 'light')
    setDark(isDark)
  }

  return (
    <IconButton aria-label="Toggle color mode" onClick={handleToggle} variant="ghost" size="sm">
      {dark ? '☀️' : '🌙'}
    </IconButton>
  )
}

import { useState } from 'react'
import { Flex } from '@chakra-ui/react'

interface StarRatingProps {
  value: number
  onChange?: (value: number) => void
  readOnly?: boolean
  size?: 'sm' | 'md' | 'lg'
}

const SIZE_MAP = { sm: '14px', md: '18px', lg: '22px' }

const STAR_STYLE: React.CSSProperties = {
  lineHeight: '1',
  transition: 'color 0.1s ease',
  background: 'none',
  border: 'none',
  padding: 0,
  outline: 'none',
  cursor: 'pointer',
}

export default function StarRating({
  value,
  onChange,
  readOnly = false,
  size = 'md',
}: StarRatingProps) {
  const [hovered, setHovered] = useState(0)
  const display = hovered || value
  const fontSize = SIZE_MAP[size]

  return (
    <Flex gap={0.5} align="center" display="inline-flex">
      {[1, 2, 3, 4, 5].map(star =>
        readOnly ? (
          <span
            key={star}
            style={{
              fontSize,
              lineHeight: '1',
              color: star <= display ? '#f59e0b' : '#a3a3a3',
            }}
          >
            {star <= display ? '★' : '☆'}
          </span>
        ) : (
          <button
            key={star}
            type="button"
            style={{
              ...STAR_STYLE,
              fontSize,
              color: star <= display ? '#f59e0b' : '#a3a3a3',
            }}
            onMouseEnter={() => setHovered(star)}
            onMouseLeave={() => setHovered(0)}
            onClick={() => onChange && onChange(star === value ? 0 : star)}
            aria-label={`Rate ${star} star${star > 1 ? 's' : ''}`}
          >
            {star <= display ? '★' : '☆'}
          </button>
        ),
      )}
    </Flex>
  )
}

import { Box } from '@chakra-ui/react'

interface TagBadgeProps {
  name: string
  onRemove?: () => void
  size?: 'sm' | 'md'
}

export default function TagBadge({ name, onRemove, size = 'sm' }: TagBadgeProps) {
  return (
    <Box
      display="inline-flex"
      alignItems="center"
      gap={1}
      px={size === 'sm' ? 2 : 3}
      py={size === 'sm' ? '2px' : 1}
      borderRadius="full"
      fontSize={size === 'sm' ? 'xs' : 'sm'}
      fontWeight="500"
      bg="accent.subtle"
      color="accent"
      letterSpacing="0.01em"
    >
      {name}
      {onRemove && (
        <button
          type="button"
          onClick={onRemove}
          aria-label={`Remove ${name} tag`}
          style={{
            display: 'flex',
            alignItems: 'center',
            background: 'none',
            border: 'none',
            padding: 0,
            marginLeft: '2px',
            cursor: 'pointer',
            lineHeight: 1,
            color: 'inherit',
            opacity: 0.7,
            fontSize: '1em',
          }}
          onMouseEnter={e => { e.currentTarget.style.opacity = '1' }}
          onMouseLeave={e => { e.currentTarget.style.opacity = '0.7' }}
        >
          ×
        </button>
      )}
    </Box>
  )
}

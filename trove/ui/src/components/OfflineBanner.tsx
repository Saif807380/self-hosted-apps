import { Box, Text } from '@chakra-ui/react'
import { useSync } from '@/hooks/useSync'

export default function OfflineBanner() {
  const { status } = useSync()
  if (status !== 'offline') return null

  return (
    <Box
      style={{
        background: 'rgba(234, 179, 8, 0.12)',
        borderBottom: '1px solid rgba(234, 179, 8, 0.25)',
        padding: '6px 24px',
        textAlign: 'center',
      }}
    >
      <Text fontSize="xs" color="text.secondary">
        Working offline — changes will sync when connected
      </Text>
    </Box>
  )
}

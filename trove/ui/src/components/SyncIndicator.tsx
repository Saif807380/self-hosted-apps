import { useState, useRef, useEffect } from 'react'
import { Box, Button, Text, Spinner, Flex, Stack } from '@chakra-ui/react'
import { useLiveQuery } from 'dexie-react-hooks'
import { useSync } from '@/hooks/useSync'
import { db } from '@/lib/db'

function formatTime(iso: string | null): string {
  if (!iso) return 'Never'
  return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

const STATUS_DOT_COLOR: Record<string, string> = {
  syncing: '#3b82f6',
  error:   '#ef4444',
  offline: '#9ca3af',
  idle:    '#22c55e',
}
const PENDING_DOT_COLOR = '#f59e0b'

export default function SyncIndicator() {
  const { status, lastSyncTime, error, syncNow, bootstrapping } = useSync()
  const pendingChanges = useLiveQuery(() => db.syncOutbox.count(), [], 0) ?? 0
  const [open, setOpen] = useState(false)
  const [triggering, setTriggering] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onDown = (e: MouseEvent) => {
      if (!containerRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDown)
    return () => document.removeEventListener('mousedown', onDown)
  }, [open])

  const handleSyncNow = async () => {
    setTriggering(true)
    try { await syncNow() } finally { setTriggering(false) }
  }

  const effectiveStatus = bootstrapping ? 'syncing' : status
  const hasPending = pendingChanges > 0
  const dotColor = effectiveStatus === 'idle' && hasPending
    ? PENDING_DOT_COLOR
    : STATUS_DOT_COLOR[effectiveStatus] ?? STATUS_DOT_COLOR.idle

  const label = bootstrapping ? 'Loading…'
    : status === 'syncing' ? 'Syncing…'
    : status === 'error'   ? 'Sync error'
    : status === 'offline' ? 'Offline'
    : hasPending           ? `${pendingChanges} pending`
    :                        'Up to date'

  return (
    <Box position="relative" ref={containerRef}>
      <button
        onClick={() => setOpen(o => !o)}
        aria-label={`Sync: ${label}`}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: '6px',
          padding: '5px 10px',
          borderRadius: '8px',
          border: 'none',
          background: 'transparent',
          cursor: 'pointer',
          color: 'var(--chakra-colors-text-secondary)',
          fontSize: '0.8125rem',
        }}
      >
        {effectiveStatus === 'syncing' ? (
          <Spinner size="xs" style={{ color: dotColor }} />
        ) : (
          <Box
            w="7px" h="7px" borderRadius="full" flexShrink={0}
            style={{ background: dotColor }}
          />
        )}
        <span>{label}</span>
      </button>

      {open && (
        <Box
          position="absolute"
          top="calc(100% + 6px)"
          right={0}
          bg="bg.surface"
          border="1px solid"
          borderColor="border.default"
          borderRadius="10px"
          p={3}
          w="210px"
          zIndex={200}
          style={{ boxShadow: '0 4px 20px rgba(0,0,0,0.12)' }}
        >
          <Stack gap={2}>
            <Flex align="center" justify="space-between">
              <Text fontSize="xs" fontWeight="600" color="text.primary">{label}</Text>
            </Flex>

            {status === 'error' && error && (
              <Text fontSize="xs" color="red.400" style={{ wordBreak: 'break-word' }}>
                {error}
              </Text>
            )}

            <Text fontSize="xs" color="text.muted">
              Last sync: {formatTime(lastSyncTime)}
            </Text>

            {hasPending && (
              <Text fontSize="xs" color="text.muted">
                {pendingChanges} change{pendingChanges !== 1 ? 's' : ''} pending
              </Text>
            )}

            <Button
              size="xs"
              variant="outline"
              borderRadius="7px"
              mt={0.5}
              onClick={handleSyncNow}
              disabled={triggering || status === 'offline' || bootstrapping}
              loading={triggering}
            >
              Sync now
            </Button>
          </Stack>
        </Box>
      )}
    </Box>
  )
}

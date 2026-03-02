import { useState } from 'react'
import { Dialog, Button, Text, Portal } from '@chakra-ui/react'

interface ConfirmDialogProps {
  isOpen: boolean
  onClose: () => void
  onConfirm: () => Promise<void>
  title: string
  message: string
  confirmLabel?: string
}

export default function ConfirmDialog({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmLabel = 'Delete',
}: ConfirmDialogProps) {
  const [loading, setLoading] = useState(false)

  const handleConfirm = async () => {
    setLoading(true)
    try {
      await onConfirm()
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog.Root
      open={isOpen}
      onOpenChange={({ open }) => !open && !loading && onClose()}
      role="alertdialog"
    >
      <Portal>
        <Dialog.Backdrop />
        <Dialog.Positioner>
          <Dialog.Content maxW="400px" mx={4}>
            <Dialog.Header pb={2}>
              <Dialog.Title fontFamily="heading" fontWeight="600" fontSize="lg">
                {title}
              </Dialog.Title>
            </Dialog.Header>
            <Dialog.Body pb={4}>
              <Text color="text.secondary" fontSize="sm">
                {message}
              </Text>
            </Dialog.Body>
            <Dialog.Footer gap={2}>
              <Button
                variant="outline"
                size="sm"
                onClick={onClose}
                disabled={loading}
              >
                Cancel
              </Button>
              <Button
                colorPalette="red"
                size="sm"
                onClick={handleConfirm}
                loading={loading}
                loadingText="Deleting…"
              >
                {confirmLabel}
              </Button>
            </Dialog.Footer>
          </Dialog.Content>
        </Dialog.Positioner>
      </Portal>
    </Dialog.Root>
  )
}

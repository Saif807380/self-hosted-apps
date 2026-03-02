import type { ToastOptions } from '@ark-ui/react/toast'
import {
  Toaster,
  ToastRoot,
  ToastTitle,
  ToastDescription,
  ToastCloseTrigger,
  ToastIndicator,
  Stack,
  CloseButton,
} from '@chakra-ui/react'
import { toaster } from '@/lib/toaster'

export function AppToaster() {
  return (
    <Toaster toaster={toaster}>
      {(toast: ToastOptions) => (
        <ToastRoot width="sm" gap={3}>
          <ToastIndicator />
          <Stack gap={1} flex={1} maxW="100%">
            {toast.title && <ToastTitle>{toast.title}</ToastTitle>}
            {toast.description && (
              <ToastDescription>{toast.description}</ToastDescription>
            )}
          </Stack>
          <ToastCloseTrigger asChild>
            <CloseButton size="sm" mt={-1} me={-1} />
          </ToastCloseTrigger>
        </ToastRoot>
      )}
    </Toaster>
  )
}

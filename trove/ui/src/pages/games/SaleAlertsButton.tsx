import { useState } from 'react'
import { Button, Menu, Portal } from '@chakra-ui/react'
import { toaster } from '@/lib/toaster'
import { usePushAlerts, type PushState } from '@/hooks/usePushAlerts'

// Why alerts can't be turned on, shown when the button is tapped.
const UNAVAILABLE: Partial<Record<PushState, string>> = {
  unsupported: "This browser can't receive web push.",
  unavailable: "The server isn't set up for alerts yet (VAPID keys missing).",
  blocked: 'Notifications are blocked for Trove. Allow them in your browser or phone settings.',
}

/** Turns sale notifications on or off for this device. */
export default function SaleAlertsButton({ compact }: { compact?: boolean }) {
  const { state, detail, enable, disable, sendTest } = usePushAlerts()
  const [busy, setBusy] = useState(false)

  const run = async (action: () => Promise<void>, failure: string) => {
    setBusy(true)
    try {
      await action()
    } catch {
      toaster.create({ title: failure, type: 'error', duration: 4000 })
    } finally {
      setBusy(false)
    }
  }

  const handleEnable = () => run(async () => {
    await enable()
    toaster.create({ title: 'Sale alerts on for this device', type: 'success', duration: 3000 })
  }, "Couldn't turn on sale alerts")

  const handleTest = () => run(async () => {
    const delivered = await sendTest()
    toaster.create({
      title: delivered ? `Test sent to ${delivered} ${delivered === 1 ? 'device' : 'devices'}` : 'No device accepted the test',
      type: delivered ? 'success' : 'warning', duration: 3000,
    })
  }, "Couldn't send a test")

  const handleDisable = () => run(disable, "Couldn't turn off sale alerts")

  if (state === 'loading') return null

  if (state !== 'on') {
    // Not disabled when alerts can't work: a tooltip can't be read on a
    // phone, so tapping explains why instead.
    const reason = (state === 'unsupported' && detail) || UNAVAILABLE[state]
    const handleClick = reason
      ? () => toaster.create({ title: "Sale alerts can't be turned on", description: reason, type: 'warning', duration: 6000 })
      : handleEnable
    return (
      <Button
        size="sm" variant="outline" borderRadius="8px" onClick={handleClick}
        opacity={reason ? 0.6 : 1} loading={busy}
      >
        🔔 {compact ? 'Alerts' : 'Turn on sale alerts'}
      </Button>
    )
  }

  return (
    <Menu.Root positioning={{ placement: 'bottom-end' }}>
      <Menu.Trigger asChild>
        <Button size="sm" variant="outline" borderRadius="8px" loading={busy}>
          🔔 {compact ? 'On' : 'Alerts on'}
        </Button>
      </Menu.Trigger>
      <Portal>
        <Menu.Positioner>
          <Menu.Content>
            <Menu.Item value="test" onClick={handleTest}>Send a test alert</Menu.Item>
            <Menu.Item value="off" onClick={handleDisable} color="fg.error">Turn off on this device</Menu.Item>
          </Menu.Content>
        </Menu.Positioner>
      </Portal>
    </Menu.Root>
  )
}

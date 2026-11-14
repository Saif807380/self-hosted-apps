import type { ReactNode } from 'react'
import { Button, Flex, Spinner, Text } from '@chakra-ui/react'

// Small pieces shared by the sections of a game's detail view.

export function LinkButton({ href, children }: { href: string; children: ReactNode }) {
  return (
    <Button
      asChild size="xs" variant="outline" borderRadius="6px"
      bg="bg.surface" borderColor="text.muted" _hover={{ borderColor: 'text.secondary' }}
    >
      <a href={href} target="_blank" rel="noreferrer">{children}</a>
    </Button>
  )
}

export function SectionTitle({ children }: { children: ReactNode }) {
  return (
    <Text fontSize="xs" color="text.muted" fontWeight="500" textTransform="uppercase" letterSpacing="0.04em">
      {children}
    </Text>
  )
}

export function Note({ children }: { children: ReactNode }) {
  return <Text fontSize="xs" color="text.muted" textAlign="center" pt={2}>{children}</Text>
}

export function Loading({ label = 'Loading prices…' }: { label?: string }) {
  return (
    <Flex align="center" justify="center" gap={2} pt={2}>
      <Spinner size="xs" color="accent" />
      <Text fontSize="xs" color="text.muted">{label}</Text>
    </Flex>
  )
}

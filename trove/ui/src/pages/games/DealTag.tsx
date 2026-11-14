import { Badge, Flex, Text } from '@chakra-ui/react'
import type { Deal } from '@/services/gaming'
import { formatDay, listingLabel, shortPrice } from './prices'

/** "−40%  $19.99 · eShop · Switch 2", with the sale's end when known. */
export default function DealTag({ deal, size = 'sm' }: { deal: Deal; size?: 'sm' | 'xs' }) {
  const fontSize = size === 'sm' ? 'xs' : '10px'
  return (
    <Flex direction="column" gap={0.5} mb={1.5}>
      <Flex align="center" gap={1.5} wrap="wrap">
        <Badge colorPalette="green" variant="solid" fontSize={fontSize} px={1.5}>
          −{deal.discountPct ?? 0}%
        </Badge>
        <Text fontSize={fontSize} fontWeight="600" color="text.primary">
          {shortPrice(deal.finalMinor, deal.currency)}
        </Text>
        <Text fontSize={fontSize} color="text.secondary" truncate>{listingLabel(deal)}</Text>
      </Flex>
      {(deal.lowest || deal.saleEndsAt) && (
        <Text fontSize="10px" color="text.muted">
          {[deal.lowest && 'Lowest yet', deal.saleEndsAt && `until ${formatDay(deal.saleEndsAt)}`].filter(Boolean).join(' · ')}
        </Text>
      )}
    </Flex>
  )
}

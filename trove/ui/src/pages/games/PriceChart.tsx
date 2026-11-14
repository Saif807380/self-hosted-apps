import { Box, Text } from '@chakra-ui/react'
import {
  CartesianGrid, Legend, Line, LineChart, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis,
} from 'recharts'
import { useIsDark } from '@/hooks/useIsDark'
import { chartTheme } from '@/lib/chart-theme'
import type { ChartRow, ChartSeries } from './prices'
import { formatDay, formatRupees, seriesColor } from './prices'

interface PriceChartProps {
  series: ChartSeries[]
  rows: ChartRow[]
  targetRupees?: number
}

/**
 * Price history in rupees, one step line per store edition. Styled to match
 * the workout progress chart.
 */
export default function PriceChart({ series, rows, targetRupees }: PriceChartProps) {
  const isDark = useIsDark()
  const colors = chartTheme(isDark)

  // Two checks are the least that shows a trend; before that, say why the
  // chart is empty instead of drawing a lone dot.
  if (rows.length < 2) {
    return (
      <Box
        h="120px" display="flex" alignItems="center" justifyContent="center" px={4}
        border="1px dashed" borderColor="border.default" borderRadius="8px"
      >
        <Text fontSize="xs" color="text.muted" textAlign="center">
          Price history builds up as prices are checked every 6 hours.
        </Text>
      </Box>
    )
  }

  return (
    <Box h="220px">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={rows} margin={{ top: 5, right: 10, left: -4, bottom: 5 }}>
          <CartesianGrid strokeDasharray="3 3" stroke={colors.border} />
          <XAxis
            dataKey="t" type="number" scale="time" domain={['dataMin', 'dataMax']}
            tickFormatter={formatDay} tick={{ fontSize: 12, fill: colors.muted }}
            axisLine={{ stroke: colors.border }} tickLine={false} minTickGap={28}
          />
          <YAxis
            tickFormatter={formatRupees} width={64} domain={['auto', 'auto']}
            tick={{ fontSize: 12, fill: colors.muted }} axisLine={false} tickLine={false}
          />
          <Tooltip
            contentStyle={{
              background: colors.surface,
              border: `1px solid ${colors.border}`,
              borderRadius: '8px',
              fontSize: '0.8rem',
              color: colors.primary,
            }}
            labelStyle={{ color: colors.primary, fontWeight: 600 }}
            itemStyle={{ color: colors.secondary }}
            labelFormatter={t => formatDay(t as number)}
            formatter={value => formatRupees(value as number)}
          />
          {series.length > 1 && <Legend wrapperStyle={{ fontSize: '0.8rem', color: colors.secondary }} />}
          {targetRupees && (
            <ReferenceLine
              y={targetRupees} stroke={colors.muted} strokeDasharray="4 4" ifOverflow="extendDomain"
              label={{ value: 'Target', position: 'insideTopRight', fontSize: 11, fill: colors.secondary }}
            />
          )}
          {series.map(s => {
            const color = seriesColor(s.listing, isDark)
            return (
              <Line
                key={s.key} dataKey={s.key} name={s.label} type="stepAfter"
                stroke={color} strokeWidth={2} dot={{ r: 4, fill: color }}
                isAnimationActive={false}
              />
            )
          })}
        </LineChart>
      </ResponsiveContainer>
    </Box>
  )
}

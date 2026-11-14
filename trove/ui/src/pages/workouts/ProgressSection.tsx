import { useState } from 'react'
import { Box, Flex, Text } from '@chakra-ui/react'
import {
  ResponsiveContainer,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
} from 'recharts'
import type { Exercise, WorkoutLog } from '@/types/api'
import { useIsDark } from '@/hooks/useIsDark'
import { chartTheme } from '@/lib/chart-theme'

interface ProgressSectionProps {
  exercises: Exercise[]
  logs: WorkoutLog[]
  isMobile?: boolean
}

export default function ProgressSection({ exercises, logs, isMobile }: ProgressSectionProps) {
  const [selectedExerciseId, setSelectedExerciseId] = useState<string>(exercises[0]?.id ?? '')
  const isDark = useIsDark()

  const colors = chartTheme(isDark)

  const exercise = exercises.find(e => e.id === selectedExerciseId)
  const exerciseLogs = logs
    .filter(l => l.exerciseId === selectedExerciseId)
    .sort((a, b) => a.weekNumber - b.weekNumber)

  const chartData = exerciseLogs.map(l => ({
    week: `W${l.weekNumber}`,
    weight: l.weightKg ?? null,
    reps: l.reps ? (isNaN(Number(l.reps)) ? null : Number(l.reps)) : null,
    sets: l.sets ?? null,
  }))

  if (!exercises.length) {
    return (
      <Text fontSize="sm" color="text.muted" fontStyle="italic">
        Add exercises to see progress charts.
      </Text>
    )
  }

  return (
    <Box>
      <Flex gap={2} mb={4} wrap="wrap">
        {exercises.map(ex => {
          const active = selectedExerciseId === ex.id
          return (
            <button
              key={ex.id}
              type="button"
              onClick={() => setSelectedExerciseId(ex.id)}
              style={{
                padding: isMobile ? '6px 12px' : '3px 10px',
                borderRadius: '6px',
                fontSize: '0.8rem',
                fontWeight: 500,
                cursor: 'pointer',
                border: `1px solid ${active ? 'var(--chakra-colors-accent)' : 'var(--chakra-colors-border-default)'}`,
                background: active ? 'var(--chakra-colors-accent)' : 'var(--chakra-colors-bg-surface)',
                color: active ? 'white' : 'var(--chakra-colors-text-secondary)',
                transition: 'all 0.15s ease',
              }}
            >
              {ex.name}
            </button>
          )
        })}
      </Flex>

      {chartData.length < 2 ? (
        <Box
          h="200px"
          display="flex" alignItems="center" justifyContent="center"
          border="1px dashed"
          borderColor="border.default"
          borderRadius="8px"
        >
          <Text fontSize="sm" color="text.muted">
            {chartData.length === 0
              ? `No data logged for ${exercise?.name ?? 'this exercise'} yet.`
              : 'Log at least 2 weeks to see a trend.'}
          </Text>
        </Box>
      ) : (
        <Box h="260px">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={chartData} margin={{ top: 5, right: 10, left: -10, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" stroke={colors.border} />
              <XAxis
                dataKey="week"
                tick={{ fontSize: 12, fill: colors.muted }}
                axisLine={{ stroke: colors.border }}
                tickLine={false}
              />
              <YAxis
                tick={{ fontSize: 12, fill: colors.muted }}
                axisLine={false}
                tickLine={false}
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
              />
              <Legend
                wrapperStyle={{ fontSize: '0.8rem', color: colors.secondary }}
              />
              {chartData.some(d => d.weight != null) && (
                <Line
                  type="monotone"
                  dataKey="weight"
                  name="Weight (kg)"
                  stroke="#6366f1"
                  strokeWidth={2}
                  dot={{ r: 4, fill: '#6366f1' }}
                  connectNulls
                />
              )}
              {chartData.some(d => d.reps != null) && (
                <Line
                  type="monotone"
                  dataKey="reps"
                  name="Reps"
                  stroke="#10b981"
                  strokeWidth={2}
                  dot={{ r: 4, fill: '#10b981' }}
                  connectNulls
                />
              )}
              {chartData.some(d => d.sets != null) && (
                <Line
                  type="monotone"
                  dataKey="sets"
                  name="Sets"
                  stroke="#f59e0b"
                  strokeWidth={2}
                  dot={{ r: 4, fill: '#f59e0b' }}
                  connectNulls
                />
              )}
            </LineChart>
          </ResponsiveContainer>
        </Box>
      )}
    </Box>
  )
}

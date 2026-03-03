import { useState, useEffect, useRef, type CSSProperties } from 'react'
import { Box, Flex, Text } from '@chakra-ui/react'
import type { Exercise, WorkoutLog } from '@/types/api'

interface RowState {
  sets: string
  reps: string
  weightKg: string
}

interface WeekLogSectionProps {
  exercises: Exercise[]
  logs: WorkoutLog[]
  weekNumber: number
  onUpsertLog: (exerciseId: string, weekNumber: number, sets?: number, reps?: string, weightKg?: number) => Promise<void>
}

function getLogForExercise(logs: WorkoutLog[], exerciseId: string, weekNumber: number): WorkoutLog | undefined {
  return logs.find(l => l.exerciseId === exerciseId && l.weekNumber === weekNumber)
}

function getLatestLogBefore(logs: WorkoutLog[], exerciseId: string, weekNumber: number): WorkoutLog | undefined {
  return logs
    .filter(l => l.exerciseId === exerciseId && l.weekNumber < weekNumber)
    .sort((a, b) => b.weekNumber - a.weekNumber)[0]
}

function buildRowState(log?: WorkoutLog): RowState {
  return {
    sets: log?.sets != null ? String(log.sets) : '',
    reps: log?.reps ?? '',
    weightKg: log?.weightKg != null ? String(log.weightKg) : '',
  }
}

const INPUT_STYLE: CSSProperties = {
  width: '100%',
  padding: '4px 8px',
  borderRadius: '6px',
  fontSize: '0.8rem',
  border: '1px solid var(--chakra-colors-border-default)',
  background: 'var(--chakra-colors-bg-canvas)',
  color: 'var(--chakra-colors-text-primary)',
  outline: 'none',
}

export default function WeekLogSection({ exercises, logs, weekNumber, onUpsertLog }: WeekLogSectionProps) {
  const [rows, setRows] = useState<Record<string, RowState>>({})
  const prevLogsRef = useRef(logs)
  const prevWeekRef = useRef(weekNumber)

  // Sync row state when logs, exercises, or weekNumber change
  useEffect(() => {
    const weekChanged = prevWeekRef.current !== weekNumber
    prevWeekRef.current = weekNumber

    setRows(prev => {
      const next: Record<string, RowState> = {}
      for (const ex of exercises) {
        const log = getLogForExercise(logs, ex.id, weekNumber)
        if (weekChanged) {
          // Week changed — use this week's log if it exists, otherwise pre-populate from last week
          next[ex.id] = buildRowState(log ?? getLatestLogBefore(logs, ex.id, weekNumber))
        } else {
          // Same week — only reset if the log changed externally (e.g. after save)
          const prevLog = getLogForExercise(prevLogsRef.current, ex.id, weekNumber)
          const logChanged =
            log?.id !== prevLog?.id ||
            log?.sets !== prevLog?.sets ||
            log?.reps !== prevLog?.reps ||
            log?.weightKg !== prevLog?.weightKg
          next[ex.id] = logChanged ? buildRowState(log) : (prev[ex.id] ?? buildRowState(log))
        }
      }
      prevLogsRef.current = logs
      return next
    })
  }, [exercises, logs, weekNumber])

  // Save all exercises in the week that have at least one value
  const handleBlur = async () => {
    await Promise.all(exercises.map(async ex => {
      const row = rows[ex.id]
      if (!row) return
      const sets = row.sets ? parseInt(row.sets, 10) : undefined
      const reps = row.reps.trim() || undefined
      const weightKg = row.weightKg ? parseFloat(row.weightKg) : undefined
      if (sets == null && reps == null && weightKg == null) return
      await onUpsertLog(ex.id, weekNumber, sets, reps, weightKg)
    }))
  }

  const handleChange = (exerciseId: string, field: keyof RowState, value: string) => {
    setRows(prev => ({ ...prev, [exerciseId]: { ...prev[exerciseId], [field]: value } }))
  }

  if (!exercises.length) {
    return (
      <Text fontSize="sm" color="text.muted" fontStyle="italic">
        No exercises added yet.
      </Text>
    )
  }

  return (
    <Box overflowX="auto">
      <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.85rem' }}>
        <thead>
          <tr>
            {['Exercise', 'Sets', 'Reps / Duration', 'Weight (kg)'].map(h => (
              <th
                key={h}
                style={{
                  textAlign: h === 'Exercise' ? 'left' : 'center',
                  padding: '6px 10px',
                  fontSize: '0.75rem',
                  fontWeight: 600,
                  color: 'var(--chakra-colors-text-muted)',
                  borderBottom: '1px solid var(--chakra-colors-border-default)',
                  whiteSpace: 'nowrap',
                }}
              >
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {exercises.map((ex, i) => {
            const row = rows[ex.id] ?? { sets: '', reps: '', weightKg: '' }
            const isAlt = i % 2 === 1
            return (
              <tr key={ex.id} style={{ background: isAlt ? 'var(--chakra-colors-bg-subtle)' : 'transparent' }}>
                <td style={{ padding: '6px 10px', color: 'var(--chakra-colors-text-primary)', fontWeight: 500 }}>
                  {ex.name}
                </td>
                {(['sets', 'reps', 'weightKg'] as const).map(field => (
                  <td key={field} style={{ padding: '4px 8px', textAlign: 'center' }}>
                    <input
                      type={field === 'reps' ? 'text' : 'number'}
                      min={0}
                      step={field === 'weightKg' ? 0.5 : 1}
                      value={row[field]}
                      placeholder="—"
                      onChange={e => handleChange(ex.id, field, e.target.value)}
                      onBlur={handleBlur}
                      style={{ ...INPUT_STYLE, width: field === 'reps' ? '80px' : '60px' }}
                    />
                  </td>
                ))}
              </tr>
            )
          })}
        </tbody>
      </table>
    </Box>
  )
}

interface WeekHistoryTableProps {
  exercises: Exercise[]
  logs: WorkoutLog[]
}

export function WeekHistoryTable({ exercises, logs }: WeekHistoryTableProps) {
  const weeks = [...new Set(logs.map(l => l.weekNumber))].sort((a, b) => b - a)

  if (!weeks.length) {
    return (
      <Text fontSize="sm" color="text.muted" fontStyle="italic">
        No workout data logged yet.
      </Text>
    )
  }

  const exMap = Object.fromEntries(exercises.map(e => [e.id, e]))

  return (
    <Box overflowX="auto">
      <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.85rem' }}>
        <thead>
          <tr>
            {['Week', 'Exercise', 'Sets', 'Reps / Duration', 'Weight (kg)'].map(h => (
              <th
                key={h}
                style={{
                  textAlign: h === 'Week' || h === 'Exercise' ? 'left' : 'center',
                  padding: '6px 10px',
                  fontSize: '0.75rem',
                  fontWeight: 600,
                  color: 'var(--chakra-colors-text-muted)',
                  borderBottom: '1px solid var(--chakra-colors-border-default)',
                  whiteSpace: 'nowrap',
                }}
              >
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {weeks.flatMap(week => {
            const weekLogs = logs.filter(l => l.weekNumber === week)
            return weekLogs.map((log, i) => (
              <tr key={log.id} style={{ background: week % 2 === 0 ? 'var(--chakra-colors-bg-subtle)' : 'transparent' }}>
                <td style={{ padding: '5px 10px', fontWeight: 600, color: 'var(--chakra-colors-text-secondary)', whiteSpace: 'nowrap' }}>
                  {i === 0 ? `Week ${week}` : ''}
                </td>
                <td style={{ padding: '5px 10px', color: 'var(--chakra-colors-text-primary)' }}>
                  {exMap[log.exerciseId]?.name ?? '—'}
                </td>
                <td style={{ padding: '5px 10px', textAlign: 'center', color: 'var(--chakra-colors-text-secondary)' }}>
                  {log.sets ?? '—'}
                </td>
                <td style={{ padding: '5px 10px', textAlign: 'center', color: 'var(--chakra-colors-text-secondary)' }}>
                  {log.reps ?? '—'}
                </td>
                <td style={{ padding: '5px 10px', textAlign: 'center', color: 'var(--chakra-colors-text-secondary)' }}>
                  {log.weightKg != null ? log.weightKg : '—'}
                </td>
              </tr>
            ))
          })}
        </tbody>
      </table>
    </Box>
  )
}

interface WeekSelectorProps {
  logs: WorkoutLog[]
  selectedWeek: number
  onSelect: (week: number) => void
  onDeleteWeek: (week: number) => void
}

export function WeekSelector({ logs, selectedWeek, onSelect, onDeleteWeek }: WeekSelectorProps) {
  const existingWeeks = [...new Set(logs.map(l => l.weekNumber))].sort((a, b) => a - b)
  const maxWeek = existingWeeks.length ? Math.max(...existingWeeks) : 0
  const nextWeek = maxWeek + 1

  const weeks = existingWeeks.includes(selectedWeek)
    ? existingWeeks
    : [...existingWeeks, selectedWeek].sort((a, b) => a - b)

  const isSelected = (w: number) => selectedWeek === w
  const isExisting = (w: number) => existingWeeks.includes(w)

  return (
    <Flex gap={2} align="center" wrap="wrap">
      {weeks.map(w => (
        <Flex
          key={w}
          align="center"
          borderRadius="6px"
          overflow="hidden"
          border="1px solid"
          borderColor={isSelected(w) ? 'accent' : 'border.default'}
          style={{ transition: 'border-color 0.15s ease' }}
        >
          <button
            type="button"
            onClick={() => onSelect(w)}
            style={{
              padding: '3px 10px',
              fontSize: '0.8rem',
              fontWeight: 600,
              cursor: 'pointer',
              border: 'none',
              background: isSelected(w)
                ? 'var(--chakra-colors-accent)'
                : 'var(--chakra-colors-bg-surface)',
              color: isSelected(w)
                ? 'white'
                : 'var(--chakra-colors-text-secondary)',
              transition: 'all 0.15s ease',
            }}
          >
            Week {w}
          </button>
          {isExisting(w) && (
            <button
              type="button"
              onClick={() => onDeleteWeek(w)}
              title={`Delete week ${w}`}
              style={{
                padding: '0 6px',
                cursor: 'pointer',
                border: 'none',
                borderLeft: '1px solid var(--chakra-colors-border-default)',
                background: isSelected(w)
                  ? 'var(--chakra-colors-accent)'
                  : 'var(--chakra-colors-bg-surface)',
                color: isSelected(w) ? 'rgba(255,255,255,0.7)' : 'var(--chakra-colors-text-muted)',
                transition: 'all 0.15s ease',
                display: 'flex',
                alignItems: 'center',
                alignSelf: 'stretch',
              }}
            >
              ×
            </button>
          )}
        </Flex>
      ))}
      {!existingWeeks.includes(nextWeek) && nextWeek !== selectedWeek && (
        <button
          type="button"
          onClick={() => onSelect(nextWeek)}
          style={{
            padding: '3px 10px',
            borderRadius: '6px',
            fontSize: '0.8rem',
            fontWeight: 500,
            cursor: 'pointer',
            border: '1px dashed var(--chakra-colors-border-default)',
            background: 'transparent',
            color: 'var(--chakra-colors-text-muted)',
            transition: 'all 0.15s ease',
          }}
        >
          + Week {nextWeek}
        </button>
      )}
    </Flex>
  )
}

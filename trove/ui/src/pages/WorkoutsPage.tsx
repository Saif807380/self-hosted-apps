import { useState, useEffect, useMemo, useRef, useCallback, type ReactNode } from 'react'
import { Box, Flex, Heading, Text, Spinner } from '@chakra-ui/react'
import { useWorkouts } from '@/hooks/useWorkouts'
import { toaster } from '@/lib/toaster'
import type { WorkoutType } from '@/types/api'
import WeekLogSection, { WeekHistoryTable, WeekSelector } from './workouts/WeekLogSection'
import ProgressSection from './workouts/ProgressSection'

const SIDEBAR_WIDTH = 220

function SectionHeading({ children }: { children: ReactNode }) {
  return (
    <Text
      fontSize="xs" fontWeight="700" letterSpacing="0.08em" textTransform="uppercase"
      color="text.muted" mb={3}
    >
      {children}
    </Text>
  )
}

function AddItemRow({ placeholder, onAdd }: { placeholder: string; onAdd: (name: string) => Promise<void> }) {
  const [value, setValue] = useState('')
  const [saving, setSaving] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const handleSubmit = useCallback(async () => {
    const name = value.trim()
    if (!name) return
    setSaving(true)
    try {
      await onAdd(name)
      setValue('')
    } finally {
      setSaving(false)
      setTimeout(() => inputRef.current?.focus(), 0)
    }
  }, [value, onAdd])

  return (
    <Flex gap={1} mt={2}>
      <input
        ref={inputRef}
        type="text"
        value={value}
        placeholder={placeholder}
        disabled={saving}
        onChange={e => setValue(e.target.value)}
        onKeyDown={e => { if (e.key === 'Enter') handleSubmit() }}
        style={{
          flex: 1,
          minWidth: 0,
          padding: '4px 8px',
          borderRadius: '6px',
          fontSize: '0.8rem',
          border: '1px solid var(--chakra-colors-border-default)',
          background: 'var(--chakra-colors-bg-canvas)',
          color: 'var(--chakra-colors-text-primary)',
          outline: 'none',
        }}
      />
      <button
        type="button"
        disabled={saving || !value.trim()}
        onClick={handleSubmit}
        style={{
          padding: '4px 10px',
          borderRadius: '6px',
          fontSize: '0.8rem',
          fontWeight: 600,
          cursor: saving || !value.trim() ? 'not-allowed' : 'pointer',
          border: '1px solid var(--chakra-colors-accent)',
          background: 'var(--chakra-colors-accent)',
          color: 'white',
          opacity: saving || !value.trim() ? 0.5 : 1,
          transition: 'opacity 0.15s ease',
        }}
      >
        Add
      </button>
    </Flex>
  )
}

export default function WorkoutsPage() {
  const {
    types, logs, loading, error,
    fetchLogsForType,
    createType, deleteType,
    createExercise, deleteExercise,
    upsertLog, deleteWeekLogs,
  } = useWorkouts()

  const [selectedTypeId, setSelectedTypeId] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<'log' | 'history' | 'progress'>('log')
  const [selectedWeek, setSelectedWeek] = useState(1)

  const selectedType: WorkoutType | undefined = types.find(t => t.id === selectedTypeId)

  // Auto-select first type
  useEffect(() => {
    if (!selectedTypeId && types.length > 0) {
      setSelectedTypeId(types[0].id)
    }
  }, [types, selectedTypeId])

  // Fetch logs when selected type (or its exercises) changes.
  // Auto-select the latest week only when the type itself switches, not on log saves.
  const prevTypeIdRef = useRef<string | null>(null)
  useEffect(() => {
    if (!selectedTypeId) return
    const typeChanged = selectedTypeId !== prevTypeIdRef.current
    fetchLogsForType(selectedTypeId, types).then(fetched => {
      if (typeChanged) {
        prevTypeIdRef.current = selectedTypeId
        const weeks = [...new Set(fetched.map(l => l.weekNumber))]
        setSelectedWeek(weeks.length ? Math.max(...weeks) : 1)
      }
    })
  }, [selectedTypeId, types, fetchLogsForType])

  const handleSelectType = (id: string) => {
    setSelectedTypeId(id)
    setActiveTab('log')
  }

  const handleCreateType = async (name: string) => {
    const newType = await createType(name)
    setSelectedTypeId(newType.id)
    toaster.create({ title: 'Workout type added', type: 'success', duration: 3000 })
  }

  const handleDeleteType = async (type: WorkoutType) => {
    await deleteType(type.id)
    toaster.create({ title: 'Workout type deleted', type: 'success', duration: 3000 })
    setSelectedTypeId(null)
  }

  const handleCreateExercise = async (name: string) => {
    if (!selectedTypeId) return
    await createExercise(selectedTypeId, name)
    // useEffect watching `types` will re-fetch logs automatically
  }

  const handleDeleteExercise = async (typeId: string, exerciseId: string) => {
    await deleteExercise(typeId, exerciseId)
  }

  const handleUpsertLog = async (exerciseId: string, weekNumber: number, sets?: number, reps?: string, weightKg?: number) => {
    try {
      await upsertLog(exerciseId, weekNumber, sets, reps, weightKg)
    } catch {
      toaster.create({ title: 'Failed to save log', type: 'error', duration: 3000 })
    }
  }

  const handleDeleteWeek = async (weekNumber: number) => {
    await deleteWeekLogs(weekNumber)
    toaster.create({ title: `Week ${weekNumber} deleted`, type: 'success', duration: 3000 })
    if (selectedWeek === weekNumber) {
      const remaining = [...new Set(
        logs.filter(l => l.weekNumber !== weekNumber).map(l => l.weekNumber)
      )].sort((a, b) => b - a)
      setSelectedWeek(remaining[0] ?? 1)
    }
  }

  const TABS = [
    { id: 'log' as const, label: 'Log Entry' },
    { id: 'history' as const, label: 'History' },
    { id: 'progress' as const, label: 'Progress' },
  ]

  const exercises = useMemo(() => selectedType?.exercises ?? [], [selectedType])

  return (
    <Box>
      <Flex align="center" justify="space-between" mb={5}>
        <Box>
          <Heading
            fontFamily="heading" fontWeight="700" fontSize="2xl"
            color="text.primary" letterSpacing="-0.02em"
          >
            🏋️ Workouts
          </Heading>
          {!loading && (
            <Text fontSize="sm" color="text.muted" mt={0.5}>
              {types.length} {types.length === 1 ? 'workout type' : 'workout types'}
            </Text>
          )}
        </Box>
      </Flex>

      {loading ? (
        <Flex justify="center" py={24}><Spinner color="accent" size="lg" /></Flex>
      ) : error ? (
        <Flex justify="center" py={16}><Text color="red.500" fontSize="sm">{error}</Text></Flex>
      ) : (
        <Flex gap={6} align="flex-start">
          {/* Sidebar — workout types */}
          <Box
            w={`${SIDEBAR_WIDTH}px`}
            flexShrink={0}
            bg="bg.surface"
            border="1px solid"
            borderColor="border.default"
            borderRadius="10px"
            p={4}
            position="sticky"
            top="72px"
          >
            <SectionHeading>Workout Types</SectionHeading>

            <Flex direction="column" gap={1}>
              {types.map(type => (
                <Flex
                  key={type.id}
                  align="center"
                  justify="space-between"
                  px={2} py={1.5}
                  borderRadius="6px"
                  cursor="pointer"
                  bg={selectedTypeId === type.id ? 'bg.subtle' : 'transparent'}
                  _hover={{ bg: 'bg.subtle' }}
                  onClick={() => handleSelectType(type.id)}
                >
                  <Text
                    fontSize="sm"
                    fontWeight={selectedTypeId === type.id ? 600 : 400}
                    color={selectedTypeId === type.id ? 'text.primary' : 'text.secondary'}
                    style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
                  >
                    {type.name}
                  </Text>
                  <button
                    type="button"
                    onClick={e => { e.stopPropagation(); handleDeleteType(type) }}
                    title="Delete type"
                    style={{
                      background: 'none', border: 'none', cursor: 'pointer',
                      color: 'var(--chakra-colors-text-muted)',
                      fontSize: '0.9rem', lineHeight: 1, padding: '2px 4px', borderRadius: '4px',
                      flexShrink: 0,
                    }}
                  >
                    ×
                  </button>
                </Flex>
              ))}
            </Flex>

            <Box mt={3} pt={3} borderTop="1px solid" borderColor="border.default">
              <AddItemRow placeholder="New type name…" onAdd={handleCreateType} />
            </Box>
          </Box>

          {/* Main content */}
          <Box flex={1} minW={0}>
            {!selectedType ? (
              <Flex
                h="300px" align="center" justify="center" direction="column" gap={2}
                border="1px dashed" borderColor="border.default" borderRadius="10px"
              >
                <Text fontSize="2xl">🏋️</Text>
                <Text fontSize="sm" color="text.muted">
                  {types.length === 0
                    ? 'Add a workout type to get started.'
                    : 'Select a workout type from the sidebar.'}
                </Text>
              </Flex>
            ) : (
              <Box>
                {/* Type header */}
                <Box mb={5}>
                  <Heading
                    fontFamily="heading" fontWeight="700" fontSize="xl"
                    color="text.primary" letterSpacing="-0.01em"
                  >
                    {selectedType.name}
                  </Heading>
                </Box>

                {/* Exercises section */}
                <Box
                  bg="bg.surface" border="1px solid" borderColor="border.default"
                  borderRadius="10px" p={4} mb={4}
                >
                  <SectionHeading>Exercises</SectionHeading>

                  {exercises.length === 0 ? (
                    <Text fontSize="sm" color="text.muted" fontStyle="italic" mb={3}>
                      No exercises yet. Add one below.
                    </Text>
                  ) : (
                    <Flex direction="column" gap={1} mb={3}>
                      {exercises.map(ex => (
                        <Flex
                          key={ex.id}
                          align="center" justify="space-between"
                          px={2} py={1.5}
                          borderRadius="6px"
                          _hover={{ bg: 'bg.subtle' }}
                        >
                          <Text fontSize="sm" color="text.primary">{ex.name}</Text>
                          <button
                            type="button"
                            onClick={() => handleDeleteExercise(selectedType.id, ex.id)}
                            title="Remove exercise"
                            style={{
                              background: 'none', border: 'none', cursor: 'pointer',
                              color: 'var(--chakra-colors-text-muted)',
                              fontSize: '0.9rem', lineHeight: 1, padding: '2px 4px', borderRadius: '4px',
                            }}
                          >
                            ×
                          </button>
                        </Flex>
                      ))}
                    </Flex>
                  )}

                  <AddItemRow placeholder="New exercise name…" onAdd={handleCreateExercise} />
                </Box>

                {/* Tab bar */}
                <Flex
                  gap={0} mb={4}
                  border="1px solid" borderColor="border.default"
                  borderRadius="8px" overflow="hidden"
                  display="inline-flex"
                >
                  {TABS.map(tab => (
                    <button
                      key={tab.id}
                      type="button"
                      onClick={() => setActiveTab(tab.id)}
                      style={{
                        padding: '6px 16px',
                        fontSize: '0.82rem',
                        fontWeight: 500,
                        cursor: 'pointer',
                        border: 'none',
                        borderRight: '1px solid var(--chakra-colors-border-default)',
                        background: activeTab === tab.id
                          ? 'var(--chakra-colors-bg-subtle)'
                          : 'var(--chakra-colors-bg-surface)',
                        color: activeTab === tab.id
                          ? 'var(--chakra-colors-text-primary)'
                          : 'var(--chakra-colors-text-secondary)',
                        transition: 'all 0.15s ease',
                      }}
                    >
                      {tab.label}
                    </button>
                  ))}
                </Flex>

                {/* Tab content */}
                <Box
                  bg="bg.surface" border="1px solid" borderColor="border.default"
                  borderRadius="10px" p={4}
                >
                  {activeTab === 'log' && (
                    <Box>
                      <Flex align="center" justify="space-between" mb={4}>
                        <SectionHeading>Week</SectionHeading>
                      </Flex>
                      <Box mb={4}>
                        <WeekSelector
                          logs={logs}
                          selectedWeek={selectedWeek}
                          onSelect={setSelectedWeek}
                          onDeleteWeek={handleDeleteWeek}
                        />
                      </Box>
                      <WeekLogSection
                        exercises={exercises}
                        logs={logs}
                        weekNumber={selectedWeek}
                        onUpsertLog={handleUpsertLog}
                      />
                    </Box>
                  )}

                  {activeTab === 'history' && (
                    <Box>
                      <SectionHeading>All Weeks</SectionHeading>
                      <WeekHistoryTable exercises={exercises} logs={logs} />
                    </Box>
                  )}

                  {activeTab === 'progress' && (
                    <Box>
                      <SectionHeading>Exercise Progress</SectionHeading>
                      <ProgressSection exercises={exercises} logs={logs} />
                    </Box>
                  )}
                </Box>
              </Box>
            )}
          </Box>
        </Flex>
      )}
    </Box>
  )
}

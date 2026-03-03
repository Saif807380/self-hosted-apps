type View = 'grid' | 'list'

interface ViewToggleProps {
  view: View
  onChange: (v: View) => void
}

const GridIcon = () => (
  <svg width="15" height="15" viewBox="0 0 16 16" fill="currentColor">
    <rect x="1" y="1" width="6" height="6" rx="1" />
    <rect x="9" y="1" width="6" height="6" rx="1" />
    <rect x="1" y="9" width="6" height="6" rx="1" />
    <rect x="9" y="9" width="6" height="6" rx="1" />
  </svg>
)

const ListIcon = () => (
  <svg width="15" height="15" viewBox="0 0 16 16" fill="currentColor">
    <rect x="1" y="2" width="14" height="2.5" rx="1" />
    <rect x="1" y="6.75" width="14" height="2.5" rx="1" />
    <rect x="1" y="11.5" width="14" height="2.5" rx="1" />
  </svg>
)

const BUTTONS = [
  { value: 'grid' as View, Icon: GridIcon, label: 'Grid view' },
  { value: 'list' as View, Icon: ListIcon, label: 'List view' },
]

export default function ViewToggle({ view, onChange }: ViewToggleProps) {
  return (
    <div
      style={{
        display: 'flex',
        border: '1px solid var(--chakra-colors-border-default)',
        borderRadius: '7px',
        overflow: 'hidden',
      }}
    >
      {BUTTONS.map(({ value, Icon, label }) => (
        <button
          key={value}
          type="button"
          aria-label={label}
          onClick={() => onChange(value)}
          style={{
            padding: '5px 9px',
            cursor: 'pointer',
            border: 'none',
            borderRight: value === 'grid' ? '1px solid var(--chakra-colors-border-default)' : 'none',
            background: view === value
              ? 'var(--chakra-colors-bg-subtle)'
              : 'var(--chakra-colors-bg-surface)',
            color: view === value
              ? 'var(--chakra-colors-text-primary)'
              : 'var(--chakra-colors-text-muted)',
            display: 'flex',
            alignItems: 'center',
            transition: 'background 0.15s ease, color 0.15s ease',
          }}
        >
          <Icon />
        </button>
      ))}
    </div>
  )
}

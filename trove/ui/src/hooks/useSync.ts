import { useContext } from 'react'
import { SyncContext } from '@/contexts/SyncContext'

export function useSync() {
  return useContext(SyncContext)
}

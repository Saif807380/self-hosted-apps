import type { VideoGame, VideoGameFormData } from '@/types/api'
import { rpc } from './client'

const SVC = 'trove.v1.games.GameService'

export const gamesApi = {
  list: (params: { search?: string; yearPlayed?: number } = {}): Promise<VideoGame[]> => {
    const body: Record<string, unknown> = {}
    if (params.search) body.search = params.search
    if (params.yearPlayed) body.yearPlayed = params.yearPlayed
    return rpc<{ games?: VideoGame[] }>(SVC, 'ListGames', body).then(r => r.games ?? [])
  },

  create: (data: VideoGameFormData): Promise<VideoGame> =>
    rpc<{ game: VideoGame }>(SVC, 'CreateGame', data).then(r => r.game),

  update: (id: string, data: VideoGameFormData): Promise<VideoGame> =>
    rpc<{ game: VideoGame }>(SVC, 'UpdateGame', { id, ...data }).then(r => r.game),

  delete: (id: string): Promise<void> =>
    rpc<Record<string, never>>(SVC, 'DeleteGame', { id }).then(() => undefined),
}

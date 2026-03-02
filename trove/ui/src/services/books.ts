import type { Book, BookFormData, Tag, Collection } from '@/types/api'
import { rpc } from './client'

const SVC = 'trove.v1.books.BookService'

export const booksApi = {
  list: (params: { search?: string; tagId?: string; yearRead?: number } = {}): Promise<Book[]> => {
    const body: Record<string, unknown> = {}
    if (params.search) body.search = params.search
    if (params.tagId) body.tagId = params.tagId
    if (params.yearRead) body.yearRead = params.yearRead
    return rpc<{ books?: Book[] }>(SVC, 'ListBooks', body).then(r => r.books ?? [])
  },

  get: (id: string): Promise<Book> =>
    rpc<{ book: Book }>(SVC, 'GetBook', { id }).then(r => r.book),

  create: (data: BookFormData): Promise<Book> =>
    rpc<{ book: Book }>(SVC, 'CreateBook', data).then(r => r.book),

  update: (id: string, data: BookFormData): Promise<Book> =>
    rpc<{ book: Book }>(SVC, 'UpdateBook', { id, ...data }).then(r => r.book),

  delete: (id: string): Promise<void> =>
    rpc<Record<string, never>>(SVC, 'DeleteBook', { id }).then(() => undefined),

  listTags: (): Promise<Tag[]> =>
    rpc<{ tags?: Tag[] }>(SVC, 'ListTags', {}).then(r => r.tags ?? []),

  createTag: (name: string): Promise<Tag> =>
    rpc<{ tag: Tag }>(SVC, 'CreateTag', { name }).then(r => r.tag),

  listCollections: (): Promise<Collection[]> =>
    rpc<{ collections?: Collection[] }>(SVC, 'ListCollections', {}).then(r => r.collections ?? []),

  createCollection: (name: string): Promise<Collection> =>
    rpc<{ collection: Collection }>(SVC, 'CreateCollection', { name }).then(r => r.collection),

  addToCollection: (collectionId: string, bookId: string): Promise<void> =>
    rpc<Record<string, never>>(SVC, 'AddBookToCollection', { collectionId, bookId }).then(() => undefined),

  removeFromCollection: (collectionId: string, bookId: string): Promise<void> =>
    rpc<Record<string, never>>(SVC, 'RemoveBookFromCollection', { collectionId, bookId }).then(() => undefined),
}

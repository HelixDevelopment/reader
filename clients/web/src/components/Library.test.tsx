import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { Api, Page, Work } from '../lib/api'
import { ApiError } from '../lib/api'
import { Library } from './Library'

const work = (id: string, title: string, author = 'Ann Author'): Work => ({
  id, canonical_title: title, created_at: '', updated_at: '',
  editions: [{ id: id + '-e', title, authors: [author], identifiers: [], files: [{ id: id + '-f', format: 'txt', mime_type: 'text/plain', size: 10, sha256: 'x', protected: false }], identification_state: 'identified', identification_confidence: 1 }],
})
const page = (works: Work[], extra: Partial<Page> = {}): Page => ({ works, total: works.length, offset: 0, limit: 20, has_next: false, ...extra })
const apiWith = (listWorks: Api['listWorks']): Api => ({ listWorks, getWork: vi.fn(), cover: vi.fn(async () => { throw new ApiError(404, 'cover not found') }), fileContent: vi.fn() })

describe('Library', () => {
  it('shows a loading state, then the works from the API', async () => {
    const listWorks = vi.fn(async () => page([work('1', 'The Glass Meridian'), work('2', 'Orbital Gardens')]))
    const api = apiWith(listWorks)
    render(<Library api={api} onOpen={() => {}} />)
    expect(screen.getByRole('status')).toHaveTextContent(/loading/i)
    expect(await screen.findByRole('heading', { name: 'The Glass Meridian' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Orbital Gardens' })).toBeInTheDocument()
    expect(screen.getAllByRole('listitem')).toHaveLength(2)
    // The API reports cover_href; with none there is no cover to request (avoids a 404 per card).
    expect(api.cover).not.toHaveBeenCalled()
  })

  it('shows an honest empty state', async () => {
    render(<Library api={apiWith(async () => page([]))} onOpen={() => {}} />)
    expect(await screen.findByText(/no works in this library/i)).toBeInTheDocument()
  })

  it('shows an error with a retry that refetches', async () => {
    const listWorks = vi.fn<Api['listWorks']>()
      .mockRejectedValueOnce(new ApiError(0, 'Cannot reach the server'))
      .mockResolvedValueOnce(page([work('1', 'Recovered')]))
    render(<Library api={apiWith(listWorks)} onOpen={() => {}} />)
    expect(await screen.findByRole('alert')).toHaveTextContent('Cannot reach the server')
    await userEvent.click(screen.getByRole('button', { name: /retry/i }))
    expect(await screen.findByRole('heading', { name: 'Recovered' })).toBeInTheDocument()
  })

  it('passes the search text to the API and opens a work', async () => {
    const listWorks = vi.fn(async () => page([work('1', 'Needle')]))
    const onOpen = vi.fn()
    render(<Library api={apiWith(listWorks)} onOpen={onOpen} />)
    await screen.findByRole('heading', { name: 'Needle' })
    await userEvent.type(screen.getByRole('searchbox', { name: /search library/i }), 'nee')
    await waitFor(() => expect(listWorks).toHaveBeenLastCalledWith(expect.objectContaining({ q: 'nee' })))
    await userEvent.click(screen.getByRole('button', { name: /open needle/i }))
    expect(onOpen).toHaveBeenCalledWith('1')
  })
})

import { render, screen } from '@testing-library/react'
import type { Api, File } from '../lib/api'
import { ApiError } from '../lib/api'
import { Reader } from './Reader'

const file = (over: Partial<File>): File => ({ id: 'f1', format: 'txt', mime_type: 'text/plain', size: 5, sha256: 'x', protected: false, ...over })
const apiWith = (fileContent: Api['fileContent']): Api => ({ listWorks: vi.fn(), getWork: vi.fn(), cover: vi.fn(), fileContent })

describe('Reader', () => {
  it('renders plain text content fetched from the content route', async () => {
    const fileContent = vi.fn(async () => ({ blob: new Blob(['First line.\n\nSecond paragraph.']), mime: 'text/plain' }))
    render(<Reader api={apiWith(fileContent)} file={file({})} title="Notes" />)
    expect(await screen.findByText('Second paragraph.')).toBeInTheDocument()
    expect(fileContent).toHaveBeenCalledWith('f1')
    expect(screen.getByRole('article', { name: 'Notes' })).toBeInTheDocument()
  })

  it('says honestly that EPUB cannot be rendered yet and offers the file', async () => {
    const fileContent = vi.fn(async () => ({ blob: new Blob(['PK']), mime: 'application/epub+zip' }))
    render(<Reader api={apiWith(fileContent)} file={file({ format: 'epub', mime_type: 'application/epub+zip' })} title="Book" />)
    expect(await screen.findByText(/does not render epub/i)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /download original/i })).toHaveAttribute('download')
  })

  it('does not fetch DRM-protected files and says so', () => {
    const fileContent = vi.fn()
    render(<Reader api={apiWith(fileContent)} file={file({ protected: true, drm_scheme: 'lcp' })} title="Locked" />)
    expect(screen.getByText(/protected/i)).toBeInTheDocument()
    expect(fileContent).not.toHaveBeenCalled()
  })

  it('shows the server error when the content route fails', async () => {
    render(<Reader api={apiWith(async () => { throw new ApiError(409, 'file source is not connected') })} file={file({})} title="X" />)
    expect(await screen.findByRole('alert')).toHaveTextContent('file source is not connected')
  })
})

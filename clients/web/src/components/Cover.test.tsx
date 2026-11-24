import { render, screen } from '@testing-library/react'
import type { Api } from '../lib/api'
import { Cover } from './Cover'

const apiWith = (cover: Api['cover']): Api => ({ listWorks: vi.fn(), getWork: vi.fn(), fileContent: vi.fn(), cover })

describe('Cover', () => {
  it('fetches the cover blob when the edition has one and shows an image', async () => {
    const cover = vi.fn(async () => new Blob(['png']))
    render(<Cover api={apiWith(cover)} editionId="e1" hasCover title="Book" />)
    expect(await screen.findByRole('img', { name: 'Cover of Book' })).toHaveAttribute('src', expect.stringContaining('blob:'))
    expect(cover).toHaveBeenCalledWith('e1')
  })

  it('shows a labelled placeholder and never requests a missing cover', () => {
    const cover = vi.fn()
    render(<Cover api={apiWith(cover)} editionId="e1" hasCover={false} title="Book" />)
    expect(screen.getByRole('img', { name: 'No cover for Book' })).toBeInTheDocument()
    expect(cover).not.toHaveBeenCalled()
  })
})

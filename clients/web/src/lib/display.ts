import type { Edition, Work } from './api'

export const primaryEdition = (w: Work): Edition | undefined => w.editions[0]
export const authorsOf = (w: Work): string => primaryEdition(w)?.authors.join(', ') || 'Unknown author'
export function formatSize(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(1)} MB`
}

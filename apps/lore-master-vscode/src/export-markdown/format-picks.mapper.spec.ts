import { formatPicks, textFormats } from './format-picks.mapper'

const formats = [
  { id: 'html', label: 'HTML', extension: '.html' },
  { id: 'docx', label: 'Word', extension: '.docx', binary: true },
]

describe('formatPicks', () => {
  it('shows the extension and ticks the preferred formats', () => {
    expect(formatPicks(formats, ['docx'])).toEqual([
      { id: 'html', label: 'HTML', description: '.html', picked: false },
      { id: 'docx', label: 'Word', description: '.docx', picked: true },
    ])
  })
})

describe('textFormats', () => {
  it('leaves out what cannot be copied', () => {
    expect(textFormats(formats).map(format => format.id)).toEqual(['html'])
  })
})

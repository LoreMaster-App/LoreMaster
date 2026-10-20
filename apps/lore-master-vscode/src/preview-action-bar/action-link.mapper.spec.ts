import { exportLink, parseActionLink } from './action-link.mapper'

const parts = (link: string): { scheme: string, path: string, query: string } => {
  const [scheme, rest] = link.split(/:(.*)/s, 2)
  const [path, query] = rest.split('?', 2)

  return { scheme, path, query: query ?? '' }
}

describe('the action links of the bar', () => {
  it('carry the document through the export link and back', () => {
    const document = 'file:///c%3A/repo/docs/a%20b.md'

    const link = parts(exportLink(document))

    expect(parseActionLink(link)).toEqual({ action: 'export', document })
  })

  it('ignore any other link', () => {
    expect(parseActionLink(parts('https://example.com/export?document=x'))).toBeUndefined()
    expect(parseActionLink(parts('loremaster-action:/other?document=x'))).toBeUndefined()
    expect(parseActionLink(parts('loremaster-action:/export'))).toBeUndefined()
  })
})

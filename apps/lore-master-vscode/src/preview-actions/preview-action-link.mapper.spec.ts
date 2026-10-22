import { parsePreviewActionLink, previewActionLink } from './preview-action-link.mapper'

const parts = (link: string): { scheme: string, path: string, query: string } => {
  const [scheme, rest] = link.split(/:(.*)/s, 2)
  const [path, query] = rest.split('?', 2)

  return { scheme, path, query: query ?? '' }
}

describe('the action links of the preview', () => {
  it('carry an action and its parameters through the link and back', () => {
    const params = { document: 'file:///c%3A/repo/docs/a%20b.md', line: '12' }
    const link = parts(previewActionLink('toggle-task', params))

    expect(parsePreviewActionLink(link)).toEqual({ action: 'toggle-task', params })
  })

  it('allow an action without parameters', () => {
    const link = parts(previewActionLink('ping', {}))

    expect(parsePreviewActionLink(link)).toEqual({ action: 'ping', params: {} })
  })

  it('ignore any other link', () => {
    expect(parsePreviewActionLink(parts('https://example.com/export?document=x'))).toBeUndefined()
    expect(parsePreviewActionLink(parts('loremaster-action:'))).toBeUndefined()
  })
})

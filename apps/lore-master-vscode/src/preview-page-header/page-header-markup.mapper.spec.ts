import { pageHeaderMarkup } from './page-header-markup.mapper'

describe('pageHeaderMarkup', () => {
  it('shows the storage, the page title and the status of each row', () => {
    const html = pageHeaderMarkup([
      { storage: 'Docs', pageTitle: 'Team: Guide', status: 'local-changes' },
      { storage: 'Site', pageTitle: 'Guide' },
    ])

    expect(html).toContain('loremaster-chip-local-changes">Local changes')
    expect(html.match(/loremaster-page-header-row/g)).toHaveLength(2)
    expect(html.match(/loremaster-chip /g)).toHaveLength(1)
  })

  it('escapes what the files and settings say', () => {
    expect(pageHeaderMarkup([{ storage: 'A&B', pageTitle: '<script>"x"' }])).toContain('A&amp;B')
    expect(pageHeaderMarkup([{ storage: 'A', pageTitle: '<script>"x"' }])).toContain('&lt;script&gt;&quot;x&quot;')
  })
})

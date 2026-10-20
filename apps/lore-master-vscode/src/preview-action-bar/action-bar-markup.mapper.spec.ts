import { actionBarMarkup, TOGGLES } from './action-bar-markup.mapper'

describe('actionBarMarkup', () => {
  it('shows the storage, the page title and the status of each page', () => {
    const html = actionBarMarkup('file:///a.md', {
      state: 'tracked',
      pages: [{ storage: 'Docs', pageTitle: 'Team: Guide', status: 'local-changes' }, { storage: 'Site', pageTitle: 'Guide' }],
    })

    expect(html).toContain('loremaster-chip-local-changes">Local changes')
    expect(html.match(/loremaster-bar-page"/g)).toHaveLength(2)
    expect(html.match(/loremaster-chip /g)).toHaveLength(1)
  })

  it('always carries the Export link and a switch for each addition', () => {
    const html = actionBarMarkup('file:///a%20b.md', { state: 'checking' })

    expect(html).toContain('Checking the sync status')
    expect(html).toContain('href="loremaster-action:/export?document=file%3A%2F%2F%2Fa%2520b.md"')
    for (const toggle of TOGGLES) {
      expect(html).toContain(`data-toggle="${toggle.key}"`)
    }
  })

  it('says why there is no status', () => {
    expect(actionBarMarkup('file:///a.md', { state: 'untracked' })).toContain('No storage syncs this file')
    expect(actionBarMarkup('file:///a.md', { state: 'unavailable', reason: 'engine <down>' })).toContain('unavailable: engine &lt;down&gt;')
  })

  it('escapes what the files and settings say', () => {
    const html = actionBarMarkup('file:///a.md', { state: 'tracked', pages: [{ storage: 'A&B', pageTitle: '<script>"x"' }] })

    expect(html).toContain('A&amp;B')
    expect(html).toContain('&lt;script&gt;&quot;x&quot;')
  })
})

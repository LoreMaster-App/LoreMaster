import { renderStorageFormPage } from './storage-form-page.mapper'

describe('renderStorageFormPage', () => {
  const page = renderStorageFormPage('abc123', 'Add a storage')

  it('runs only scripts and styles under its nonce', () => {
    expect(page).toContain("script-src 'nonce-abc123'")
    expect(page).toContain("style-src 'nonce-abc123'")
    expect(page.match(/<script/g)).toHaveLength(1)
    expect(page).not.toContain('onclick=')
  })

  it('has a field for every setting of a storage, and starts with no platform chosen', () => {
    for (const id of ['platform', 'connection', 'space', 'parentPageId', 'titlePrefix', 'direction', 'mermaidMode', 'linkMode', 'titleCollision', 'repo', 'branch', 'path', 'roots', 'excludes', 'ignore']) {
      expect(page).toContain(`id="${id}"`)
    }
    expect(page).toContain('<option value="">Choose a storage…</option>')
    expect(page).not.toMatch(/<option[^>]*selected/)
  })
})

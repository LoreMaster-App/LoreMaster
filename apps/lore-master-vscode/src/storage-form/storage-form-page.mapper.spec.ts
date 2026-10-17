import { renderStorageFormPage } from './storage-form-page.mapper'

describe('renderStorageFormPage', () => {
  const page = renderStorageFormPage('abc123', 'Add a storage')

  it('runs only scripts and styles under its nonce', () => {
    expect(page).toContain("script-src 'nonce-abc123'")
    expect(page).toContain("style-src 'nonce-abc123'")
    expect(page.match(/<script/g)).toHaveLength(1)
    expect(page).not.toContain('onclick=')
  })

  it('knows no platform or setting: it builds the form from the description it is sent', () => {
    for (const word of ['confluence', 'github-pages', 'github-wiki', 'parentPageId', 'mermaidMode', 'titlePrefix', 'branch']) {
      expect(page).not.toContain(word)
    }
    expect(page).toContain('message.init.description')
  })

  it('starts with no platform chosen', () => {
    expect(page).toContain('Choose a storage…')
  })
})

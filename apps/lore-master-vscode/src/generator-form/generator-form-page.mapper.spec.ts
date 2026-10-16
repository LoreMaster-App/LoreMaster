import { renderGeneratorFormPage } from './generator-form-page.mapper'

describe('renderGeneratorFormPage', () => {
  const page = renderGeneratorFormPage('abc123', 'Add a generator')

  it('runs only scripts and styles under its nonce', () => {
    expect(page).toContain("script-src 'nonce-abc123'")
    expect(page).toContain("style-src 'nonce-abc123'")
    expect(page.match(/<script/g)).toHaveLength(1)
    expect(page).not.toContain('onclick=')
  })

  it('has a field for every setting of a generator', () => {
    for (const id of ['kind', 'output', 'reads', 'command', 'title']) {
      expect(page).toContain(`id="${id}"`)
    }
  })
})

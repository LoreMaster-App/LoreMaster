import { configureGitHubPages } from './configure-github-pages.use-case'

describe('configureGitHubPages', () => {
  it('builds a one-way github-pages output from the repo and branch prompts', async () => {
    const output = await configureGitHubPages({ promptRepo: () => Promise.resolve('owner/name'), promptBranch: () => Promise.resolve('docs') })

    expect(output).toMatchObject({ platform: 'github-pages', repo: 'owner/name', branch: 'docs', direction: 'to-platform' })
    expect(output?.content[0]).toEqual({ type: 'markdown', roots: ['.'], template: 'default' })
  })

  it('keeps the Confluence-only fields empty so the validator accepts it', async () => {
    const output = await configureGitHubPages({ promptRepo: () => Promise.resolve(''), promptBranch: () => Promise.resolve('') })

    expect(output).toMatchObject({ baseUrl: '', space: '', parentPageId: '', titlePrefix: '', repo: '', branch: '' })
  })

  it('returns undefined when a prompt is cancelled', async () => {
    expect(await configureGitHubPages({ promptRepo: () => Promise.resolve(undefined), promptBranch: () => Promise.resolve('x') })).toBeUndefined()
    expect(await configureGitHubPages({ promptRepo: () => Promise.resolve('o/n'), promptBranch: () => Promise.resolve(undefined) })).toBeUndefined()
  })
})

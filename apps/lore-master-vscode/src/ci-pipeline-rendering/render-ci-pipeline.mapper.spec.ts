import type { CiNeeds, CiProviderId } from './ci-pipeline.contract'
import { renderCiPipeline } from './render-ci-pipeline.mapper'

const confluence: CiNeeds = {
  version:            '1.2.3',
  command:            'sync --yes',
  secrets:            [{ name: 'CONFLUENCE_API_TOKEN', purpose: 'the API token' }, { name: 'CONFLUENCE_USER', purpose: 'the account' }],
  pushesToGit:        false,
  commitsAnnotations: true,
  prepareCommands:    ['npm ci'],
  settingsInRepo:     true,
  settingsJson:       '{}',
}
const pagesOnly: CiNeeds = { ...confluence, secrets: [], pushesToGit: true, commitsAnnotations: false, prepareCommands: [] }
const NEW_PROVIDERS: CiProviderId[] = ['gitlab-ci', 'bitbucket-pipelines', 'jenkins', 'circleci']

describe('every pipeline', () => {
  it.each(NEW_PROVIDERS)('%s pins the engine to the extension version, checks the download and runs the command', provider => {
    const file = renderCiPipeline(provider, confluence)

    expect(file).toContain('1.2.3')
    expect(file).toContain('sha256sum --check')
    expect(file).toContain('lore-master-vscode%40$LORE_MASTER_VERSION')
    expect(file).toContain('./lore-master-engine sync --yes')
    expect(file.endsWith('\n')).toBe(true)
    expect(file).not.toContain('\t')
  })

  it.each(NEW_PROVIDERS)('%s commits the sync state only when the sync writes it, with a message that stops a loop', provider => {
    const withState = renderCiPipeline(provider, confluence)
    const without = renderCiPipeline(provider, pagesOnly)

    expect(withState).toContain('git add -- lore-master.json || true')
    expect(withState).toContain('[skip ci]')
    expect(without).not.toContain('git commit')
  })

  it.each(NEW_PROVIDERS)('%s says where a generator\'s prepare command belongs, and nothing when there are none', provider => {
    expect(renderCiPipeline(provider, confluence)).toContain('npm ci')
    expect(renderCiPipeline(provider, pagesOnly)).not.toContain('prepare command')
  })

  it.each(NEW_PROVIDERS)('%s never writes a secret value', provider => {
    expect(renderCiPipeline(provider, confluence)).not.toMatch(/ey[a-z0-9]{20,}|password\s*[:=]\s*\S+/i)
  })
})

describe('GitLab CI/CD', () => {
  const file = renderCiPipeline('gitlab-ci', confluence)

  it('is its own file that the main one includes, and runs on a push to the default branch that touches Markdown', () => {
    expect(file).toContain('- local: .gitlab/lore-master.yml')
    expect(file).toContain('$CI_PIPELINE_SOURCE == "push" && $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH')
    expect(file).toContain("- '**/*.md'")
  })

  it('points the remote at a token only when the job writes', () => {
    expect(file).toContain('${LORE_MASTER_GIT_TOKEN}')
    expect(renderCiPipeline('gitlab-ci', { ...confluence, commitsAnnotations: false })).not.toContain('LORE_MASTER_GIT_TOKEN')
    expect(renderCiPipeline('gitlab-ci', pagesOnly)).toContain('LORE_MASTER_GIT_TOKEN')
  })

  it('takes its secrets from the CI variables of the same name, so it lists none', () => {
    expect(file).not.toContain('CONFLUENCE_API_TOKEN')
  })
})

describe('Bitbucket Pipelines', () => {
  it('runs a step on main when Markdown changes', () => {
    const file = renderCiPipeline('bitbucket-pipelines', confluence)

    expect(file).toContain('main:')
    expect(file).toContain('includePaths:')
    expect(file).toContain("- '**/*.md'")
    expect(file).toContain('copy this step into yours')
  })
})

describe('Jenkins', () => {
  const file = renderCiPipeline('jenkins', confluence)

  it('is a declarative pipeline that binds each secret as a credential of the same ID', () => {
    expect(file.startsWith('// LoreMaster')).toBe(true)
    expect(file).toContain('pipeline {')
    expect(file).toContain("CONFLUENCE_API_TOKEN = credentials('CONFLUENCE_API_TOKEN')")
    expect(file).toContain("CONFLUENCE_USER = credentials('CONFLUENCE_USER')")
  })

  it('works in a plain job and a multibranch one, only on main, and only when Markdown changed', () => {
    expect(file).toContain("env.BRANCH_NAME == null || env.BRANCH_NAME == 'main'")
    expect(file).toContain("changeset '**/*.md'")
  })

  it('leaves out the environment block when there are no secrets', () => {
    expect(renderCiPipeline('jenkins', pagesOnly)).not.toContain('credentials(')
  })

  it('keeps shell variables out of Groovy interpolation: every script is a single-quoted string', () => {
    expect(file).not.toMatch(/sh """/)
    expect(file.match(/sh '''/g)).toHaveLength(2)
  })
})

describe('CircleCI', () => {
  it('defines a job and a workflow that runs it on main', () => {
    const file = renderCiPipeline('circleci', confluence)

    expect(file).toContain('version: 2.1')
    expect(file).toContain('lore-master-docs:')
    expect(file).toContain('only: main')
    expect(file).toContain('copy the job and the workflow into it')
  })
})

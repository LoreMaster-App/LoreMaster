import type { Output, Settings } from '../engine-protocol'
import { planCiNeeds } from './plan-ci-needs.policy'

function output (overrides: Partial<Output>): Output {
  return {
    platform:       'confluence',
    baseUrl:        '',
    space:          '',
    parentPageId:   '',
    titlePrefix:    '',
    direction:      'to-platform',
    content:        [],
    mermaidMode:    '',
    titleCollision: '',
    linkMode:       '',
    ...overrides,
  }
}

function plan (settings: Partial<Settings>, settingsInRepo = true) {
  return planCiNeeds({ settings: { version: 1, outputs: [], ...settings }, version: '0.15.7', settingsInRepo })
}

describe('planCiNeeds', () => {
  it('asks a Confluence Cloud site for an email and a token', () => {
    const needs = plan({ outputs: [output({ baseUrl: 'https://acme.atlassian.net/wiki' })] })

    expect(needs.secrets.map(secret => secret.name)).toEqual(['LORE_MASTER_EMAIL', 'LORE_MASTER_TOKEN'])
    expect(needs.commitsAnnotations).toBe(true)
    expect(needs.pushesToGit).toBe(false)
  })

  it('asks a self-hosted site for a personal access token', () => {
    const needs = plan({ outputs: [output({ baseUrl: 'https://wiki.acme.example' })] })

    expect(needs.secrets.map(secret => secret.name)).toEqual(['LORE_MASTER_PAT'])
  })

  it('asks for each credential once when several outputs share an edition', () => {
    const needs = plan({ outputs: [output({ baseUrl: 'https://a.atlassian.net' }), output({ baseUrl: 'https://b.atlassian.net' })] })

    expect(needs.secrets).toHaveLength(2)
  })

  it('needs no secret for a GitHub Pages output, but write access to the repository', () => {
    const needs = plan({ outputs: [output({ platform: 'github-pages' })] })

    expect(needs.secrets).toEqual([])
    expect(needs.pushesToGit).toBe(true)
    expect(needs.commitsAnnotations).toBe(false)
  })

  it('adds --generate for generators and --prepare when one has a prepare command', () => {
    const outputs = [output({ platform: 'github-pages' })]

    expect(plan({ outputs, generators: [{ type: 'test-results', output: 'docs/tests' }] }).command).toBe('sync --generate --yes')
    expect(plan({ outputs, generators: [{ type: 'test-results', output: 'docs/tests', prepare: 'npm test' }] })).toMatchObject({
      command:         'sync --generate --prepare --yes',
      prepareCommands: ['npm test'],
    })
    expect(plan({ outputs }).command).toBe('sync --yes')
  })

  it('carries the settings as loreMaster.* keys for a repository that does not hold them', () => {
    const needs = plan({ outputs: [output({ platform: 'github-pages' })], ignore: ['drafts/'] }, false)

    expect(needs.settingsInRepo).toBe(false)
    expect(JSON.parse(needs.settingsJson)).toMatchObject({ 'loreMaster.outputs': [{ platform: 'github-pages' }], 'loreMaster.ignore': ['drafts/'] })
  })
})

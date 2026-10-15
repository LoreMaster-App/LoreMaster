import { detectCiProviders } from './detect-ci-providers.policy'

const ids = (files: string[]): string[] => detectCiProviders(files).map(provider => provider.id)

describe('detectCiProviders', () => {
  it('finds GitHub Actions from a workflow file', () => {
    expect(ids(['.github/workflows/ci.yml', 'README.md'])).toEqual(['github-actions'])
  })

  it('finds Azure Pipelines from the root file or its folder', () => {
    expect(ids(['azure-pipelines.yml'])).toEqual(['azure-pipelines'])
    expect(ids(['.azure-pipelines/build.yaml'])).toEqual(['azure-pipelines'])
  })

  it('reports every CI in use, in the providers order', () => {
    expect(ids(['azure-pipelines.yml', '.github/workflows/docs.yaml'])).toEqual(['github-actions', 'azure-pipelines'])
  })

  it('finds nothing in a repository without a pipeline, and ignores nested look-alikes', () => {
    expect(ids(['package.json', '.github/workflows/nested/ci.yml', 'docs/azure-pipelines.yml'])).toEqual([])
  })
})

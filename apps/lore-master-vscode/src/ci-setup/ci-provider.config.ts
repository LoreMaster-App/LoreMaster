import type { CiProviderId } from './ci-pipeline.contract'

export interface CiProvider {
  id:          CiProviderId
  label:       string
  /** Where the command writes its own pipeline file; an existing pipeline is never edited. */
  path:        string
  /** Workspace-relative globs whose presence says the repository already uses this CI. */
  detectGlobs: string[]
}

export const CI_PROVIDERS: readonly CiProvider[] = [
  {
    id:          'github-actions',
    label:       'GitHub Actions',
    path:        '.github/workflows/lore-master.yml',
    detectGlobs: ['.github/workflows/*.yml', '.github/workflows/*.yaml'],
  },
  {
    id:          'azure-pipelines',
    label:       'Azure Pipelines',
    path:        '.azure-pipelines/lore-master.yml',
    detectGlobs: ['azure-pipelines.yml', 'azure-pipelines.yaml', '.azure-pipelines/*.yml', '.azure-pipelines/*.yaml'],
  },
]

/** The engine binary inside a release's zip. */
export const ENGINE_BINARY = 'lore-master-engine'

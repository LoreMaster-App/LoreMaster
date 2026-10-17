import type { CiProviderId } from '../ci-pipeline-rendering'

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
  {
    id:          'gitlab-ci',
    label:       'GitLab CI/CD',
    path:        '.gitlab/lore-master.yml',
    detectGlobs: ['.gitlab-ci.yml', '.gitlab-ci.yaml', '.gitlab/*.yml'],
  },
  {
    id:          'bitbucket-pipelines',
    label:       'Bitbucket Pipelines',
    path:        'bitbucket-pipelines.yml',
    detectGlobs: ['bitbucket-pipelines.yml'],
  },
  {
    id:          'jenkins',
    label:       'Jenkins',
    path:        'Jenkinsfile.lore-master',
    detectGlobs: ['Jenkinsfile', 'Jenkinsfile.*'],
  },
  {
    id:          'circleci',
    label:       'CircleCI',
    path:        '.circleci/config.yml',
    detectGlobs: ['.circleci/config.yml', '.circleci/config.yaml'],
  },
]

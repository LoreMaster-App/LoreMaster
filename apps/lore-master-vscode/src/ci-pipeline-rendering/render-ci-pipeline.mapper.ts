import type { CiNeeds, CiProviderId } from './ci-pipeline.contract'
import { renderAzurePipelines } from './render-azure-pipelines.mapper'
import { renderBitbucketPipelines } from './render-bitbucket-pipelines.mapper'
import { renderCircleCi } from './render-circleci.mapper'
import { renderGitHubActions } from './render-github-actions.mapper'
import { renderGitLabCi } from './render-gitlab-ci.mapper'
import { renderJenkinsfile } from './render-jenkinsfile.mapper'

const RENDERERS: Record<CiProviderId, (needs: CiNeeds) => string> = {
  'github-actions':      renderGitHubActions,
  'azure-pipelines':     renderAzurePipelines,
  'gitlab-ci':           renderGitLabCi,
  'bitbucket-pipelines': renderBitbucketPipelines,
  'jenkins':             renderJenkinsfile,
  'circleci':            renderCircleCi,
}

/** The pipeline file of a CI system for these needs. */
export function renderCiPipeline (provider: CiProviderId, needs: CiNeeds): string {
  return RENDERERS[provider](needs)
}

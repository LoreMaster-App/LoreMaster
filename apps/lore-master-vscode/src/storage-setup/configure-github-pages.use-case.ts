import type { Output } from '../engine-protocol'
import type { GitHubPagesSetupUI } from './set-up-storages.use-case'

/** Configures one GitHub Pages output: the repository (blank = the workspace's own origin)
 *  and the branch (blank = gh-pages). Returns undefined when the user cancels. */
export async function configureGitHubPages (ui: GitHubPagesSetupUI): Promise<Output | undefined> {
  const repo = await ui.promptRepo()
  if (repo === undefined) {
    return undefined
  }
  const branch = await ui.promptBranch()
  if (branch === undefined) {
    return undefined
  }

  return {
    platform:       'github-pages',
    baseUrl:        '',
    space:          '',
    parentPageId:   '',
    titlePrefix:    '',
    direction:      'to-platform',
    content:        [{ type: 'markdown', roots: ['.'], template: 'default' }],
    mermaidMode:    '',
    titleCollision: '',
    linkMode:       '',
    repo:           repo.trim(),
    branch:         branch.trim(),
  }
}

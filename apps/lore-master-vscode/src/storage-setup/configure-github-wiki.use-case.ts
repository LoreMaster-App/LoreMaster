import type { Output } from '../engine-protocol'
import type { GitHubPagesSetupUI } from './set-up-storages.use-case'

/** Configures one GitHub wiki output: the repository (blank = the workspace's own origin).
 *  The wiki's branch is whatever GitHub gave it, so it is not asked. Returns undefined when
 *  the user cancels. */
export async function configureGitHubWiki (ui: GitHubPagesSetupUI): Promise<Output | undefined> {
  const repo = await ui.promptRepo()
  if (repo === undefined) {
    return undefined
  }

  return {
    platform:       'github-wiki',
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
    branch:         '',
  }
}

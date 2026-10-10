import type { Output } from './rpc-protocol.contract'

/** True for the outputs that publish through the user's git: a GitHub Pages site or a GitHub wiki. */
export function isGitHubOutput (output: Output): boolean {
  return output.platform === 'github-pages' || output.platform === 'github-wiki'
}

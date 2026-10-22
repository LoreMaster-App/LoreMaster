import { relative, dirname } from 'node:path'
import type { LinkMarkdownIt, LinkProblem, LinkTargets, LinkToken } from './link-check.contract'
import { markdownTargetOf } from './relative-markdown-link.algorithm'

/** What a problem is called on the page, and what the person is told. */
const TITLES: Record<LinkProblem, (shown: string) => string> = {
  broken:   shown => `This file does not exist: ${shown}`,
  unsynced: () => 'No storage syncs this file, so the link will not resolve on the platform',
}

function problemOf (target: string, documentPath: string, targets: LinkTargets): LinkProblem | undefined {
  if (!targets.exists(target)) { return 'broken' }

  return targets.isSynced(documentPath) === true && targets.isSynced(target) === false ? 'unsynced' : undefined
}

/**
 * Marks the relative links to other Markdown files the way a sync would find them: a link to a file that is not there is
 * `broken`, and a link from a synced file to one no storage syncs is `unsynced`. Both get a class and a title; links that
 * resolve are left alone.
 */
export function addLinkChecks<Md extends LinkMarkdownIt> (md: Md, targets: LinkTargets): Md {
  md.core.ruler.push('loremaster_link_checks', state => {
    const documentPath = state.env.currentDocument?.fsPath
    if (documentPath === undefined) { return }
    for (const token of state.tokens) {
      const children = token.type === 'inline' ? token.children ?? [] : []
      for (const child of children) {
        mark(child, documentPath, targets)
      }
    }
  })

  return md
}

function mark (token: LinkToken, documentPath: string, targets: LinkTargets): void {
  if (token.type !== 'link_open') { return }
  const href = token.attrGet('href')
  const target = href === null ? undefined : markdownTargetOf(href, documentPath)
  if (target === undefined) { return }
  const problem = problemOf(target, documentPath, targets)
  if (problem === undefined) { return }
  token.attrJoin('class', `loremaster-link-${problem}`)
  const shown = relative(dirname(documentPath), target)
  token.attrSet('title', TITLES[problem](shown))
}

import { relative, dirname } from 'node:path'
import type { LinkMarkdownIt, LinkProblem, LinkRenderRule, LinkTargets, LinkToken } from './link-check.contract'
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

/** The token as it is drawn, with the mark on it: a copy, since VS Code keeps the parsed page and renders it again. */
function marked (token: LinkToken, documentPath: string, targets: LinkTargets): LinkToken {
  const href = token.attrGet('href')
  const target = href === null ? undefined : markdownTargetOf(href, documentPath)
  const problem = target === undefined ? undefined : problemOf(target, documentPath, targets)
  if (target === undefined || problem === undefined) { return token }
  const copy: LinkToken = Object.assign(Object.create(Object.getPrototypeOf(token) as object) as LinkToken, token, { attrs: token.attrs?.map(pair => [...pair]) ?? null })
  copy.attrJoin('class', `loremaster-link-${problem}`)
  const shown = relative(dirname(documentPath), target)
  copy.attrSet('title', TITLES[problem](shown))

  return copy
}

/**
 * Marks the relative links to other Markdown files the way a sync would find them: a link to a file that is not there is
 * `broken`, and a link from a synced file to one no storage syncs is `unsynced`. Both get a class and a title; links that
 * resolve are left alone. VS Code only tells the renderer which file is open (the parse runs without it), so the check is made
 * when each link is drawn, by wrapping the renderer of links.
 */
export function addLinkChecks<Md extends LinkMarkdownIt> (md: Md, targets: LinkTargets): Md {
  const drawn = md.renderer.rules.link_open
  const rule: LinkRenderRule = (tokens, index, options, env, self) => {
    const documentPath = env.currentDocument?.fsPath
    const shown = documentPath === undefined ? tokens[index] : marked(tokens[index], documentPath, targets)
    const list = shown === tokens[index] ? tokens : tokens.map((each, at) => at === index ? shown : each)

    return drawn === undefined ? self.renderToken(list, index, options) : drawn(list, index, options, env, self)
  }
  md.renderer.rules.link_open = rule

  return md
}

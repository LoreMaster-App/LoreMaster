import {
  type Output,
  PAGES_PUBLISH_METHOD,
  type PagesPublishResult,
  SETTINGS_READ_METHOD,
  SETTINGS_SAVE_METHOD,
  type SettingsReadResult,
} from '../engine-protocol'

/** The one thing the flow needs from the engine client: to make a request. */
export interface PagesEngine {
  request<R> (method: string, params?: unknown): Promise<R>
}

/** Everything the publish flow asks of the editor, abstracted so it tests without the
 *  `vscode` module. */
export interface PagesUI {
  withProgress<T> (title: string, task: () => Promise<T>): Promise<T>
  report (lines: string[]): void
  status (message: string): void
  error (message: string): Promise<void>
}

/** What a GitHub Pages publish needs: the engine, the folder and the editor UI. */
export interface PagesDeps {
  engine:        PagesEngine
  workspaceRoot: string
  ui:            PagesUI
}

/**
 * Publishes the workspace's Markdown as a static site to its github-pages output. If the
 * workspace has no such output yet, one is added (publishing to the repo's own origin, the
 * gh-pages branch) and saved to .lore-master.yaml. The engine does the rest — generate and
 * push — through the user's own git.
 */
export async function publishPages (deps: PagesDeps): Promise<void> {
  const { engine, workspaceRoot, ui } = deps

  let output: number
  try {
    output = await ensureGitHubPagesOutput(engine, workspaceRoot)
  } catch (error) {
    await ui.error(messageOf(error))

    return
  }

  await publishPagesOutput({ engine, workspaceRoot, output, ui })
}

/** Publishes one already-chosen github-pages output and reports the result. This is the
 *  per-output path the unified sync fan-out calls for each github-pages output. */
export async function publishPagesOutput (deps: { engine: PagesEngine; workspaceRoot: string; output: number; ui: PagesUI }): Promise<void> {
  const { engine, workspaceRoot, output, ui } = deps

  let result: PagesPublishResult
  try {
    result = await ui.withProgress('LoreMaster: publishing to GitHub Pages', () =>
      engine.request<PagesPublishResult>(PAGES_PUBLISH_METHOD, { workspaceRoot, output }))
  } catch (error) {
    await ui.error(messageOf(error))

    return
  }

  reportResult(result, ui)
}

/** Finds the workspace's github-pages output, adding a default one when there is none. A
 *  fresh workspace is read back with one unconfigured Confluence scaffold; replace it
 *  rather than leaving a half-configured output beside the new one. */
async function ensureGitHubPagesOutput (engine: PagesEngine, workspaceRoot: string): Promise<number> {
  const read = await engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot })
  const existing = read.settings.outputs.findIndex(output => output.platform === 'github-pages')
  if (existing !== -1) {
    return existing
  }

  const outputs = read.firstSync ? [gitHubPagesOutput()] : [...read.settings.outputs, gitHubPagesOutput()]
  await engine.request(SETTINGS_SAVE_METHOD, { workspaceRoot, settings: { version: read.settings.version || 1, outputs } })

  return outputs.length - 1
}

/** A default github-pages output: the whole workspace's Markdown, to the repo's own origin,
 *  the gh-pages branch. Confluence-only fields stay empty (the validator rejects them here). */
function gitHubPagesOutput (): Output {
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
    repo:           '',
    branch:         '',
  }
}

function reportResult (result: PagesPublishResult, ui: PagesUI): void {
  const lines: string[] = Array.from(result.warnings ?? [], warning => `warning: ${warning}`)

  const errors = result.errors ?? []
  if (errors.length > 0) {
    for (const error of errors) {
      lines.push(`error: ${error}`)
    }
    lines.unshift(`Not published: the Markdown has ${errors.length} error(s). Fix them and publish again.`)
    ui.report(lines)
    ui.status(`LoreMaster: GitHub Pages not published (${errors.length} error(s))`)

    return
  }

  if (!result.changed) {
    lines.unshift('GitHub Pages is already up to date.')
    ui.report(lines)
    ui.status('LoreMaster: GitHub Pages already up to date')

    return
  }

  const where = result.commit ? `${result.branch} (${result.commit})` : result.branch
  lines.unshift(`Published ${result.files} file(s) to ${where}.${result.url ? ` ${result.url}` : ''}`)
  ui.report(lines)
  ui.status(result.url ? `LoreMaster: published to GitHub Pages — ${result.url}` : 'LoreMaster: published to GitHub Pages')
}

function messageOf (error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

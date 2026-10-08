import { type GeneratorRun, WATCH_ROUTE_METHOD, type WatchRouteResult } from '../engine-protocol'
import { runGenerators } from '../generators-command'

/** What applying a batch of changes needs from the editor. */
export interface ApplyChangesDeps {
  engine:        { request<R> (method: string, params?: unknown): Promise<R> }
  workspaceRoot: string
  /** Syncs these workspace-relative files, or the whole workspace when absent. Throws when the
   *  sync failed, so the batch is tried again. */
  sync:          (scope: string[] | undefined) => Promise<void>
  log:           (line: string) => void
}

/** What was written, so the changes it causes are not mistaken for the user's. */
export interface AppliedChanges {
  /** Every file the batch is known to have written: generated pages and the pages synced. */
  touched: string[]
}

/**
 * Handles one batch of changed files: asks the engine which generators read them, runs only
 * those, and syncs only the pages that changed or were rewritten. A generator that fails is
 * logged but does not stop the sync of the rest; a changed settings file regenerates and syncs
 * everything.
 */
export async function applyChanges (deps: ApplyChangesDeps, changed: string[]): Promise<AppliedChanges> {
  const { engine, workspaceRoot, log } = deps
  const route = await engine.request<WatchRouteResult>(WATCH_ROUTE_METHOD, { workspaceRoot, changed })

  const written: string[] = []
  if (route.generators.length > 0) {
    const { runs } = await runGenerators(engine, workspaceRoot, route.generators)
    for (const run of runs) {
      log(describe(run))
      written.push(...(run.written ?? []))
    }
  }

  const scope = [...new Set([...route.markdown, ...written.filter(path => isMarkdown(path))])].sort(byCodePoint)
  if (scope.length === 0 && !route.everything) {
    log(`nothing to sync for ${summary(changed)}`)

    return { touched: written }
  }

  log(route.everything ? 'settings changed: syncing everything' : `syncing ${summary(scope)}`)
  await deps.sync(route.everything ? undefined : scope)

  return { touched: [...new Set([...written, ...scope])] }
}

/** Orders paths the way the engine does (by code point), whatever the locale. */
function byCodePoint (a: string, b: string): number {
  if (a < b) {
    return -1
  }

  return a > b ? 1 : 0
}

function isMarkdown (path: string): boolean {
  return path.toLowerCase().endsWith('.md')
}

function describe (run: GeneratorRun): string {
  if (run.error) {
    return `${run.type} → ${run.output}: failed: ${run.error}`
  }

  return `${run.type} → ${run.output}: ${run.written?.length ?? 0} written, ${run.unchanged?.length ?? 0} unchanged, ${run.removed?.length ?? 0} removed`
}

/** Up to three files by name, then a count. */
function summary (files: string[]): string {
  return files.length <= 3 ? files.join(', ') : `${files.slice(0, 3).join(', ')} and ${files.length - 3} more`
}

import type { EngineClient } from '../engine-process'
import { PAGES_CHECK_METHOD, type PagesCheckResult } from '../engine-protocol'

/** What a check of a GitHub Pages output found, or why it could not be made. */
export type CheckPagesOutcome = { ok: true; message: string; upToDate: boolean } | { ok: false; reason: string }

const SHOWN_CHANGES = 5

/**
 * Asks the engine whether publishing a GitHub Pages output would change anything. It renders
 * the site and compares it with the published branch, publishing nothing; the answer is one
 * sentence naming the first few files.
 */
export async function checkPages (engine: EngineClient, workspaceRoot: string, output: number): Promise<CheckPagesOutcome> {
  let result: PagesCheckResult
  try {
    result = await engine.request<PagesCheckResult>(PAGES_CHECK_METHOD, { workspaceRoot, output })
  } catch (error) {
    return { ok: false, reason: error instanceof Error ? error.message : String(error) }
  }

  if (result.errors?.length) {
    return { ok: false, reason: `the site would not build: ${result.errors[0]}${result.errors.length > 1 ? ` (and ${result.errors.length - 1} more)` : ''}` }
  }
  const branch = result.branch ?? 'the site branch'
  if (result.upToDate) {
    return { ok: true, upToDate: true, message: `${branch} is up to date (${result.files} files).` }
  }

  const changes = result.changes ?? []
  const named = changes.slice(0, SHOWN_CHANGES).map(change => `${change.kind} ${change.path}`).join(', ')
  const more = result.changesTotal > SHOWN_CHANGES ? `, and ${result.changesTotal - SHOWN_CHANGES} more` : ''

  return { ok: true, upToDate: false, message: `${branch} is out of date: ${result.changesTotal} files would change (${named}${more}).` }
}

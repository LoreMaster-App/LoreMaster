import { HOST_OPEN_EXTERNAL_METHOD, type OpenExternalParams } from '../engine-protocol'
import type { EngineClient } from '../engine-process'

/**
 * Answers the engine's `host/openExternal` by opening the URL in the user's browser — the
 * OAuth sign-in's authorize page. The opener is injected (`vscode.env.openExternal` in the
 * extension) so this stays testable without the `vscode` module. Returns the subscription
 * to dispose on deactivate.
 */
export function answerOpenExternal (deps: { engine: EngineClient; open: (url: string) => Promise<void> }): { dispose (): void } {
  return deps.engine.onRequest(HOST_OPEN_EXTERNAL_METHOD, async params => {
    await deps.open((params as OpenExternalParams).url)

    return {}
  })
}

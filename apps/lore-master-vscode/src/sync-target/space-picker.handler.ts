import { type Space, SPACE_LIST_METHOD, type SpaceListResult } from '../engine-protocol'
import type { EngineRequester } from '../connection-setup'

/** Shows the session's spaces and returns the chosen one. The engine returns every space
 *  already (it follows the platform's paging), so the QuickPick's own type-to-filter is
 *  all the narrowing needed. */
export async function pickSpace (deps: { engine: EngineRequester; sessionId: string; ui: SpacePickerUI }): Promise<Space | undefined> {
  const { spaces } = await deps.engine.request<SpaceListResult>(SPACE_LIST_METHOD, { sessionId: deps.sessionId })

  return deps.ui.pickSpace(spaces)
}

/** The one prompt the space picker drives. */
export interface SpacePickerUI {
  pickSpace (spaces: Space[]): Promise<Space | undefined>
}

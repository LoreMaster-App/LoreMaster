import { FILES_PLAN_MOVE_METHOD, type FileMove, type FilesPlanMoveResult } from '../engine-protocol'

export interface FileMovesEngine {
  request: <T>(method: string, params: unknown) => Promise<T>
}

/**
 * Asks the engine what else a move calls for. A failure is not a reason to hold the user's
 * move back, so it answers undefined and the move goes ahead as it was.
 */
export async function planMove (engine: FileMovesEngine, workspaceRoot: string, moves: FileMove[]): Promise<FilesPlanMoveResult | undefined> {
  try {
    return await engine.request<FilesPlanMoveResult>(FILES_PLAN_MOVE_METHOD, { workspaceRoot, moves })
  } catch {
    return undefined
  }
}

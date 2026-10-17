import type { Output } from '../engine-protocol'
import { configurePlatform } from './configure-platform.use-case'
import type { StorageSetupDeps } from './storage-setup.contract'

/**
 * The first-run "where to sync": pick one or more of the platforms the engine describes, configure
 * each in turn, and return the outputs to save. Returns undefined when the user cancels or
 * configures nothing, so the caller writes nothing.
 */
export async function setUpStorages (deps: StorageSetupDeps): Promise<Output[] | undefined> {
  const picked = await deps.ui.pickStorageTypes(deps.description.platforms)
  if (!picked || picked.length === 0) {
    return undefined
  }

  const outputs: Output[] = []
  const chosen = deps.description.platforms.filter(each => picked.includes(each.id))
  for (const platform of chosen) {
    const output = await configurePlatform(deps, platform)
    if (output) {
      outputs.push(output)
    }
  }

  return outputs.length > 0 ? outputs : undefined
}

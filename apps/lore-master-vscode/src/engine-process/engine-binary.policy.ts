import { join } from 'node:path'

/** What {@link resolveEngineBinary} needs to decide. */
export interface EngineBinaryLocation {
  /** `process.platform` of the host running the extension. */
  platform:        NodeJS.Platform
  /** The installed extension's root (`context.extensionPath`). */
  extensionPath:   string
  /** The `loreMaster.engine.path` setting, for pointing at a locally built engine
   *  during development. A blank or whitespace value is treated as unset. */
  configuredPath?: string
}

/**
 * Decides which engine binary to spawn.
 *
 * A configured dev path wins; otherwise the engine bundled in the extension's `bin/`.
 * Each per-platform `.vsix` carries exactly one binary at `bin/lore-master-engine`
 * (`.exe` on win32), so the architecture never enters the path — the Marketplace
 * already served the package built for this platform (see #59's correction to E5).
 */
export function resolveEngineBinary ({ platform, extensionPath, configuredPath }: EngineBinaryLocation): string {
  const override = configuredPath?.trim()
  if (override) {
    return override
  }

  const name = platform === 'win32' ? 'lore-master-engine.exe' : 'lore-master-engine'

  return join(extensionPath, 'bin', name)
}

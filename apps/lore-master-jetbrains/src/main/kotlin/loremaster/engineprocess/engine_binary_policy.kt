package loremaster.engineprocess

import java.nio.file.Path

/**
 * Decides which engine binary to spawn. A configured path (for a locally built engine) wins; otherwise the
 * engine bundled in the plugin's `bin/` folder. Each platform build carries exactly one binary there, so the
 * architecture never enters the path.
 */
fun resolveEngineBinary(isWindows: Boolean, pluginPath: Path, configuredPath: String? = null): Path {
    val override = configuredPath?.trim()
    if (!override.isNullOrEmpty()) {
        return Path.of(override)
    }

    return pluginPath.resolve("bin").resolve(if (isWindows) "lore-master-engine.exe" else "lore-master-engine")
}

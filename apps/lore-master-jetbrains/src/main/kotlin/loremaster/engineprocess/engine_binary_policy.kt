package loremaster.engineprocess

import java.nio.file.Path

/** The engine's platform folder name (`windows-amd64`, `darwin-arm64`, ...) for the JVM's `os.name` and `os.arch`. */
fun enginePlatform(osName: String, osArch: String): String {
    val os = osName.lowercase()
    val goos = when {
        os.contains("win") -> "windows"
        os.contains("mac") || os.contains("darwin") -> "darwin"
        else -> "linux"
    }
    val arch = osArch.lowercase()
    val goarch = if (arch == "aarch64" || arch == "arm64") "arm64" else "amd64"

    return "$goos-$goarch"
}

/**
 * Decides which engine binary to spawn. A configured path (for a locally built engine) wins; otherwise the engine
 * bundled for this platform under the plugin's `bin/<platform>/`. One plugin zip serves every platform, so it carries
 * a binary per platform, laid out as the engine's own build writes them.
 */
fun resolveEngineBinary(platform: String, pluginPath: Path, configuredPath: String? = null): Path {
    val override = configuredPath?.trim()
    if (!override.isNullOrEmpty()) {
        return Path.of(override)
    }
    val name = if (platform.startsWith("windows")) "lore-master-engine.exe" else "lore-master-engine"

    return pluginPath.resolve("bin").resolve(platform).resolve(name)
}

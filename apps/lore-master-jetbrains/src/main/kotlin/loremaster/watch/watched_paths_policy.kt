package loremaster.watch

import java.nio.file.Path

/** Folders whose changes never matter: dependencies, build output and tool caches. Hidden folders are skipped too. */
private val SKIPPED_FOLDERS = setOf("node_modules", "vendor", "bin", "obj", "dist", "build", "out", "target", "venv", "env", "__pycache__", "coverage", "testdata")

/** The files that can matter: Markdown, the settings, and what the generators read. */
private val WATCHED_EXTENSIONS = setOf(
    ".md", ".xml", ".go", ".mod", ".yaml", ".yml", ".json", ".ts", ".tsx", ".mts", ".cts", ".js", ".jsx",
    ".py", ".toml", ".cfg", ".cs", ".csproj", ".props", ".dart",
)

/** The configuration: the editor's settings, and the deprecated yaml until it is gone. */
private val SETTINGS_FILES = setOf(".lore-master.yaml", ".vscode/settings.json")

/** The workspace-relative, '/'-separated path of a file, or null when it is outside the folder. */
fun workspacePath(folder: Path, file: Path): String? {
    val relative = runCatching { folder.relativize(file) }.getOrNull() ?: return null
    if (relative.toString().isEmpty() || relative.startsWith("..") || relative.isAbsolute) {
        return null
    }

    return relative.joinToString("/")
}

/** Whether a change to this file (workspace-relative, '/'-separated) can matter to the sync. */
fun isWatchedPath(path: String): Boolean {
    if (path in SETTINGS_FILES) {
        return true
    }
    val segments = path.split('/')
    if (segments.dropLast(1).any { it.startsWith(".") || it.lowercase() in SKIPPED_FOLDERS }) {
        return false
    }
    val name = segments.last()
    val dot = name.lastIndexOf('.')

    return !name.startsWith(".") && dot > 0 && name.substring(dot).lowercase() in WATCHED_EXTENSIONS
}

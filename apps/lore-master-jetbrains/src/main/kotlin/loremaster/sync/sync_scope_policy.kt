package loremaster.sync

import java.nio.file.Path

private val MARKDOWN_EXTENSIONS = setOf("md", "markdown")

fun isMarkdownFile(path: Path): Boolean = path.fileName?.toString()?.substringAfterLast('.', "")?.lowercase() in MARKDOWN_EXTENSIONS

/**
 * The files a sync is limited to, as the engine wants them: workspace-relative with '/'. Null (everything) when nothing is
 * selected, or when a selected file lies outside the workspace and so cannot be scoped.
 */
fun syncScope(workspaceRoot: Path, selected: List<Path>): List<String>? {
    val markdown = selected.filter(::isMarkdownFile)
    if (markdown.isEmpty()) {
        return null
    }
    val relative = markdown.map { workspaceRoot.relativize(it) }
    if (relative.any { it.startsWith("..") || it.isAbsolute }) {
        return null
    }

    return relative.map { it.joinToString("/") }.distinct().sorted()
}

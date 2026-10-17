package loremaster.watch

/** The engine's `watch/route`: which generators read the changed files and which Markdown pages they are. Reads settings only. */
const val METHOD_WATCH_ROUTE = "watch/route"

/** [changed] are workspace-relative, '/'-separated paths. */
data class WatchRouteParams(val workspaceRoot: String, val changed: List<String>)

/**
 * What to do about them. [everything] is true when the settings changed: run every generator, sync everything.
 * [generators] index the settings' generators list; [markdown] are the changed Markdown files.
 */
data class WatchRouteResult(
    val everything: Boolean = false,
    val generators: List<Int> = emptyList(),
    val markdown: List<String> = emptyList(),
)

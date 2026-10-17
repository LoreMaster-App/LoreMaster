package loremaster.watch

import loremaster.engineprocess.EngineRpcClient
import loremaster.generators.describeRuns
import loremaster.generators.runGenerators
import java.util.concurrent.TimeUnit

/** What applying a batch did: the files it is known to have written (so their change events are not mistaken for the user's). */
data class AppliedChanges(val touched: List<String>, val log: List<String>)

/**
 * Handles one batch of changed files: asks the engine which generators read them, runs only those, and syncs only the pages that
 * changed or were rewritten. [sync] gets the workspace-relative files to sync, or null for everything; it throws when the sync
 * failed so the batch is tried again. A changed settings file regenerates and syncs everything.
 */
fun applyChanges(engine: EngineRpcClient, workspaceRoot: String, changed: List<String>, sync: (scope: List<String>?) -> Unit): AppliedChanges {
    val route = engine.request(METHOD_WATCH_ROUTE, WatchRouteParams(workspaceRoot, changed), WatchRouteResult::class.java)
        .get(1, TimeUnit.MINUTES)
    val log = mutableListOf<String>()

    val written = mutableListOf<String>()
    if (route.generators.isNotEmpty()) {
        val result = runGenerators(engine, workspaceRoot, prepare = false, only = route.generators)
        log += describeRuns(result).lines()
        result.runs.forEach { written += it.written.orEmpty() }
    }

    val scope = (route.markdown + written.filter { it.lowercase().endsWith(".md") }).distinct().sorted()
    if (scope.isEmpty() && !route.everything) {
        log += "nothing to sync for ${summary(changed)}"

        return AppliedChanges(written, log)
    }

    log += if (route.everything) "settings changed: syncing everything" else "syncing ${summary(scope)}"
    sync(if (route.everything) null else scope)

    return AppliedChanges((written + scope).distinct(), log)
}

/** Up to three files by name, then a count. */
fun summary(files: List<String>): String =
    if (files.size <= 3) files.joinToString(", ") else "${files.take(3).joinToString(", ")} and ${files.size - 3} more"

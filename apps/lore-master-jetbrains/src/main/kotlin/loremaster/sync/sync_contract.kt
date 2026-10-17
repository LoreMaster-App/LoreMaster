package loremaster.sync

/** The engine's `sync/plan`: plans a sync of one output and writes nothing. */
const val METHOD_SYNC_PLAN = "sync/plan"

/** The engine's `sync/execute`: carries out a plan made by `sync/plan`. */
const val METHOD_SYNC_EXECUTE = "sync/execute"

/** The engine's `host/progress` notification, sent while a sync runs. */
const val METHOD_HOST_PROGRESS = "host/progress"

/** [output] is the index in the settings' outputs list; [scope] limits the plan to these workspace-relative files. */
data class SyncPlanParams(val sessionId: String, val workspaceRoot: String, val output: Int, val scope: List<String>? = null)

/** What the sync will do to one page. [kind] is create, adopt, update, move, rename_title, unchanged, conflict or orphan. */
data class PlanAction(
    val kind: String = "",
    val path: String? = null,
    val title: String = "",
    val pageId: String? = null,
    val url: String? = null,
    val changes: List<String>? = null,
    val reason: String? = null,
)

/** The plan. With [errors] it cannot be executed. */
data class SyncPlanResult(
    val planId: String = "",
    val actions: List<PlanAction> = emptyList(),
    val counts: Map<String, Int> = emptyMap(),
    val warnings: List<String>? = null,
    val errors: List<String>? = null,
)

data class SyncExecuteParams(val planId: String, val force: Boolean = false, val prune: Boolean = false)

/** [outcome] is written, unchanged, skipped, reported, trashed or failed. */
data class PageOutcome(
    val path: String? = null,
    val title: String = "",
    val planned: String = "",
    val outcome: String = "",
    val pageId: String? = null,
    val url: String? = null,
    val error: String? = null,
)

data class SyncExecuteResult(
    val pages: List<PageOutcome> = emptyList(),
    val rewritten: List<String>? = null,
    val journalId: String? = null,
    val warnings: List<String>? = null,
)

/** One step of a running sync. */
data class ProgressParams(val planId: String = "", val message: String = "", val done: Int = 0, val total: Int = 0)

/** What the user chose after reading the plan. */
data class SyncChoice(val force: Boolean = false, val prune: Boolean = false)

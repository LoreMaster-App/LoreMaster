package loremaster.rollback

/** The engine's `sync/journals`: the syncs of a workspace that can still be rolled back, newest first. */
const val METHOD_SYNC_JOURNALS = "sync/journals"

/** The engine's `sync/rollback`: pages the sync created are trashed, pages it updated are put back, local files get their earlier content. */
const val METHOD_SYNC_ROLLBACK = "sync/rollback"

data class SyncJournalsParams(val workspaceRoot: String)

/** One rollbackable sync. [startedAt] is an ISO-8601 instant. */
data class JournalSummary(
    val id: String = "",
    val startedAt: String = "",
    val name: String? = null,
    val baseUrl: String = "",
    val space: String = "",
    val parentPageId: String? = null,
    val created: Int = 0,
    val updated: Int = 0,
    val trashed: Int = 0,
    val files: Int = 0,
)

data class SyncJournalsResult(val journals: List<JournalSummary> = emptyList())

/** [dryRun] reports what would happen and changes nothing; [force] also rolls back pages and files edited since the sync. */
data class SyncRollbackParams(
    val sessionId: String,
    val workspaceRoot: String,
    val journalId: String? = null,
    val dryRun: Boolean = false,
    val force: Boolean = false,
)

/** [kind] is created, updated or trashed (what the sync did); [result] trashed, restored, would-trash, would-restore, gone, skipped, failed or irreversible. */
data class RollbackPage(val pageId: String = "", val title: String = "", val kind: String = "", val result: String = "", val reason: String? = null)

data class RollbackFile(val path: String = "", val result: String = "", val reason: String? = null)

data class SyncRollbackResult(
    val journalId: String = "",
    val dryRun: Boolean = false,
    val pages: List<RollbackPage> = emptyList(),
    val files: List<RollbackFile> = emptyList(),
    val complete: Boolean = false,
)

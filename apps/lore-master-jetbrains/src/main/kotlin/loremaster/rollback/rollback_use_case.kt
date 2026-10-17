package loremaster.rollback

import loremaster.engineprocess.EngineRpcClient
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.concurrent.TimeUnit

/** The syncs of the workspace that can still be rolled back, newest first. */
fun listJournals(engine: EngineRpcClient, workspaceRoot: String): List<JournalSummary> =
    engine.request(METHOD_SYNC_JOURNALS, SyncJournalsParams(workspaceRoot), SyncJournalsResult::class.java)
        .get(1, TimeUnit.MINUTES)
        .journals

/** Undoes one sync (or, with a null [journalId], the latest). With [dryRun] nothing changes and the result says what would. */
fun rollbackSync(engine: EngineRpcClient, sessionId: String, workspaceRoot: String, journalId: String?, dryRun: Boolean, force: Boolean): SyncRollbackResult =
    engine.request(METHOD_SYNC_ROLLBACK, SyncRollbackParams(sessionId, workspaceRoot, journalId, dryRun, force), SyncRollbackResult::class.java)
        .get(30, TimeUnit.MINUTES)

private val WHEN = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm").withZone(ZoneId.systemDefault())

/** What a picker shows for a journal: when, where, and what it did. */
fun journalLabel(journal: JournalSummary): String {
    val moment = runCatching { WHEN.format(Instant.parse(journal.startedAt)) }.getOrDefault(journal.startedAt)
    val where = journal.name?.takeIf { it.isNotBlank() } ?: journal.space.ifBlank { journal.baseUrl }

    return "$moment  $where  (${journal.created} created, ${journal.updated} updated, ${journal.trashed} trashed, ${journal.files} files)"
}

/** Results that mean something was left alone, because the page or file was edited since the sync. */
private val LEFT_ALONE = setOf("skipped", "irreversible", "failed")

/** Whether a dry run shows work that only [SyncRollbackParams.force] would do. */
fun needsForce(result: SyncRollbackResult): Boolean =
    result.pages.any { it.result == "skipped" } || result.files.any { it.result == "skipped" }

/** One line per kind of outcome, then the pages and files that were left alone and why. */
fun describeRollback(result: SyncRollbackResult): String {
    val counts = (result.pages.map { it.result } + result.files.map { it.result }).groupingBy { it }.eachCount()
    val summary = if (counts.isEmpty()) "Nothing to undo." else counts.entries.joinToString(", ") { "${it.value} ${it.key}" }
    val left = result.pages.filter { it.result in LEFT_ALONE }.map { "${it.title}: ${it.result}${it.reason?.let { reason -> " ($reason)" }.orEmpty()}" } +
        result.files.filter { it.result in LEFT_ALONE }.map { "${it.path}: ${it.result}${it.reason?.let { reason -> " ($reason)" }.orEmpty()}" }

    return (listOf(summary) + left.take(8) + if (left.size > 8) listOf("and ${left.size - 8} more") else emptyList()).joinToString("\n")
}

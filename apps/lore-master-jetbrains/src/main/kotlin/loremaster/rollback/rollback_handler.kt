package loremaster.rollback

import com.intellij.notification.NotificationType
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.Messages
import com.intellij.openapi.vfs.LocalFileSystem
import com.intellij.openapi.vfs.VfsUtil
import loremaster.connection.closeSession
import loremaster.connection.openSession
import loremaster.connection.signInCredential
import loremaster.engineprogress.TaskReport
import loremaster.engineprogress.chooseOne
import loremaster.engineprogress.onUiThread
import loremaster.engineprogress.runEngineTask
import java.nio.file.Path

private const val ROLL_BACK = "Roll back"
private const val ROLL_BACK_AND_DISCARD = "Roll back and discard later edits"

/**
 * "LoreMaster: Roll Back a Sync": picks one of the syncs that can still be undone, shows what undoing it would do (a dry run),
 * and on a yes trashes the pages it created, puts back the pages it updated and restores the local files.
 */
class RollbackHandler : AnAction() {
    override fun actionPerformed(event: AnActionEvent) {
        val project = event.project ?: return
        val root = project.basePath ?: return
        runEngineTask(project, "Rolling back a sync") { engine, _ ->
            val journals = listJournals(engine, root)
            if (journals.isEmpty()) return@runEngineTask TaskReport("There is no sync to roll back.")
            val journal = chooseOne(project, "LoreMaster: Roll Back", "Which sync?", journals.map(::journalLabel))?.let(journals::getOrNull)
                ?: return@runEngineTask TaskReport("Cancelled.")
            val credential = signInCredential(project, journal.baseUrl) ?: return@runEngineTask TaskReport("Cancelled.")

            val session = openSession(engine, journal.baseUrl, credential)
            try {
                val preview = rollbackSync(engine, session.sessionId, root, journal.id, dryRun = true, force = false)
                val force = confirm(project, preview) ?: return@runEngineTask TaskReport("Cancelled.")
                val done = rollbackSync(engine, session.sessionId, root, journal.id, dryRun = false, force = force)
                refresh(root)

                TaskReport(
                    (if (done.complete) "Rolled back.\n" else "Rolled back in part; the sync stays in the list.\n") + describeRollback(done),
                    if (done.complete) NotificationType.INFORMATION else NotificationType.WARNING,
                )
            } finally {
                runCatching { closeSession(engine, session.sessionId) }
            }
        }
    }

    /** Asks with the dry run in front of the user. Null: cancelled; otherwise whether to discard edits made since the sync. */
    private fun confirm(project: Project, preview: SyncRollbackResult): Boolean? = onUiThread {
        val text = "Undoing this sync would:\n\n${describeRollback(preview)}"
        if (needsForce(preview)) {
            when (Messages.showDialog(project, text, "LoreMaster: Roll Back", arrayOf(ROLL_BACK, ROLL_BACK_AND_DISCARD, "Cancel"), 0, Messages.getQuestionIcon())) {
                0 -> false
                1 -> true
                else -> null
            }
        } else {
            if (Messages.showYesNoDialog(project, text, "LoreMaster: Roll Back", ROLL_BACK, "Cancel", Messages.getQuestionIcon()) == Messages.YES) false else null
        }
    }

    private fun refresh(root: String) {
        LocalFileSystem.getInstance().refreshAndFindFileByNioFile(Path.of(root))?.let { VfsUtil.markDirtyAndRefresh(true, true, true, it) }
    }
}

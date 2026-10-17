package loremaster.generators

import com.intellij.notification.NotificationType
import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.application.ModalityState
import com.intellij.openapi.progress.ProgressIndicator
import com.intellij.openapi.progress.ProgressManager
import com.intellij.openapi.progress.Task
import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.Messages
import loremaster.engineprocess.EngineProcessStore
import loremaster.engineprogress.TaskReport
import loremaster.engineprogress.notify
import loremaster.sync.StorageSync
import loremaster.sync.SyncTarget

/**
 * Runs the project's generators, then syncs [target], so generated pages (test results, API documentation) are as fresh as the
 * code when they reach the storage. When a generator fails the user is asked before the pages go out; with no generator
 * configured the Markdown is synced and the user told why nothing was generated.
 */
fun generateAndSync(project: Project, target: SyncTarget, onFinished: () -> Unit = {}) {
    val root = project.basePath ?: return
    val prepare = askToPrepare(project, "LoreMaster: Generate and Sync") ?: return

    ProgressManager.getInstance().run(object : Task.Backgroundable(project, "Generating the documentation", false) {
        private var syncNow = false

        override fun run(indicator: ProgressIndicator) {
            val engine = ApplicationManager.getApplication().getService(EngineProcessStore::class.java).connection()
            val outcome = runCatching { runGenerators(engine, root, prepare) }
            val result = outcome.getOrNull()
            if (result == null) {
                notify(project, TaskReport("The generators failed: ${outcome.exceptionOrNull()?.message}", NotificationType.ERROR))

                return
            }
            val failed = result.runs.any { it.error != null }
            notify(project, TaskReport(describeRuns(result), if (failed) NotificationType.WARNING else NotificationType.INFORMATION))
            syncNow = !failed || askSyncAnyway(project)
        }

        override fun onSuccess() {
            if (syncNow) StorageSync(project).run(null, target, onFinished) else onFinished()
        }
    })
}

private fun askSyncAnyway(project: Project): Boolean {
    var answer = false
    ApplicationManager.getApplication().invokeAndWait({
        answer = Messages.showYesNoDialog(
            project,
            "Some generators failed, so some generated pages may be missing or stale. Sync anyway?",
            "LoreMaster: Generate and Sync",
            "Sync anyway",
            "Cancel",
            Messages.getWarningIcon(),
        ) == Messages.YES
    }, ModalityState.any())

    return answer
}

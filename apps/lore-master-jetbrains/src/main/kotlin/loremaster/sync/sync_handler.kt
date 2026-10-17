package loremaster.sync

import com.google.gson.Gson
import com.intellij.notification.NotificationGroupManager
import com.intellij.notification.NotificationType
import com.intellij.openapi.actionSystem.ActionUpdateThread
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.actionSystem.CommonDataKeys
import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.application.ModalityState
import com.intellij.openapi.progress.ProgressIndicator
import com.intellij.openapi.progress.ProgressManager
import com.intellij.openapi.progress.Task
import com.intellij.openapi.project.Project
import com.intellij.openapi.vfs.LocalFileSystem
import com.intellij.openapi.vfs.VfsUtil
import com.intellij.openapi.vfs.VirtualFile
import loremaster.connection.Credential
import loremaster.connection.signInCredential
import loremaster.engineprocess.EngineProcessStore
import loremaster.engineprocess.EngineRpcError
import loremaster.engineprogress.chooseOne
import loremaster.projectsettings.OutputSettings
import loremaster.projectsettings.readProjectSettings
import java.nio.file.Path
import java.util.concurrent.ExecutionException

private const val UNAUTHORIZED = -32001

/** "LoreMaster: Sync to Confluence": the selected Markdown files, or the whole project, to a Confluence output. */
class SyncHandler : AnAction() {
    private val gson = Gson()

    override fun getActionUpdateThread() = ActionUpdateThread.BGT

    override fun update(event: AnActionEvent) {
        event.presentation.isEnabledAndVisible = event.project?.basePath != null
    }

    override fun actionPerformed(event: AnActionEvent) {
        val project = event.project ?: return
        val root = project.basePath ?: return
        val selected = event.getData(CommonDataKeys.VIRTUAL_FILE_ARRAY).orEmpty().flatMap(::markdownBelow).map { Path.of(it.path) }
        val scope = syncScope(Path.of(root), selected)

        ProgressManager.getInstance().run(object : Task.Backgroundable(project, "Syncing to Confluence", false) {
            override fun run(indicator: ProgressIndicator) {
                val store = ApplicationManager.getApplication().getService(EngineProcessStore::class.java)
                val outcome = runCatching { sync(project, root, scope, store, indicator) }
                tell(project, root, outcome)
            }
        })
    }

    private fun sync(project: Project, root: String, scope: List<String>?, store: EngineProcessStore, indicator: ProgressIndicator): SyncOutcome? {
        val engine = store.connection()
        val outputs = readProjectSettings(engine, root).settings.outputs
        val confluence = outputs.withIndex().filter { it.value.platform == "confluence" }
        if (confluence.isEmpty()) {
            error("No Confluence output is configured. Add one to loreMaster.outputs (LoreMaster: Open Project Settings).")
        }
        val chosen = if (confluence.size == 1) confluence.single() else chooseOutput(project, confluence) ?: return null

        val stopListening = store.listen { method, params ->
            if (method == METHOD_HOST_PROGRESS && params != null) {
                val step = gson.fromJson(params, ProgressParams::class.java)
                indicator.text = step.message
                if (step.total > 0) indicator.fraction = step.done.toDouble() / step.total
            }
        }
        try {
            val flow = RunSync(engine, { askCredential(project, it) }, { review(project, it, chosen.value) })

            return flow.run(root, chosen.value, chosen.index, scope)
        } finally {
            stopListening()
        }
    }

    private fun chooseOutput(project: Project, outputs: List<IndexedValue<OutputSettings>>): IndexedValue<OutputSettings>? {
        return chooseOne(project, "LoreMaster: Sync", "Which output?", outputs.map { it.value.label() })?.let(outputs::getOrNull)
    }

    private fun askCredential(project: Project, baseUrl: String): Credential? = signInCredential(project, baseUrl)

    private fun review(project: Project, plan: SyncPlanResult, output: OutputSettings): SyncChoice? {
        if (changeCount(plan) == 0 && (plan.counts["conflict"] ?: 0) == 0 && (plan.counts["orphan"] ?: 0) == 0) {
            return SyncChoice()
        }
        var choice: SyncChoice? = null
        onUiThread {
            val dialog = SyncPlanDialogComponent(project, plan, output.label())
            if (dialog.showAndGet()) choice = dialog.choice()
        }

        return choice
    }

    private fun tell(project: Project, root: String, outcome: Result<SyncOutcome?>) {
        val group = NotificationGroupManager.getInstance().getNotificationGroup("LoreMaster")
        val result = outcome.getOrNull()
        val failure = outcome.exceptionOrNull()
        val (message, type) = when {
            failure != null -> describeFailure(failure) to NotificationType.ERROR
            result == null || result is SyncOutcome.Cancelled -> "The sync was cancelled." to NotificationType.INFORMATION
            result is SyncOutcome.Refused -> "The sync cannot run:\n${result.errors.joinToString("\n")}" to NotificationType.ERROR
            result is SyncOutcome.Done -> report(result)
            else -> return
        }
        group.createNotification("LoreMaster", message, type).notify(project)
        LocalFileSystem.getInstance().findFileByNioFile(Path.of(root))?.let { VfsUtil.markDirtyAndRefresh(true, true, true, it) }
    }

    private fun report(done: SyncOutcome.Done): Pair<String, NotificationType> {
        val pages = done.report.pages
        val failed = pages.filter { it.outcome == "failed" }
        val written = pages.count { it.outcome == "written" }
        val text = buildString {
            append("$written page(s) written, ${pages.count { it.outcome == "unchanged" }} unchanged")
            if (failed.isNotEmpty()) append(", ${failed.size} failed: ${failed.first().title}: ${failed.first().error}")
            done.report.warnings.orEmpty().forEach { append("\nWarning: $it") }
        }

        return text to if (failed.isEmpty()) NotificationType.INFORMATION else NotificationType.WARNING
    }

    private fun describeFailure(failure: Throwable): String {
        val cause = (failure as? ExecutionException)?.cause ?: failure
        if (cause is EngineRpcError && cause.code == UNAUTHORIZED) {
            return "Confluence refused the sign-in. Run LoreMaster: Connect to Confluence to enter it again."
        }

        return "The sync failed: ${cause.message ?: cause.javaClass.simpleName}"
    }

    private fun markdownBelow(file: VirtualFile): List<VirtualFile> {
        if (!file.isDirectory) return listOf(file)
        val found = mutableListOf<VirtualFile>()
        VfsUtil.visitChildrenRecursively(file, object : com.intellij.openapi.vfs.VirtualFileVisitor<Unit>() {
            override fun visitFile(candidate: VirtualFile): Boolean {
                if (!candidate.isDirectory) found.add(candidate)
                return true
            }
        })

        return found
    }

    private fun onUiThread(body: () -> Unit) = ApplicationManager.getApplication().invokeAndWait(body, ModalityState.any())
}

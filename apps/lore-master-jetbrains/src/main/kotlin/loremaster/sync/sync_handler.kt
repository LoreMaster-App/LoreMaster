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
import loremaster.configdescription.ConfigDescription
import loremaster.configdescription.describeConfig
import loremaster.configdescription.isPages
import loremaster.configdescription.isSite
import loremaster.connection.Credential
import loremaster.connection.signInCredential
import loremaster.engineprocess.EngineProcessStore
import loremaster.engineprocess.EngineRpcError
import loremaster.engineprogress.chooseOne
import loremaster.engineprogress.notify
import loremaster.pages.publishPages
import loremaster.pages.publishReport
import loremaster.projectsettings.OutputSettings
import loremaster.projectsettings.readProjectSettings
import java.nio.file.Path
import java.util.concurrent.ExecutionException

private const val UNAUTHORIZED = -32001

/**
 * A sync of the project's outputs, started from the IDE: the outputs [SyncTarget] names are synced one after the other, a plan
 * reviewed for each that syncs page by page and a site or wiki published, each ending in its own balloon.
 */
class StorageSync(private val project: Project) {
    private val gson = Gson()

    /** Starts the sync under a progress bar; [scope] limits it to files (workspace-relative), null is everything. */
    fun run(scope: List<String>?, target: SyncTarget, onFinished: () -> Unit = {}) {
        val root = project.basePath ?: return
        ProgressManager.getInstance().run(object : Task.Backgroundable(project, "Syncing", false) {
            override fun run(indicator: ProgressIndicator) {
                val store = ApplicationManager.getApplication().getService(EngineProcessStore::class.java)
                runCatching { syncOutputs(root, scope, target, store, indicator) }.onFailure { tell(root, Result.failure<SyncOutcome?>(it)) }
            }

            override fun onFinished() = onFinished()
        })
    }

    private fun syncOutputs(root: String, scope: List<String>?, target: SyncTarget, store: EngineProcessStore, indicator: ProgressIndicator) {
        val engine = store.connection()
        val outputs = readProjectSettings(engine, root).settings.outputs
        val description = describeConfig(engine)
        val syncable = outputs.withIndex().filter { canSync(description, it.value, target) }
        val chosen = when (val selection = selectOutputs(target, syncable)) {
            is SyncSelection.Nothing -> error(selection.reason)
            is SyncSelection.Run -> selection.outputs
            is SyncSelection.AskWhich -> listOfNotNull(chooseOutput(selection.outputs))
        }

        for (output in chosen) {
            if (description.isSite(output.value.platform)) {
                indicator.text = "Publishing ${output.value.label()}"
                notify(project, publishReport(output.value, publishPages(engine, root, output.index)))
            } else {
                tell(root, runCatching { syncOne(engine, store, root, output, scope, indicator) })
            }
        }
    }

    /** Sites and wikis publish as a whole, so they are left out of a sync that asks which output (the Tools menu's Sync). */
    private fun canSync(description: ConfigDescription, output: OutputSettings, target: SyncTarget): Boolean =
        description.isPages(output.platform) || (target !is SyncTarget.Choose && description.isSite(output.platform))

    private fun syncOne(engine: loremaster.engineprocess.EngineRpcClient, store: EngineProcessStore, root: String, output: IndexedValue<OutputSettings>, scope: List<String>?, indicator: ProgressIndicator): SyncOutcome? {
        val stopListening = store.listen { method, params ->
            if (method == METHOD_HOST_PROGRESS && params != null) {
                val step = gson.fromJson(params, ProgressParams::class.java)
                indicator.text = step.message
                if (step.total > 0) indicator.fraction = step.done.toDouble() / step.total
            }
        }
        try {
            val flow = RunSync(engine, { askCredential(it) }, { review(it, output.value) })

            return flow.run(root, output.value, output.index, scope)
        } finally {
            stopListening()
        }
    }

    private fun chooseOutput(outputs: List<IndexedValue<OutputSettings>>): IndexedValue<OutputSettings>? =
        onUiThread { chooseOne(project, "LoreMaster: Sync", "Which output?", outputs.map { it.value.label() })?.let(outputs::getOrNull) }

    private fun askCredential(baseUrl: String): Credential? = signInCredential(project, baseUrl)

    private fun review(plan: SyncPlanResult, output: OutputSettings): SyncChoice? {
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

    private fun tell(root: String, outcome: Result<SyncOutcome?>) {
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
            return "The platform refused the sign-in. Run LoreMaster: Connect to Confluence to enter it again."
        }

        return "The sync failed: ${cause.message ?: cause.javaClass.simpleName}"
    }

    private fun <T> onUiThread(body: () -> T): T {
        var result: T? = null
        ApplicationManager.getApplication().invokeAndWait({ result = body() }, ModalityState.any())

        @Suppress("UNCHECKED_CAST")
        return result as T
    }
}

/** "LoreMaster: Sync": the selected Markdown files, or the whole project, to an output that syncs page by page. */
class SyncHandler : AnAction() {
    override fun getActionUpdateThread() = ActionUpdateThread.BGT

    override fun update(event: AnActionEvent) {
        event.presentation.isEnabledAndVisible = event.project?.basePath != null
    }

    override fun actionPerformed(event: AnActionEvent) {
        val project = event.project ?: return
        val root = project.basePath ?: return
        val selected = event.getData(CommonDataKeys.VIRTUAL_FILE_ARRAY).orEmpty().flatMap(::markdownBelow).map { Path.of(it.path) }

        StorageSync(project).run(syncScope(Path.of(root), selected), SyncTarget.Choose)
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
}

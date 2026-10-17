package loremaster.syncview

import com.intellij.icons.AllIcons
import com.intellij.notification.NotificationType
import com.intellij.openapi.actionSystem.ActionManager
import com.intellij.openapi.actionSystem.ActionUpdateThread
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.actionSystem.DefaultActionGroup
import com.intellij.openapi.actionSystem.ToggleAction
import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.Messages
import loremaster.configdescription.describeConfig
import loremaster.connection.signInCredential
import loremaster.engineprocess.EngineProcessStore
import loremaster.engineprogress.TaskReport
import loremaster.engineprogress.notify
import loremaster.engineprogress.runEngineTask
import loremaster.generators.generateAndSync
import loremaster.outputs.AddOutputDialogComponent
import loremaster.outputs.removeOutput
import loremaster.outputs.replaceOutput
import loremaster.pages.checkPages
import loremaster.pages.checkReport
import loremaster.projectsettings.readRawOutputs
import loremaster.sync.StorageSync
import loremaster.sync.SyncTarget
import loremaster.sync.checkRemote
import loremaster.sync.describePlan
import javax.swing.Icon
import javax.swing.SwingUtilities

/**
 * The actions of the Sync view, in two groups: what applies to every storage (the view's toolbar) and what applies to the row
 * selected (the toolbar under it, and the right-click menu). A storage's row has all of its; Local only the display ones and
 * Refresh, since pages are not sent to it.
 */
class SyncViewActions(private val view: SyncViewComponent, private val project: Project) {
    /** The actions for every storage: add, config, refresh all, sync all, generate and sync all. */
    fun wholeViewGroup() = DefaultActionGroup(
        Whole("Add Sync Storage", "Choose where your Markdown goes and add it", AllIcons.General.Add) { delegate("loremaster.addOutput", it) },
        Whole("LoreMaster Config", "Open the project's LoreMaster settings", AllIcons.General.Settings) { delegate("loremaster.openProjectSettings", it) },
        Whole("Refresh All Storages", "Read every storage's pages again", AllIcons.Actions.Refresh) { view.reload() },
        Whole("Sync All Storages", "Sync every storage that is set up", AllIcons.Actions.Upload) { StorageSync(project).run(null, SyncTarget.All) },
        Whole("Generate and Sync All", "Run the generators, then sync every storage", AllIcons.Actions.Execute) { generateAndSync(project, SyncTarget.All) },
    )

    /** The actions of the selected row. */
    fun rowGroup() = DefaultActionGroup(
        RowAction("Check Remote Status", "Ask the platform where this storage stands, without writing anything", AllIcons.Actions.Checked, storageOnly = true) { checkRemote(it) },
        RowToggle("Show File Names", "Name pages by their file instead of their title", AllIcons.Actions.ShowAsTree, storageOnly = false,
            selected = { it.display.fileNames }, change = { it.copy(fileNames = !it.fileNames) }),
        RowToggle("Flat View", "List every page by path instead of as a tree", AllIcons.Actions.GroupByPackage, storageOnly = false,
            selected = { it.display.flat }, change = { it.copy(flat = !it.flat) }),
        RowToggle("Hide Ignored", "Do not list the files this storage leaves out", AllIcons.Actions.ToggleVisibility, storageOnly = true,
            selected = { it.display.hideIgnored }, change = { it.copy(hideIgnored = !it.hideIgnored) }),
        RowAction("Refresh Storage", "Read this location's pages again", AllIcons.Actions.Refresh, storageOnly = false) { view.reload(it.index) },
        RowAction("Sync Storage", "Sync this storage", AllIcons.Actions.Upload, storageOnly = true) { StorageSync(project).run(null, SyncTarget.Only(listOf(it.index))) },
        RowAction("Generate and Sync", "Run the generators, then sync this storage", AllIcons.Actions.Execute, storageOnly = true) {
            generateAndSync(project, SyncTarget.Only(listOf(it.index)))
        },
        RowAction("Edit Storage", "Change this storage's settings", AllIcons.Actions.Edit, storageOnly = true) { edit(it) },
        RowAction("Remove Storage", "Remove this storage from the project's settings", AllIcons.General.Remove, storageOnly = true) { remove(it) },
    )

    private fun delegate(actionId: String, event: AnActionEvent) {
        val action = ActionManager.getInstance().getAction(actionId) ?: return
        ActionManager.getInstance().tryToExecute(action, null, view, event.place, true)
    }

    /** Asks the platform where a storage stands: a plan for one that syncs page by page, a comparison for a site. */
    private fun checkRemote(row: Row.LocationRow) {
        val storage = view.storages.firstOrNull { it.index == row.index } ?: return
        val root = project.basePath ?: return
        runEngineTask(project, "Checking ${row.label}") { engine, _ ->
            if (view.isSite(row.index)) {
                checkReport(storage.output, checkPages(engine, root, row.index))
            } else {
                val plan = checkRemote(engine, { signInCredential(project, it) }, root, storage.output, row.index)
                    ?: return@runEngineTask TaskReport("The check was cancelled.")
                TaskReport(describePlan(row.label, plan), if (plan.errors.isNullOrEmpty()) NotificationType.INFORMATION else NotificationType.ERROR)
            }
        }
    }

    /** Opens the storage's settings in the dialog that added it; a save replaces it in place, keeping what the form does not show. */
    private fun edit(row: Row.LocationRow) {
        val root = project.basePath ?: return
        ApplicationManager.getApplication().executeOnPooledThread {
            val outcome = runCatching {
                val engine = ApplicationManager.getApplication().getService(EngineProcessStore::class.java).connection()
                Triple(engine, describeConfig(engine), readRawOutputs(engine, root).getOrNull(row.index))
            }
            SwingUtilities.invokeLater {
                val (engine, description, existing) = outcome.getOrElse {
                    Messages.showErrorDialog(project, "The engine did not answer: ${it.message}", "LoreMaster")

                    return@invokeLater
                }
                if (existing == null) {
                    Messages.showErrorDialog(project, "That storage is no longer in the project's settings.", "LoreMaster")

                    return@invokeLater
                }
                val dialog = AddOutputDialogComponent(project, engine, description, existing)
                if (!dialog.showAndGet()) return@invokeLater
                val saved = dialog.output()
                runEngineTask(project, "Saving ${row.label}") { client, _ ->
                    replaceOutput(client, root, row.index, saved)
                    TaskReport("Saved ${row.label}.")
                }
            }
        }
    }

    private fun remove(row: Row.LocationRow) {
        val root = project.basePath ?: return
        val answer = Messages.showYesNoDialog(project, "Remove ${row.label} from the project's LoreMaster settings?", "LoreMaster: Remove Storage", "Remove", "Cancel", Messages.getWarningIcon())
        if (answer != Messages.YES) return
        runEngineTask(project, "Removing ${row.label}") { engine, _ ->
            removeOutput(engine, root, row.index)
            TaskReport("Removed ${row.label}. The pages it made on the platform are left where they are.")
        }
    }

    /** An action of the whole view. */
    private class Whole(text: String, description: String, icon: Icon, private val run: (AnActionEvent) -> Unit) : AnAction(text, description, icon) {
        override fun getActionUpdateThread() = ActionUpdateThread.EDT

        override fun actionPerformed(event: AnActionEvent) = run(event)
    }

    /** An action of the selected row: shown for a storage, or for any location when [storageOnly] is false. */
    private inner class RowAction(text: String, description: String, icon: Icon, private val storageOnly: Boolean, private val run: (Row.LocationRow) -> Unit) :
        AnAction(text, description, icon) {
        override fun getActionUpdateThread() = ActionUpdateThread.EDT

        override fun update(event: AnActionEvent) {
            val row = view.selectedLocation
            event.presentation.isEnabledAndVisible = row != null && (!storageOnly || !row.isLocal)
        }

        override fun actionPerformed(event: AnActionEvent) {
            view.selectedLocation?.let(run)
        }
    }

    /** A display choice of the selected row, shown checked when it is on. */
    private inner class RowToggle(
        text: String,
        description: String,
        icon: Icon,
        private val storageOnly: Boolean,
        private val selected: (Row.LocationRow) -> Boolean,
        private val change: (LocationDisplay) -> LocationDisplay,
    ) : ToggleAction(text, description, icon) {
        override fun getActionUpdateThread() = ActionUpdateThread.EDT

        override fun update(event: AnActionEvent) {
            super.update(event)
            val row = view.selectedLocation
            event.presentation.isEnabledAndVisible = row != null && (!storageOnly || !row.isLocal)
        }

        override fun isSelected(event: AnActionEvent): Boolean = view.selectedLocation?.let(selected) ?: false

        override fun setSelected(event: AnActionEvent, state: Boolean) {
            val row = view.selectedLocation ?: return
            project.getService(LocationDisplayStore::class.java).change(row.index, change)
            view.redraw()
        }
    }
}

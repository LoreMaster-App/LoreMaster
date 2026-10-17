package loremaster.watch

import com.intellij.notification.NotificationType
import com.intellij.openapi.actionSystem.ActionUpdateThread
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.actionSystem.ToggleAction
import com.intellij.openapi.application.ApplicationManager
import loremaster.engineprogress.TaskReport
import loremaster.engineprogress.notify

/** "LoreMaster: Watch Mode": a checked menu item that turns watch mode on and off for the project. */
class ToggleWatchHandler : ToggleAction() {
    override fun getActionUpdateThread() = ActionUpdateThread.BGT

    override fun isSelected(event: AnActionEvent): Boolean = event.project?.getService(WatchSessionStore::class.java)?.active == true

    override fun setSelected(event: AnActionEvent, selected: Boolean) {
        val project = event.project ?: return
        val store = project.getService(WatchSessionStore::class.java)
        if (!selected) {
            store.stop()
            notify(project, TaskReport("Watch mode is off."))

            return
        }
        ApplicationManager.getApplication().executeOnPooledThread {
            val problem = runCatching { store.start() }.getOrElse { it.message ?: it.javaClass.simpleName }
            notify(
                project,
                if (problem == null) TaskReport("LoreMaster is watching: changes sync on their own after a moment of quiet. Run Watch Mode again to stop.")
                else TaskReport("Watch mode did not start: $problem", NotificationType.WARNING),
            )
        }
    }
}

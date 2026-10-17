package loremaster.outputs

import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.application.ApplicationManager
import loremaster.engineprocess.EngineProcessStore
import loremaster.engineprogress.TaskReport
import loremaster.engineprogress.runEngineTask

/**
 * "LoreMaster: Add Confluence Output...": signs in, lists the spaces, lets the user browse or search for the parent page, and adds
 * the output to the project's `loreMaster.outputs` (in `.vscode/settings.json`), ready for a sync.
 */
class AddOutputHandler : AnAction() {
    override fun actionPerformed(event: AnActionEvent) {
        val project = event.project ?: return
        val root = project.basePath ?: return
        val engine = ApplicationManager.getApplication().getService(EngineProcessStore::class.java).connection()
        val dialog = OutputPickerDialogComponent(project, engine)
        if (!dialog.showAndGet()) return
        val choice = dialog.choice()

        runEngineTask(project, "Adding the output") { client, _ ->
            val index = addConfluenceOutput(client, root, choice)

            TaskReport("Added output #${index + 1}: Confluence space ${choice.space}, title prefix \"${choice.titlePrefix}\". Run LoreMaster: Sync to Confluence.")
        }
    }
}

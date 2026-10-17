package loremaster.outputs

import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.ui.Messages
import loremaster.configdescription.describeConfig
import loremaster.engineprocess.EngineProcessStore
import loremaster.engineprogress.TaskReport
import loremaster.engineprogress.runEngineTask

/**
 * "LoreMaster: Add Output...": offers the platforms the engine describes, draws the chosen one's fields (signing in and picking a
 * space or a parent page where the platform has them), and adds the output to the project's `loreMaster.outputs` (in
 * `.vscode/settings.json`), ready for a sync.
 */
class AddOutputHandler : AnAction() {
    override fun actionPerformed(event: AnActionEvent) {
        val project = event.project ?: return
        val root = project.basePath ?: return
        val engine = ApplicationManager.getApplication().getService(EngineProcessStore::class.java).connection()
        val description = runCatching { describeConfig(engine) }.getOrElse {
            Messages.showErrorDialog(project, "The engine did not say what can be configured: ${it.message}", "LoreMaster")

            return
        }
        val dialog = AddOutputDialogComponent(project, engine, description)
        if (!dialog.showAndGet()) return
        val output = dialog.output()
        val platform = dialog.platform

        runEngineTask(project, "Adding the output") { client, _ ->
            val index = addOutput(client, root, description, output)

            TaskReport("Added output #${index + 1} (${platform.label}). Run LoreMaster: Sync.")
        }
    }
}

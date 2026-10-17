package loremaster.generators

import com.intellij.notification.NotificationType
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.ui.Messages
import loremaster.engineprogress.TaskReport
import loremaster.engineprogress.runEngineTask

/**
 * "LoreMaster: Run Generators": writes Markdown from test reports and source documentation. A generator's prepare command is a
 * shell command from the project's settings, so it only runs when the user says so.
 */
class RunGeneratorsHandler : AnAction() {
    override fun actionPerformed(event: AnActionEvent) {
        val project = event.project ?: return
        val root = project.basePath ?: return
        val answer = Messages.showYesNoCancelDialog(
            project,
            "Run each generator's prepare command first (a shell command from loreMaster.generators)?",
            "LoreMaster: Run Generators",
            "Run prepare commands",
            "Only generate",
            "Cancel",
            Messages.getQuestionIcon(),
        )
        if (answer == Messages.CANCEL) {
            return
        }
        runEngineTask(project, "Running the generators") { engine, _ ->
            val result = runGenerators(engine, root, prepare = answer == Messages.YES)
            val failed = result.runs.any { it.error != null }

            TaskReport(describeRuns(result), if (failed) NotificationType.WARNING else NotificationType.INFORMATION)
        }
    }
}

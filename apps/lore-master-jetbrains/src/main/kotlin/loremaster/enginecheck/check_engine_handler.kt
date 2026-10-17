package loremaster.enginecheck

import com.intellij.notification.NotificationGroupManager
import com.intellij.notification.NotificationType
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.progress.ProgressIndicator
import com.intellij.openapi.progress.ProgressManager
import com.intellij.openapi.progress.Task
import com.intellij.openapi.project.Project
import loremaster.engineprocess.EngineProcessStore
import loremaster.engineprocess.METHOD_PING
import loremaster.engineprocess.PingResult
import java.util.concurrent.TimeUnit

/** "LoreMaster: Check Engine": starts the engine if needed, pings it and says what answered. */
class CheckEngineHandler : AnAction() {
    override fun actionPerformed(event: AnActionEvent) {
        val project = event.project ?: return
        ProgressManager.getInstance().run(object : Task.Backgroundable(project, "Checking the LoreMaster engine", false) {
            override fun run(indicator: ProgressIndicator) {
                val outcome = runCatching {
                    ApplicationManager.getApplication().getService(EngineProcessStore::class.java)
                        .connection()
                        .request(METHOD_PING, null, PingResult::class.java)
                        .get(15, TimeUnit.SECONDS)
                }
                tell(project, outcome)
            }
        })
    }

    private fun tell(project: Project, outcome: Result<PingResult>) {
        val message = outcome.fold(
            onSuccess = { "The engine answered: ${it.pong} (version ${it.version})." },
            onFailure = { "The engine did not answer: ${it.message ?: it.javaClass.simpleName}" },
        )
        val type = if (outcome.isSuccess) NotificationType.INFORMATION else NotificationType.ERROR
        NotificationGroupManager.getInstance().getNotificationGroup("LoreMaster")
            .createNotification("LoreMaster", message, type)
            .notify(project)
    }
}

package loremaster.connection

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

/** "LoreMaster: Connect to Confluence": asks for a site and credential, checks them with the engine and remembers them. */
class ConnectHandler : AnAction() {
    override fun actionPerformed(event: AnActionEvent) {
        val project = event.project ?: return
        val form = ConnectionForm()
        if (!ConnectionDialogComponent(project, form).showAndGet()) {
            return
        }
        verifyAndRemember(project, form.baseUrl, form.credential())
    }

    private fun verifyAndRemember(project: Project, baseUrl: String, credential: Credential) {
        ProgressManager.getInstance().run(object : Task.Backgroundable(project, "Connecting to $baseUrl", false) {
            override fun run(indicator: ProgressIndicator) {
                val store = ApplicationManager.getApplication().getService(EngineProcessStore::class.java)
                val outcome = runCatching {
                    val session = openSession(store.connection(), baseUrl, credential)
                    closeSession(store.connection(), session.sessionId)
                    session
                }
                outcome.onSuccess { CredentialVaultClient.save(it.baseUrl.ifBlank { baseUrl }, credential) }
                val group = NotificationGroupManager.getInstance().getNotificationGroup("LoreMaster")
                outcome.fold(
                    onSuccess = {
                        group.createNotification("LoreMaster", "Connected to ${it.baseUrl} (${it.edition}) as ${it.user.displayName}.", NotificationType.INFORMATION)
                            .notify(project)
                    },
                    onFailure = {
                        group.createNotification("LoreMaster", "Could not connect: ${it.cause?.message ?: it.message}", NotificationType.ERROR)
                            .notify(project)
                    },
                )
            }
        })
    }
}

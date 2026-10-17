package loremaster.engineprogress

import com.intellij.notification.NotificationGroupManager
import com.intellij.notification.NotificationType
import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.application.ModalityState
import com.intellij.openapi.progress.ProgressIndicator
import com.intellij.openapi.progress.ProgressManager
import com.intellij.openapi.progress.Task
import com.intellij.openapi.project.Project
import loremaster.engineprocess.EngineProcessStore
import loremaster.engineprocess.EngineRpcClient
import java.util.concurrent.ExecutionException

/** What a task says when it ends. */
data class TaskReport(val message: String, val type: NotificationType = NotificationType.INFORMATION)

/**
 * Runs [work] on a background thread with the engine, under a progress bar, and shows its report in a balloon. A failure
 * becomes an error balloon with the engine's own message.
 */
fun runEngineTask(project: Project, title: String, work: (engine: EngineRpcClient, indicator: ProgressIndicator) -> TaskReport) {
    ProgressManager.getInstance().run(object : Task.Backgroundable(project, title, false) {
        override fun run(indicator: ProgressIndicator) {
            val store = ApplicationManager.getApplication().getService(EngineProcessStore::class.java)
            val report = try {
                work(store.connection(), indicator)
            } catch (failure: Exception) {
                val cause = (failure as? ExecutionException)?.cause ?: failure
                TaskReport("$title failed: ${cause.message ?: cause.javaClass.simpleName}", NotificationType.ERROR)
            }
            notify(project, report)
        }
    })
}

/** Shows a report as a balloon. */
fun notify(project: Project, report: TaskReport) {
    NotificationGroupManager.getInstance().getNotificationGroup("LoreMaster")
        .createNotification("LoreMaster", report.message, report.type)
        .notify(project)
}

/** Runs [body] on the UI thread and waits for it, for a dialog asked from a background task. */
fun <T> onUiThread(body: () -> T): T {
    var result: T? = null
    ApplicationManager.getApplication().invokeAndWait({ result = body() }, ModalityState.any())

    @Suppress("UNCHECKED_CAST")
    return result as T
}

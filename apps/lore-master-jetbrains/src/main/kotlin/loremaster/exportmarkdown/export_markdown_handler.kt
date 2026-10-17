package loremaster.exportmarkdown

import com.intellij.ide.actions.RevealFileAction
import com.intellij.notification.Notification
import com.intellij.notification.NotificationAction
import com.intellij.notification.NotificationGroupManager
import com.intellij.notification.NotificationType
import com.intellij.openapi.actionSystem.ActionUpdateThread
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.actionSystem.CommonDataKeys
import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.fileEditor.FileDocumentManager
import com.intellij.openapi.progress.ProgressIndicator
import com.intellij.openapi.progress.ProgressManager
import com.intellij.openapi.progress.Task
import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.popup.JBPopupFactory
import com.intellij.openapi.vfs.VirtualFile
import loremaster.engineprocess.EngineProcessStore
import loremaster.pluginsettings.PluginSettingsStore
import java.nio.file.Path

/** "LoreMaster: Export Markdown As...": pick a format, the engine writes the file next to the document. */
class ExportMarkdownHandler : AnAction() {
    override fun getActionUpdateThread() = ActionUpdateThread.BGT

    override fun update(event: AnActionEvent) {
        event.presentation.isEnabledAndVisible = event.project != null && isMarkdown(event.getData(CommonDataKeys.VIRTUAL_FILE))
    }

    override fun actionPerformed(event: AnActionEvent) {
        val project = event.project ?: return
        val file = event.getData(CommonDataKeys.VIRTUAL_FILE)?.takeIf(::isMarkdown) ?: return
        val unsaved = FileDocumentManager.getInstance().getCachedDocument(file)?.text

        ProgressManager.getInstance().run(object : Task.Backgroundable(project, "Loading the LoreMaster export formats", false) {
            override fun run(indicator: ProgressIndicator) {
                val formats = runCatching { listExportFormats(engine()) }
                ApplicationManager.getApplication().invokeLater {
                    formats.fold(
                        onSuccess = { pick(project, file, unsaved, it) },
                        onFailure = { tell(project, "The engine did not answer: ${it.message}", NotificationType.ERROR) },
                    )
                }
            }
        })
    }

    private fun pick(project: Project, file: VirtualFile, unsaved: String?, formats: List<ExportFormat>) {
        JBPopupFactory.getInstance()
            .createPopupChooserBuilder(formats)
            .setTitle("Export ${file.name} as")
            .setRenderer(javax.swing.DefaultListCellRenderer().let { base ->
                javax.swing.ListCellRenderer<ExportFormat> { list, value, index, selected, focus ->
                    base.getListCellRendererComponent(list, "${value.label}  (${value.extension})", index, selected, focus)
                }
            })
            .setItemChosenCallback { run(project, file, unsaved, it) }
            .createPopup()
            .showCenteredInCurrentWindow(project)
    }

    private fun run(project: Project, file: VirtualFile, unsaved: String?, format: ExportFormat) {
        ProgressManager.getInstance().run(object : Task.Backgroundable(project, "Exporting ${file.name} as ${format.label}", false) {
            override fun run(indicator: ProgressIndicator) {
                val outcome = runCatching {
                    exportMarkdown(engine(), exportParameters(project, file, format, unsaved))
                }
                outcome.fold(
                    onSuccess = { done(project, it) },
                    onFailure = { tell(project, "The export failed: ${it.cause?.message ?: it.message}", NotificationType.ERROR) },
                )
            }
        })
    }

    private fun exportParameters(project: Project, file: VirtualFile, format: ExportFormat, unsaved: String?): ExportRunParams {
        val settings = PluginSettingsStore.current()
        val document = Path.of(file.path)
        val template = when (format.id) {
            "docx" -> settings.value("export.docxTemplate")
            "odt" -> settings.value("export.odtTemplate")
            else -> ""
        }

        return ExportRunParams(
            path = file.path,
            format = format.id,
            content = unsaved,
            outDir = exportDestination(document, project.basePath?.let(Path::of), settings.value("export.folder"))?.toString(),
            browserPath = settings.value("export.browserPath").ifBlank { null },
            referencePath = template.ifBlank { null },
            pageSize = settings.value("export.pageSize").ifBlank { null },
            imageWidth = settings.value("export.pngWidth").toIntOrNull()?.takeIf { it > 0 },
        )
    }

    private fun done(project: Project, result: ExportRunResult) {
        val notification = notification("Exported ${result.name}", NotificationType.INFORMATION)
        notification.addAction(NotificationAction.createSimple("Show in folder") { RevealFileAction.openFile(Path.of(result.path)) })
        notification.notify(project)
    }

    private fun tell(project: Project, message: String, type: NotificationType) = notification(message, type).notify(project)

    private fun notification(message: String, type: NotificationType): Notification =
        NotificationGroupManager.getInstance().getNotificationGroup("LoreMaster").createNotification("LoreMaster", message, type)

    private fun engine() = ApplicationManager.getApplication().getService(EngineProcessStore::class.java).connection()

    private fun isMarkdown(file: VirtualFile?) = file != null && !file.isDirectory && file.extension.let { it == "md" || it == "markdown" }
}

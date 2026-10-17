package loremaster.pages

import com.intellij.notification.NotificationType
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.fileChooser.FileChooser
import com.intellij.openapi.fileChooser.FileChooserDescriptorFactory
import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.Messages
import loremaster.engineprocess.EngineRpcClient
import loremaster.engineprogress.TaskReport
import loremaster.engineprogress.chooseOne
import loremaster.engineprogress.runEngineTask
import loremaster.projectsettings.OutputSettings
import loremaster.projectsettings.readProjectSettings

private val SITE_PLATFORMS = setOf("github-pages", "github-wiki")

/** What a Pages action does once its output is chosen. */
typealias PagesStep = (engine: EngineRpcClient, root: String, index: Int, output: OutputSettings) -> TaskReport

private fun withWarnings(text: String, warnings: List<String>?): String = text + warnings.orEmpty().joinToString("") { "\nWarning: $it" }

/** Common to the Pages actions: find the github-pages / github-wiki output (asking when there are several), then do the step. */
abstract class PagesAction(private val title: String) : AnAction() {
    /** Asks what the step needs from the user (on the UI thread, before the work starts), and returns the step. */
    protected abstract fun prepare(project: Project): PagesStep?

    override fun actionPerformed(event: AnActionEvent) {
        val project = event.project ?: return
        val root = project.basePath ?: return
        val step = prepare(project) ?: return
        runEngineTask(project, title) { engine, _ ->
            val outputs = readProjectSettings(engine, root).settings.outputs.withIndex().filter { it.value.platform in SITE_PLATFORMS }
            if (outputs.isEmpty()) {
                return@runEngineTask TaskReport("No github-pages or github-wiki output is configured in loreMaster.outputs.", NotificationType.WARNING)
            }
            val chosen = if (outputs.size == 1) outputs.single() else choose(project, outputs)

            chosen?.let { step(engine, root, it.index, it.value) } ?: TaskReport("Cancelled.")
        }
    }

    private fun choose(project: Project, outputs: List<IndexedValue<OutputSettings>>): IndexedValue<OutputSettings>? {
        return chooseOne(project, title, "Which output?", outputs.map { it.value.label() })?.let(outputs::getOrNull)
    }
}

/** "LoreMaster: Check GitHub Pages": what a publish would add, change or remove. */
class CheckPagesHandler : PagesAction("Checking GitHub Pages") {
    override fun prepare(project: Project): PagesStep = { engine, root, index, output ->
        val result = checkPages(engine, root, index)
        when {
            !result.errors.isNullOrEmpty() ->
                TaskReport("${output.label()} cannot be checked:\n${result.errors.joinToString("\n")}", NotificationType.ERROR)
            result.upToDate -> TaskReport(withWarnings("${output.label()} is up to date (${result.files} files).", result.warnings))
            else -> {
                val newPages = if (result.newPages > 0) ", ${result.newPages} page(s) new to the public site" else ""
                TaskReport(withWarnings("${output.label()}: a publish would change ${result.changesTotal} file(s)$newPages.", result.warnings), NotificationType.WARNING)
            }
        }
    }
}

/** "LoreMaster: Build GitHub Pages Site...": the static site into a folder you pick; no git, no network. */
class BuildPagesHandler : PagesAction("Building the site") {
    override fun prepare(project: Project): PagesStep? {
        val descriptor = FileChooserDescriptorFactory.createSingleFolderDescriptor().withTitle("Folder for the site")
        val folder = FileChooser.chooseFile(descriptor, project, null)?.path ?: return null

        return { engine, root, index, output ->
            val result = buildPages(engine, root, index, folder)
            if (!result.errors.isNullOrEmpty()) {
                TaskReport("${output.label()} was not built:\n${result.errors.joinToString("\n")}", NotificationType.ERROR)
            } else {
                TaskReport(withWarnings("${output.label()}: ${result.files} files written to ${result.outDir}.", result.warnings))
            }
        }
    }
}

/** "LoreMaster: Publish GitHub Pages": pushes the site to the output's branch with your own git. Asks first. */
class PublishPagesHandler : PagesAction("Publishing GitHub Pages") {
    override fun prepare(project: Project): PagesStep? {
        val answer = Messages.showYesNoDialog(project, "Publish the site to its branch with git now?", "LoreMaster: Publish", Messages.getQuestionIcon())
        if (answer != Messages.YES) {
            return null
        }

        return { engine, root, index, output ->
            val result = publishPages(engine, root, index)
            when {
                !result.errors.isNullOrEmpty() ->
                    TaskReport("${output.label()} was not published:\n${result.errors.joinToString("\n")}", NotificationType.ERROR)
                !result.changed -> TaskReport(withWarnings("${output.label()} already matches ${result.branch}.", result.warnings))
                else -> {
                    val where = result.url?.let { ": $it" }.orEmpty()
                    TaskReport(withWarnings("Published ${output.label()} to ${result.branch} (${result.commit}, ${result.files} files)$where.", result.warnings))
                }
            }
        }
    }
}

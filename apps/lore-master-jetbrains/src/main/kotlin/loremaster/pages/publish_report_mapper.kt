package loremaster.pages

import com.intellij.notification.NotificationType
import loremaster.engineprogress.TaskReport
import loremaster.projectsettings.OutputSettings

/** A line per warning after the text. */
fun withWarnings(text: String, warnings: List<String>?): String = text + warnings.orEmpty().joinToString("") { "\nWarning: $it" }

/** What a publish of [output] says when it ends: the engine's errors, that nothing changed, or what was pushed where. */
fun publishReport(output: OutputSettings, result: PagesPublishResult): TaskReport = when {
    !result.errors.isNullOrEmpty() ->
        TaskReport("${output.label()} was not published:\n${result.errors.joinToString("\n")}", NotificationType.ERROR)
    !result.changed -> TaskReport(withWarnings("${output.label()} already matches ${result.branch}.", result.warnings))
    else -> {
        val where = result.url?.let { ": $it" }.orEmpty()
        TaskReport(withWarnings("Published ${output.label()} to ${result.branch} (${result.commit}, ${result.files} files)$where.", result.warnings))
    }
}

/** What a check of [output] says: the engine's errors, that the site is up to date, or how much a publish would change. */
fun checkReport(output: OutputSettings, result: PagesCheckResult): TaskReport = when {
    !result.errors.isNullOrEmpty() ->
        TaskReport("${output.label()} cannot be checked:\n${result.errors.joinToString("\n")}", NotificationType.ERROR)
    result.upToDate -> TaskReport(withWarnings("${output.label()} is up to date (${result.files} files).", result.warnings))
    else -> {
        val newPages = if (result.newPages > 0) ", ${result.newPages} page(s) new to the public site" else ""
        TaskReport(withWarnings("${output.label()}: a publish would change ${result.changesTotal} file(s)$newPages.", result.warnings), NotificationType.WARNING)
    }
}

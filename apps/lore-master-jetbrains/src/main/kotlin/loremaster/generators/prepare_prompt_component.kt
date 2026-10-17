package loremaster.generators

import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.Messages

/**
 * Asks whether to run each generator's prepare command first (a shell command from the project's settings, so it only runs
 * when the user says so). True runs them, false only generates, null cancels.
 */
fun askToPrepare(project: Project, title: String): Boolean? {
    val answer = Messages.showYesNoCancelDialog(
        project,
        "Run each generator's prepare command first (a shell command from loreMaster.generators)?",
        title,
        "Run prepare commands",
        "Only generate",
        "Cancel",
        Messages.getQuestionIcon(),
    )

    return when (answer) {
        Messages.YES -> true
        Messages.NO -> false
        else -> null
    }
}

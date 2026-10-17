package loremaster.engineprogress

import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.DialogWrapper
import com.intellij.ui.components.JBLabel
import com.intellij.ui.components.JBList
import com.intellij.ui.components.JBScrollPane
import com.intellij.util.ui.JBUI
import java.awt.BorderLayout
import javax.swing.JComponent
import javax.swing.JPanel
import javax.swing.ListSelectionModel

/** A small dialog that asks for one of [labels]. */
private class ChooseOneDialog(project: Project?, title: String, private val prompt: String, labels: List<String>) : DialogWrapper(project) {
    private val list = JBList(labels).apply {
        selectionMode = ListSelectionModel.SINGLE_SELECTION
        selectedIndex = 0
    }

    init {
        this.title = title
        init()
    }

    fun index(): Int = list.selectedIndex

    override fun createCenterPanel(): JComponent = JPanel(BorderLayout(0, JBUI.scale(6))).apply {
        add(JBLabel(prompt), BorderLayout.NORTH)
        add(JBScrollPane(list).apply { preferredSize = JBUI.size(360, 140) }, BorderLayout.CENTER)
    }

    override fun getPreferredFocusedComponent(): JComponent = list
}

/** Asks which of [labels] (on the UI thread, from any thread) and returns its index, or null when the user cancels. */
fun chooseOne(project: Project?, title: String, prompt: String, labels: List<String>): Int? = onUiThread {
    val dialog = ChooseOneDialog(project, title, prompt, labels)

    if (dialog.showAndGet()) dialog.index().takeIf { it >= 0 } else null
}

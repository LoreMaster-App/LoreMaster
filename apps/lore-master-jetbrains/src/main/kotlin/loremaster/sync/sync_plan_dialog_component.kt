package loremaster.sync

import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.DialogWrapper
import com.intellij.ui.components.JBCheckBox
import com.intellij.ui.components.JBLabel
import com.intellij.ui.components.JBScrollPane
import com.intellij.ui.table.JBTable
import com.intellij.util.ui.JBUI
import java.awt.BorderLayout
import javax.swing.JComponent
import javax.swing.JPanel
import javax.swing.table.DefaultTableModel

private val ORDER = listOf("create", "adopt", "update", "move", "rename_title", "conflict", "orphan", "unchanged")

/** Summarises a plan in one line: "2 create, 1 update, 5 unchanged". */
fun summarise(plan: SyncPlanResult): String {
    val parts = ORDER.mapNotNull { kind -> plan.counts[kind]?.takeIf { it > 0 }?.let { "$it $kind" } }
    return if (parts.isEmpty()) "Nothing to do." else parts.joinToString(", ")
}

/** How many actions change something on the platform. */
fun changeCount(plan: SyncPlanResult): Int = plan.actions.count { it.kind != "unchanged" && it.kind != "conflict" }

/** Shows what a sync would do, so the author can say yes. Conflicts are pages edited on the platform since the last sync. */
class SyncPlanDialogComponent(project: Project?, private val plan: SyncPlanResult, private val destination: String) : DialogWrapper(project) {
    private val force = JBCheckBox("Overwrite pages edited on the platform since the last sync")
    private val prune = JBCheckBox("Move pages that no longer have a Markdown file to the trash")

    init {
        title = "LoreMaster: Sync to $destination"
        setOKButtonText("Sync")
        init()
    }

    fun choice() = SyncChoice(force.isSelected, prune.isSelected)

    override fun createCenterPanel(): JComponent {
        val model = object : DefaultTableModel(arrayOf("Action", "Page", "File", "Changes"), 0) {
            override fun isCellEditable(row: Int, column: Int) = false
        }
        plan.actions.filter { it.kind != "unchanged" }.forEach {
            model.addRow(arrayOf(it.kind, it.title, it.path.orEmpty(), (it.changes.orEmpty() + listOfNotNull(it.reason)).joinToString(", ")))
        }
        val notes = plan.warnings.orEmpty().joinToString("\n") { "Warning: $it" }
        val top = JPanel(BorderLayout()).apply {
            add(JBLabel("<html><b>${summarise(plan)}</b></html>"), BorderLayout.NORTH)
            if (notes.isNotEmpty()) add(JBLabel("<html>${notes.replace("\n", "<br>")}</html>"), BorderLayout.CENTER)
        }
        val options = JPanel(BorderLayout()).apply {
            if ((plan.counts["conflict"] ?: 0) > 0) add(force, BorderLayout.NORTH)
            if ((plan.counts["orphan"] ?: 0) > 0) add(prune, BorderLayout.SOUTH)
        }

        return JPanel(BorderLayout(0, JBUI.scale(8))).apply {
            preferredSize = JBUI.size(720, 360)
            add(top, BorderLayout.NORTH)
            add(JBScrollPane(JBTable(model)), BorderLayout.CENTER)
            add(options, BorderLayout.SOUTH)
        }
    }
}

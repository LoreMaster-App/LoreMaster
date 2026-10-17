package loremaster.outputs

import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.DialogWrapper
import com.intellij.ui.components.JBLabel
import com.intellij.ui.components.JBList
import com.intellij.ui.components.JBScrollPane
import com.intellij.ui.components.JBTextField
import com.intellij.util.ui.JBUI
import loremaster.engineprocess.EngineRpcClient
import java.awt.BorderLayout
import java.awt.event.MouseAdapter
import java.awt.event.MouseEvent
import javax.swing.DefaultListModel
import javax.swing.JButton
import javax.swing.JComponent
import javax.swing.JPanel
import javax.swing.ListSelectionModel
import javax.swing.SwingUtilities

/**
 * Browses a space's pages to pick the parent the project's pages go under: type to search by title, double-click a page to look at
 * its children, "Up" to go back. [home] (the space's home page) is where it starts and the default choice.
 */
class ParentPageDialogComponent(
    project: Project?,
    private val engine: EngineRpcClient,
    private val sessionId: String,
    private val space: Space,
    private val home: RemotePage?,
) : DialogWrapper(project) {
    private val model = DefaultListModel<RemotePage>()
    private val list = JBList(model).apply { selectionMode = ListSelectionModel.SINGLE_SELECTION }
    private val query = JBTextField()
    private val where = JBLabel(" ")
    private val trail = ArrayDeque<RemotePage>()

    /** The pages the list shows now. */
    val shownPages: List<RemotePage> get() = model.elements().toList()

    /** The page chosen, or the home page when nothing is selected. */
    var chosen: RemotePage? = home
        private set

    init {
        title = "Parent page in ${space.name}"
        setOKButtonText("Use this page")
        init()
        home?.let { open(it) } ?: search("")
    }

    override fun createCenterPanel(): JComponent {
        val search = JButton("Search").apply { addActionListener { search(query.text.trim()) } }
        val up = JButton("Up").apply { addActionListener { goUp() } }
        val top = JPanel(BorderLayout(JBUI.scale(4), 0)).apply {
            add(query, BorderLayout.CENTER)
            add(JPanel().apply { add(search); add(up) }, BorderLayout.EAST)
        }
        query.addActionListener { search(query.text.trim()) }
        list.addListSelectionListener { chosen = list.selectedValue ?: chosen }
        list.addMouseListener(object : MouseAdapter() {
            override fun mouseClicked(event: MouseEvent) {
                if (event.clickCount == 2) list.selectedValue?.let(::open)
            }
        })

        return JPanel(BorderLayout(0, JBUI.scale(6))).apply {
            preferredSize = JBUI.size(520, 380)
            add(top, BorderLayout.NORTH)
            add(JBScrollPane(list), BorderLayout.CENTER)
            add(where, BorderLayout.SOUTH)
        }
    }

    private fun open(page: RemotePage) {
        trail.addLast(page)
        chosen = page
        show("Below ${page.title} (double-click to go deeper; the page you select is the parent)") { pageChildren(engine, sessionId, page.id) }
    }

    private fun goUp() {
        if (trail.size <= 1) return
        trail.removeLast()
        val page = trail.removeLast()
        open(page)
    }

    private fun search(text: String) {
        trail.clear()
        show(if (text.isEmpty()) "Pages of ${space.name}" else "Pages of ${space.name} matching \"$text\"") { searchPages(engine, sessionId, space.key, text) }
    }

    private fun show(caption: String, fetch: () -> List<RemotePage>) {
        where.text = "Loading..."
        ApplicationManager.getApplication().executeOnPooledThread {
            val result = runCatching(fetch)
            SwingUtilities.invokeLater {
                model.clear()
                result.onSuccess { pages -> pages.forEach(model::addElement); where.text = "$caption: ${pages.size}" }
                    .onFailure { where.text = "The lookup failed: ${it.cause?.message ?: it.message}" }
            }
        }
    }
}

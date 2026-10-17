package loremaster.pagestree

import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.fileEditor.OpenFileDescriptor
import com.intellij.openapi.project.Project
import com.intellij.openapi.vfs.LocalFileSystem
import com.intellij.openapi.wm.ToolWindow
import com.intellij.openapi.wm.ToolWindowFactory
import com.intellij.ui.SimpleListCellRenderer
import com.intellij.ui.components.JBLabel
import com.intellij.ui.components.JBScrollPane
import com.intellij.ui.content.ContentFactory
import com.intellij.ui.treeStructure.Tree
import com.intellij.util.ui.JBUI
import loremaster.engineprocess.EngineProcessStore
import loremaster.projectsettings.OutputSettings
import loremaster.projectsettings.readProjectSettings
import java.awt.BorderLayout
import java.awt.event.MouseAdapter
import java.awt.event.MouseEvent
import java.nio.file.Path
import javax.swing.JButton
import javax.swing.JComboBox
import javax.swing.JPanel
import javax.swing.SwingUtilities
import javax.swing.tree.DefaultMutableTreeNode
import javax.swing.tree.DefaultTreeModel

/** The LoreMaster tool window: the page tree of an output, with what each file says about its page. Double-click opens the file. */
class PagesToolWindowFactory : ToolWindowFactory {
    override fun createToolWindowContent(project: Project, toolWindow: ToolWindow) {
        val view = PagesView(project)
        toolWindow.contentManager.addContent(ContentFactory.getInstance().createContent(view.component, "Pages", false))
        view.reload()
    }
}

private class PagesView(private val project: Project) {
    private val outputs = JComboBox<IndexedValue<OutputSettings>>()
    private val status = JBLabel(" ")
    private val tree = Tree(DefaultTreeModel(DefaultMutableTreeNode("Loading...")))
    val component = JPanel(BorderLayout(0, JBUI.scale(4)))

    init {
        outputs.renderer = SimpleListCellRenderer.create("") { it.value.label() }
        val refresh = JButton("Refresh")
        refresh.addActionListener { reload() }
        outputs.addActionListener { if (outputs.isPopupVisible.not()) loadSelected() }
        val top = JPanel(BorderLayout(JBUI.scale(4), 0)).apply {
            add(outputs, BorderLayout.CENTER)
            add(refresh, BorderLayout.EAST)
        }
        tree.isRootVisible = false
        tree.addMouseListener(object : MouseAdapter() {
            override fun mouseClicked(event: MouseEvent) {
                if (event.clickCount == 2) openSelected()
            }
        })
        component.add(top, BorderLayout.NORTH)
        component.add(JBScrollPane(tree), BorderLayout.CENTER)
        component.add(status, BorderLayout.SOUTH)
    }

    fun reload() {
        val root = project.basePath ?: return
        background {
            val engine = ApplicationManager.getApplication().getService(EngineProcessStore::class.java).connection()
            val found = readProjectSettings(engine, root).settings.outputs.withIndex().toList()
            SwingUtilities.invokeLater {
                val before = outputs.selectedIndex
                outputs.removeAllItems()
                found.forEach(outputs::addItem)
                if (found.isEmpty()) show(emptyList(), "No output is configured. Use LoreMaster: Open Project Settings.")
                else {
                    outputs.selectedIndex = before.coerceIn(0, found.size - 1)
                    loadSelected()
                }
            }
        }
    }

    private fun loadSelected() {
        val root = project.basePath ?: return
        val chosen = outputs.selectedItem as? IndexedValue<*> ?: return
        background {
            val engine = ApplicationManager.getApplication().getService(EngineProcessStore::class.java).connection()
            val result = readPageTree(engine, root, chosen.index)
            val notes = (result.problems.orEmpty() + result.warnings.orEmpty()).firstOrNull()
            SwingUtilities.invokeLater { show(result.nodes, notes ?: "${result.nodes.size} page(s)") }
        }
    }

    private fun show(nodes: List<TreeNode>, message: String) {
        val root = DefaultMutableTreeNode("pages")
        fun add(parent: DefaultMutableTreeNode, branch: PageBranch) {
            val row = DefaultMutableTreeNode(branch.node)
            parent.add(row)
            branch.children.forEach { add(row, it) }
        }
        toForest(nodes).forEach { add(root, it) }
        tree.model = DefaultTreeModel(root)
        tree.cellRenderer = object : com.intellij.ui.ColoredTreeCellRenderer() {
            override fun customizeCellRenderer(tree: javax.swing.JTree, value: Any?, selected: Boolean, expanded: Boolean, leaf: Boolean, row: Int, hasFocus: Boolean) {
                val node = (value as? DefaultMutableTreeNode)?.userObject as? TreeNode ?: return
                append(labelOf(node), if (node.status == "new" || node.status == "local-changes") com.intellij.ui.SimpleTextAttributes.REGULAR_BOLD_ATTRIBUTES else com.intellij.ui.SimpleTextAttributes.REGULAR_ATTRIBUTES)
                append("  ${node.path}", com.intellij.ui.SimpleTextAttributes.GRAYED_ATTRIBUTES)
            }
        }
        for (row in 0 until tree.rowCount) tree.expandRow(row)
        status.text = message
    }

    private fun openSelected() {
        val node = (tree.lastSelectedPathComponent as? DefaultMutableTreeNode)?.userObject as? TreeNode ?: return
        val base = project.basePath ?: return
        val file = LocalFileSystem.getInstance().refreshAndFindFileByNioFile(Path.of(base, node.path)) ?: return
        OpenFileDescriptor(project, file).navigate(true)
    }

    private fun background(work: () -> Unit) = ApplicationManager.getApplication().executeOnPooledThread {
        runCatching(work).onFailure { error -> SwingUtilities.invokeLater { status.text = "The engine did not answer: ${error.message}" } }
    }
}

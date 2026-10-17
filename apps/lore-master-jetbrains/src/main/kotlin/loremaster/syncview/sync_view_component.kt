package loremaster.syncview

import com.intellij.icons.AllIcons
import com.intellij.openapi.Disposable
import com.intellij.openapi.actionSystem.ActionManager
import com.intellij.openapi.actionSystem.ActionPlaces
import com.intellij.openapi.actionSystem.ActionToolbar
import com.intellij.openapi.actionSystem.DefaultActionGroup
import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.fileEditor.OpenFileDescriptor
import com.intellij.openapi.project.Project
import com.intellij.openapi.vfs.LocalFileSystem
import com.intellij.openapi.vfs.VirtualFileManager
import com.intellij.openapi.vfs.newvfs.BulkFileListener
import com.intellij.openapi.vfs.newvfs.events.VFileEvent
import com.intellij.ui.ColoredTreeCellRenderer
import com.intellij.ui.PopupHandler
import com.intellij.ui.SimpleTextAttributes
import com.intellij.ui.components.JBLabel
import com.intellij.ui.components.JBScrollPane
import com.intellij.ui.treeStructure.Tree
import com.intellij.util.ui.JBUI
import loremaster.configdescription.isSite
import loremaster.engineprocess.EngineProcessStore
import java.awt.BorderLayout
import java.awt.event.MouseAdapter
import java.awt.event.MouseEvent
import java.nio.file.Path
import javax.swing.Icon
import javax.swing.JPanel
import javax.swing.JTree
import javax.swing.SwingUtilities
import javax.swing.tree.DefaultMutableTreeNode
import javax.swing.tree.DefaultTreeModel
import javax.swing.tree.TreePath

private const val PLACE = "LoreMasterSyncView"

/** How long to wait for a burst of file changes to settle before reading the pages again. */
private const val RELOAD_AFTER_MS = 400

/** Files whose change can change what the view shows: Markdown, the settings, the sync state and the .gitignore files. */
private fun mattersToView(path: String): Boolean =
    path.endsWith(".md") || path.endsWith(".markdown") || path.endsWith("/.vscode/settings.json") || path.endsWith("/lore-master.json") ||
        path.endsWith("/.gitignore")

/**
 * The Sync view: Local and each storage as rows, with their pages under them. The view's own toolbar holds what applies to every
 * storage; the toolbar and the right-click menu under it hold the actions of the selected row (a tree cannot show icons on a row
 * the way VS Code does). Names and where each storage goes come from the engine; each row remembers how it is shown.
 */
class SyncViewComponent(private val project: Project) : JPanel(BorderLayout(0, JBUI.scale(4))), Disposable {
    private val tree = Tree(DefaultMutableTreeNode("Loading..."))
    private val status = JBLabel(" ")
    private val selectedName = JBLabel(" ")
    private val results = mutableMapOf<Int, WorkspaceTreeResult>()
    private var locations: Locations? = null
    private val reloadSoon = javax.swing.Timer(RELOAD_AFTER_MS) { reload() }.apply { isRepeats = false }

    private val display get() = project.getService(LocationDisplayStore::class.java)

    /** The storages shown now. */
    val storages: List<StorageLocation> get() = locations?.storages.orEmpty()

    /** Whether the storage at [index] publishes as a whole (a site or wiki) instead of syncing page by page. */
    fun isSite(index: Int): Boolean = storages.firstOrNull { it.index == index }?.let { locations?.description?.isSite(it.output.platform) } ?: false

    /** The row of the tree that is selected, if it is a location. */
    val selectedLocation: Row.LocationRow? get() = selectedRow() as? Row.LocationRow

    init {
        tree.isRootVisible = false
        tree.showsRootHandles = true
        tree.cellRenderer = SyncViewRenderer()
        tree.addMouseListener(object : MouseAdapter() {
            override fun mouseClicked(event: MouseEvent) {
                if (event.clickCount == 2) openSelected()
            }
        })
        tree.addTreeSelectionListener { selectionChanged() }
        val actions = SyncViewActions(this, project)
        PopupHandler.installPopupMenu(tree, actions.rowGroup(), ActionPlaces.POPUP)

        val whole = toolbar(actions.wholeViewGroup())
        val row = toolbar(actions.rowGroup())
        val header = JPanel(BorderLayout()).apply {
            add(whole.component, BorderLayout.NORTH)
            add(JPanel(BorderLayout(JBUI.scale(6), 0)).apply {
                add(selectedName, BorderLayout.WEST)
                add(row.component, BorderLayout.CENTER)
            }, BorderLayout.SOUTH)
        }
        whole.targetComponent = tree
        row.targetComponent = tree
        add(header, BorderLayout.NORTH)
        add(JBScrollPane(tree), BorderLayout.CENTER)
        add(status, BorderLayout.SOUTH)
        project.messageBus.connect(this).subscribe(VirtualFileManager.VFS_CHANGES, object : BulkFileListener {
            override fun after(events: List<VFileEvent>) {
                if (events.any { mattersToView(it.path) }) SwingUtilities.invokeLater { reloadSoon.restart() }
            }
        })
        reload()
    }

    private fun toolbar(group: DefaultActionGroup): ActionToolbar = ActionManager.getInstance().createActionToolbar(PLACE, group, true)

    override fun dispose() {
        reloadSoon.stop()
    }

    /** Reads the storages and every location's pages again, and redraws. */
    fun reload() {
        val root = project.basePath ?: return
        background {
            val engine = ApplicationManager.getApplication().getService(EngineProcessStore::class.java).connection()
            val read = readLocations(engine, root)
            val fresh = linkedMapOf<Int, WorkspaceTreeResult>()
            if (read.storages.isNotEmpty()) {
                fresh[LOCAL_INDEX] = readLocalTree(engine, root)
                read.storages.forEach { fresh[it.index] = readPageTree(engine, root, it.index) }
            }
            SwingUtilities.invokeLater {
                locations = read
                results.clear()
                results.putAll(fresh)
                redraw()
            }
        }
    }

    /** Reads one location's pages again, and redraws. */
    fun reload(index: Int) {
        val root = project.basePath ?: return
        background {
            val engine = ApplicationManager.getApplication().getService(EngineProcessStore::class.java).connection()
            val result = if (index == LOCAL_INDEX) readLocalTree(engine, root) else readPageTree(engine, root, index)
            SwingUtilities.invokeLater {
                results[index] = result
                redraw()
            }
        }
    }

    /** Draws the rows again from what was read, keeping what is expanded and selected where it can. */
    fun redraw() {
        val before = expandedPaths()
        val selected = selectedLocation?.index
        val root = DefaultMutableTreeNode("locations")
        val read = locations
        if (read == null || read.storages.isEmpty()) {
            root.add(DefaultMutableTreeNode(Row.NoticeRow("No sync storage yet. Use Add Sync Storage to choose where your Markdown goes.")))
        } else {
            add(root, location(LOCAL_INDEX, "Local", "every Markdown file in the repository", "local"))
            read.storages.forEach { add(root, location(it.index, it.label, it.detail, it.icon)) }
        }
        tree.model = DefaultTreeModel(root)
        expand(before)
        selected?.let { selectLocation(it) }
        status.text = statusText()
        selectionChanged()
    }

    private fun location(index: Int, label: String, detail: String, icon: String): RowTree {
        val shown = display.displayOf(index)
        val result = results[index] ?: WorkspaceTreeResult()

        return RowTree(Row.LocationRow(index, label, detail, icon, shown), locationRows(index, result, shown))
    }

    private fun add(parent: DefaultMutableTreeNode, row: RowTree) {
        val node = DefaultMutableTreeNode(row.row)
        parent.add(node)
        row.children.forEach { add(node, it) }
    }

    private fun statusText(): String {
        val problems = results.values.flatMap { it.problems.orEmpty() + it.warnings.orEmpty() }
        return problems.firstOrNull() ?: "${storages.size} storage(s)"
    }

    private fun selectedRow(): Row? = (tree.lastSelectedPathComponent as? DefaultMutableTreeNode)?.userObject as? Row

    private fun selectionChanged() {
        selectedName.text = selectedLocation?.let { "${it.label}:" } ?: "Select a storage:"
    }

    private fun expandedPaths(): List<String> =
        (0 until tree.rowCount).mapNotNull { tree.getPathForRow(it) }.filter { tree.isExpanded(it) }.map { describe(it) }

    private fun describe(path: TreePath): String = path.path.drop(1).joinToString("/") { part ->
        when (val row = (part as? DefaultMutableTreeNode)?.userObject) {
            is Row.LocationRow -> "L${row.index}"
            is Row.PageRow -> "P${row.node.path}"
            is Row.LeftOutRow -> "O${row.index}"
            else -> ""
        }
    }

    private fun expand(paths: List<String>) {
        val wanted = paths.toSet()
        var row = 0
        while (row < tree.rowCount) {
            tree.getPathForRow(row)?.let { if (describe(it) in wanted) tree.expandRow(row) }
            row++
        }
    }

    /** Selects the row of a location (Local is -1), as clicking it would. */
    fun selectLocation(index: Int) {
        val root = tree.model.root as? DefaultMutableTreeNode ?: return
        root.children().toList().filterIsInstance<DefaultMutableTreeNode>()
            .firstOrNull { (it.userObject as? Row.LocationRow)?.index == index }
            ?.let { tree.selectionPath = TreePath(it.path) }
    }

    /** Opens the file of the selected page or left-out file. */
    private fun openSelected() {
        val base = project.basePath ?: return
        val path = when (val row = selectedRow()) {
            is Row.PageRow -> row.node.path
            is Row.LeftOutFileRow -> row.file.path
            else -> return
        }
        val file = LocalFileSystem.getInstance().refreshAndFindFileByNioFile(Path.of(base, path)) ?: return
        OpenFileDescriptor(project, file).navigate(true)
    }

    private fun background(work: () -> Unit) = ApplicationManager.getApplication().executeOnPooledThread {
        runCatching(work).onFailure { error -> SwingUtilities.invokeLater { status.text = "The engine did not answer: ${error.message}" } }
    }

    /** How a row is drawn. */
    private inner class SyncViewRenderer : ColoredTreeCellRenderer() {
        override fun customizeCellRenderer(tree: JTree, value: Any?, selected: Boolean, expanded: Boolean, leaf: Boolean, row: Int, hasFocus: Boolean) {
            when (val shown = (value as? DefaultMutableTreeNode)?.userObject as? Row ?: return) {
                is Row.LocationRow -> {
                    icon = iconFor(shown.icon)
                    append(shown.label, SimpleTextAttributes.REGULAR_BOLD_ATTRIBUTES)
                    // The detail is where it goes; a name that already says so (a made-up one) does not need it twice.
                    if (shown.detail.isNotBlank() && !shown.label.contains(shown.detail)) append("  ${shown.detail}", SimpleTextAttributes.GRAYED_ATTRIBUTES)
                }
                is Row.PageRow -> {
                    val loud = shown.node.status == "new" || shown.node.status == "local-changes"
                    append(shown.label, if (loud) SimpleTextAttributes.REGULAR_BOLD_ATTRIBUTES else SimpleTextAttributes.REGULAR_ATTRIBUTES)
                    append(statusMark(shown.node), SimpleTextAttributes.GRAYED_ATTRIBUTES)
                    append("  ${shown.other}", SimpleTextAttributes.GRAYED_ATTRIBUTES)
                }
                is Row.LeftOutRow -> append(leftOutLabel(shown), SimpleTextAttributes.GRAYED_ATTRIBUTES)
                is Row.LeftOutFileRow -> {
                    append(shown.file.path, SimpleTextAttributes.GRAYED_ATTRIBUTES)
                    append("  ${shown.file.rule}", SimpleTextAttributes.GRAYED_SMALL_ATTRIBUTES)
                }
                is Row.NoticeRow -> append(shown.message, SimpleTextAttributes.ERROR_ATTRIBUTES)
            }
        }

        private fun statusMark(node: TreeNode): String = when (node.status) {
            "new" -> "  (new)"
            "local-changes" -> "  (changed)"
            else -> ""
        }

        private fun iconFor(hint: String): Icon = when (hint) {
            "local" -> AllIcons.Nodes.Folder
            "site" -> AllIcons.General.Web
            "wiki" -> AllIcons.Toolwindows.Documentation
            else -> AllIcons.Nodes.Related
        }
    }
}

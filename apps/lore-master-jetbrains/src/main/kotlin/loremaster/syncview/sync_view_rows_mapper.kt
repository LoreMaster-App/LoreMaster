package loremaster.syncview

/** The index the Local row stands for: every Markdown file, whatever the storages leave out. */
const val LOCAL_INDEX = -1

/** One row of the Sync view's tree. */
sealed interface Row {
    /** Local, or a storage; [detail] is where it goes; [icon] is the platform's neutral hint. */
    data class LocationRow(val index: Int, val label: String, val detail: String, val icon: String, val display: LocationDisplay) : Row {
        val isLocal: Boolean get() = index == LOCAL_INDEX
    }

    /** A page, named by [label] with [other] (the file name or title) beside it. */
    data class PageRow(val index: Int, val node: TreeNode, val label: String, val other: String) : Row

    /** The files a storage leaves out, collapsed under its pages; [total] is more than [files] when the list was capped. */
    data class LeftOutRow(val index: Int, val files: List<LeftOutFile>, val total: Int) : Row

    data class LeftOutFileRow(val index: Int, val file: LeftOutFile) : Row

    data class NoticeRow(val message: String) : Row
}

/** A row and the rows under it. */
data class RowTree(val row: Row, val children: List<RowTree> = emptyList())

private fun fileNameOf(node: TreeNode): String = node.path.substringAfterLast('/')

private fun pageTitleOf(node: TreeNode): String = node.pageTitle.ifBlank { node.title }.ifBlank { node.path }

/** The label of a page and the other name beside it, by how the location names pages. */
fun pageLabels(node: TreeNode, display: LocationDisplay): Pair<String, String> =
    if (display.fileNames) fileNameOf(node) to pageTitleOf(node) else pageTitleOf(node) to fileNameOf(node)

private fun pageRowOf(index: Int, branch: PageBranch, display: LocationDisplay, nested: Boolean): RowTree {
    val (label, other) = pageLabels(branch.node, display)

    return RowTree(Row.PageRow(index, branch.node, label, other), if (nested) branch.children.map { pageRowOf(index, it, display, true) } else emptyList())
}

private fun allBranches(branches: List<PageBranch>): List<PageBranch> = branches.flatMap { listOf(it) + allBranches(it.children) }

/**
 * The rows under a location: the engine's problems as notices, then its pages as the tree the sync would make (or one flat list
 * by path), then, unless hidden or the location is Local, the files it leaves out, collapsed.
 */
fun locationRows(index: Int, result: WorkspaceTreeResult, display: LocationDisplay): List<RowTree> {
    val local = index == LOCAL_INDEX
    val rows = result.problems.orEmpty().map { RowTree(Row.NoticeRow(it)) }.toMutableList()
    val forest = toForest(result.nodes)
    if (display.flat) {
        allBranches(forest).sortedBy { it.node.path }.forEach { rows.add(pageRowOf(index, it, display, nested = false)) }
    } else {
        forest.forEach { rows.add(pageRowOf(index, it, display, nested = true)) }
    }
    val leftOut = result.leftOut.orEmpty()
    if (!local && !display.hideIgnored && leftOut.isNotEmpty()) {
        val files = leftOut.map { RowTree(Row.LeftOutFileRow(index, it)) }
        rows.add(RowTree(Row.LeftOutRow(index, leftOut, result.leftOutTotal ?: leftOut.size), files))
    }

    return rows
}

/** The text of a Left out row: how many, and how many more there are than are listed. */
fun leftOutLabel(row: Row.LeftOutRow): String =
    if (row.total > row.files.size) "Left out (${row.total}, ${row.files.size} listed)" else "Left out (${row.total})"

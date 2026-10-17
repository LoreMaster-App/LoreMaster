package loremaster.syncview

/** A page with the pages that nest under it. */
data class PageBranch(val node: TreeNode, val children: List<PageBranch>)

/**
 * Turns the engine's flat list (parents first, each node naming its parent file) into a forest. A node whose parent is not in
 * the list (or is itself) sits at the top rather than being lost.
 */
fun toForest(nodes: List<TreeNode>): List<PageBranch> {
    val known = nodes.map { it.path }.toSet()
    val childrenOf = nodes.groupBy { node -> node.parent?.takeIf { it.isNotEmpty() && it != node.path && it in known } }

    fun branch(node: TreeNode, seen: Set<String>): PageBranch =
        PageBranch(node, childrenOf[node.path].orEmpty().filter { it.path !in seen }.map { branch(it, seen + it.path) })

    return childrenOf[null].orEmpty().map { branch(it, setOf(it.path)) }
}

/** The label a row shows: the page title, with a mark for what the files say about it. */
fun labelOf(node: TreeNode): String {
    val mark = when (node.status) {
        "new" -> "  (new)"
        "local-changes" -> "  (changed)"
        "synced" -> ""
        else -> ""
    }

    return node.pageTitle.ifBlank { node.title }.ifBlank { node.path } + mark
}

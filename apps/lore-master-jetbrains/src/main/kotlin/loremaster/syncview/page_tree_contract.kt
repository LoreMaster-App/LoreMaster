package loremaster.syncview

/** The engine's `workspace/tree`: the page tree one output would sync, from the files alone. No connection, nothing written. */
const val METHOD_WORKSPACE_TREE = "workspace/tree"

data class WorkspaceTreeParams(val workspaceRoot: String, val output: Int, val scope: String? = null)

/** One page. [path] is workspace-relative with '/'; [parent] is the file it nests under, empty for the top. */
data class TreeNode(
    val path: String = "",
    val title: String = "",
    val pageTitle: String = "",
    val parent: String? = null,
    val rule: String = "",
    val depth: Int = 0,
    /** new, synced or local-changes; empty for an output that does not track pages in the files. */
    val status: String? = null,
    val pageId: String? = null,
    val warnings: List<String>? = null,
)

data class WorkspaceTreeResult(
    val nodes: List<TreeNode> = emptyList(),
    val warnings: List<String>? = null,
    val problems: List<String>? = null,
)

package loremaster.syncview

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertIs
import kotlin.test.assertTrue

class SyncViewRowsMapperTest {
    private fun node(path: String, parent: String? = null, status: String? = null, title: String = path.substringAfterLast('/').removeSuffix(".md")) =
        TreeNode(path = path, title = title, pageTitle = "ENG: $title", parent = parent, status = status)

    private val pages = listOf(
        node("README.md", title = "Home"),
        node("docs/guide.md", "README.md", title = "Guide"),
        node("docs/api.md", "README.md", title = "Api"),
        node("other.md", title = "Other"),
    )

    private fun labelsOf(rows: List<RowTree>): List<String> = rows.mapNotNull { (it.row as? Row.PageRow)?.label }

    @Test
    fun `pages are the tree the sync would make, named by title`() {
        val rows = locationRows(0, WorkspaceTreeResult(nodes = pages), LocationDisplay())

        assertEquals(listOf("ENG: Home", "ENG: Other"), labelsOf(rows))
        assertEquals(listOf("ENG: Guide", "ENG: Api"), labelsOf(rows[0].children))
        assertEquals("README.md", (rows[0].row as Row.PageRow).other)
    }

    @Test
    fun `a location can name its pages by file name, with the title beside`() {
        val rows = locationRows(0, WorkspaceTreeResult(nodes = pages), LocationDisplay(fileNames = true))

        assertEquals(listOf("README.md", "other.md"), labelsOf(rows))
        assertEquals("ENG: Home", (rows[0].row as Row.PageRow).other)
    }

    @Test
    fun `a flat view lists every page once by path, with no nesting`() {
        val rows = locationRows(0, WorkspaceTreeResult(nodes = pages), LocationDisplay(flat = true))

        assertEquals(listOf("README.md", "docs/api.md", "docs/guide.md", "other.md"), rows.map { (it.row as Row.PageRow).node.path })
        assertTrue(rows.all { it.children.isEmpty() })
    }

    private val leftOut = listOf(LeftOutFile("drafts/a.md", "excludes"), LeftOutFile("internal/b.md", "ignore"))

    @Test
    fun `the files a storage leaves out are listed under its pages, collapsed, with how many`() {
        val rows = locationRows(0, WorkspaceTreeResult(nodes = pages, leftOut = leftOut, leftOutTotal = 5), LocationDisplay())

        val group = rows.last()
        assertIs<Row.LeftOutRow>(group.row)
        assertEquals(listOf("drafts/a.md", "internal/b.md"), group.children.map { (it.row as Row.LeftOutFileRow).file.path })
        assertEquals("Left out (5, 2 listed)", leftOutLabel(group.row as Row.LeftOutRow))
    }

    @Test
    fun `hiding ignored files, and being Local, leave the left-out files out`() {
        val result = WorkspaceTreeResult(nodes = pages, leftOut = leftOut)

        assertFalse(locationRows(0, result, LocationDisplay(hideIgnored = true)).any { it.row is Row.LeftOutRow })
        assertFalse(locationRows(LOCAL_INDEX, result, LocationDisplay()).any { it.row is Row.LeftOutRow })
    }

    @Test
    fun `the engine's problems come first as notices`() {
        val rows = locationRows(0, WorkspaceTreeResult(nodes = pages, problems = listOf("outputs[0].space is empty")), LocationDisplay())

        assertEquals(Row.NoticeRow("outputs[0].space is empty"), rows.first().row)
    }

    @Test
    fun `leftOutLabel is just the count when all are listed`() {
        assertEquals("Left out (2)", leftOutLabel(Row.LeftOutRow(0, leftOut, 2)))
    }
}

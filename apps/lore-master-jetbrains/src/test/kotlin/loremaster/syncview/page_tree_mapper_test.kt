package loremaster.syncview

import kotlin.test.Test
import kotlin.test.assertEquals

class PageTreeMapperTest {
    private fun node(path: String, parent: String? = null, status: String? = null, title: String = path) =
        TreeNode(path = path, title = title, pageTitle = "ENG: $title", parent = parent, status = status)

    @Test
    fun `the flat list becomes a forest following the parent files`() {
        val forest = toForest(listOf(node("README.md"), node("docs/a.md", "README.md"), node("docs/b.md", "README.md"), node("docs/a1.md", "docs/a.md"), node("other.md")))

        assertEquals(listOf("README.md", "other.md"), forest.map { it.node.path })
        assertEquals(listOf("docs/a.md", "docs/b.md"), forest[0].children.map { it.node.path })
        assertEquals(listOf("docs/a1.md"), forest[0].children[0].children.map { it.node.path })
    }

    @Test
    fun `a node whose parent is missing, or itself, stays at the top instead of vanishing`() {
        val forest = toForest(listOf(node("a.md", "gone.md"), node("b.md", "b.md")))

        assertEquals(listOf("a.md", "b.md"), forest.map { it.node.path })
    }

    @Test
    fun `a parent cycle does not loop forever`() {
        val forest = toForest(listOf(node("a.md", "b.md"), node("b.md", "a.md")))

        assertEquals(0, forest.size)
    }

    @Test
    fun `the label shows what the files say about the page`() {
        assertEquals("ENG: Home  (new)", labelOf(node("a.md", status = "new", title = "Home")))
        assertEquals("ENG: Home  (changed)", labelOf(node("a.md", status = "local-changes", title = "Home")))
        assertEquals("ENG: Home", labelOf(node("a.md", status = "synced", title = "Home")))
        assertEquals("a.md", labelOf(TreeNode(path = "a.md")))
    }
}

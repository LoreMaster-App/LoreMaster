package loremaster.sync

import java.nio.file.Path
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

class SyncScopePolicyTest {
    private val root = Path.of("work", "project").toAbsolutePath()

    @Test
    fun `selected markdown files become sorted workspace-relative paths`() {
        val selected = listOf(root.resolve("docs/b.md"), root.resolve("a.markdown"), root.resolve("docs/b.md"), root.resolve("logo.png"))

        assertEquals(listOf("a.markdown", "docs/b.md"), syncScope(root, selected))
    }

    @Test
    fun `no markdown selected, or one outside the project, means everything`() {
        assertNull(syncScope(root, emptyList()))
        assertNull(syncScope(root, listOf(root.resolve("logo.png"))))
        assertNull(syncScope(root, listOf(root.resolve("a.md"), root.parent.resolve("other/b.md"))))
    }

    @Test
    fun `the plan is summarised in one line`() {
        assertEquals("2 create, 1 update, 5 unchanged", summarise(SyncPlanResult(counts = mapOf("unchanged" to 5, "create" to 2, "update" to 1))))
        assertEquals("Nothing to do.", summarise(SyncPlanResult()))
    }
}

package loremaster.exportmarkdown

import java.nio.file.Path
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

class ExportDestinationPolicyTest {
    private val root = Path.of("work", "project").toAbsolutePath()

    @Test
    fun `a blank setting leaves the engine to write next to the document`() {
        assertNull(exportDestination(root.resolve("docs/a.md"), root, "  "))
    }

    @Test
    fun `a relative folder is under the project and mirrors the document's place`() {
        assertEquals(root.resolve("out/docs/guide"), exportDestination(root.resolve("docs/guide/a.md"), root, "out"))
        assertEquals(root.resolve("out"), exportDestination(root.resolve("a.md"), root, "out"))
    }

    @Test
    fun `an absolute folder is kept, and a document outside the project lands in the folder itself`() {
        val absolute = Path.of("exports").toAbsolutePath()

        assertEquals(absolute.resolve("docs"), exportDestination(root.resolve("docs/a.md"), root, absolute.toString()))
        assertEquals(root.resolve("out"), exportDestination(root.parent.resolve("elsewhere/a.md"), root, "out"))
    }
}

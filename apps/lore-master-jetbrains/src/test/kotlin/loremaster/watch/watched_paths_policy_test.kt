package loremaster.watch

import java.nio.file.Path
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertNull
import kotlin.test.assertTrue

class WatchedPathsPolicyTest {
    @Test
    fun `markdown, settings and what generators read matter`() {
        listOf("README.md", "docs/a.md", ".vscode/settings.json", "report/junit.xml", "src/main.go", "api/openapi.yaml")
            .forEach { assertTrue(isWatchedPath(it), it) }
    }

    @Test
    fun `dependencies, build output, hidden folders and other files do not`() {
        listOf("node_modules/x/readme.md", "dist/index.js", "build/out.json", ".git/config", ".idea/workspace.xml", "docs/.hidden.md", "logo.png", "Makefile")
            .forEach { assertFalse(isWatchedPath(it), it) }
    }

    @Test
    fun `a file is named relative to the folder with slashes, or not at all outside it`() {
        val folder = Path.of("work", "project").toAbsolutePath()

        assertEquals("docs/a.md", workspacePath(folder, folder.resolve("docs").resolve("a.md")))
        assertNull(workspacePath(folder, folder))
        assertNull(workspacePath(folder, folder.parent.resolve("other").resolve("a.md")))
    }
}

package loremaster.syncview

import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.ToggleAction
import com.intellij.testFramework.PlatformTestUtil
import com.intellij.testFramework.TestActionEvent
import com.intellij.testFramework.fixtures.BasePlatformTestCase
import loremaster.engineprocess.EngineRpcClient
import java.nio.file.Files
import java.nio.file.Path

/** The Sync view built in a headless IDE over the real engine, in a project with a GitHub Pages storage. */
class SyncViewComponentTest : BasePlatformTestCase() {
    private fun engineConfigured(): Boolean = System.getenv("LORE_MASTER_ENGINE")?.takeIf { it.isNotBlank() }?.let { Files.exists(Path.of(it)) } ?: false

    private fun writeProject(settings: String) {
        val root = Path.of(project.basePath!!)
        Files.createDirectories(root.resolve(".vscode"))
        Files.writeString(root.resolve(".vscode/settings.json"), settings)
        Files.writeString(root.resolve("README.md"), "# Home\n\nHello.\n")
    }

    private fun waitFor(condition: () -> Boolean) {
        repeat(100) {
            PlatformTestUtil.dispatchAllEventsInIdeEventQueue()
            if (!condition()) Thread.sleep(100)
        }
    }

    private fun visible(actions: List<AnAction>): List<String> = actions.filter {
        val event = TestActionEvent.createTestEvent(it)
        it.update(event)
        event.presentation.isEnabledAndVisible
    }.map { it.templateText.orEmpty() }

    fun `test the tool window is the Sync view`() {
        val factory = com.intellij.openapi.wm.ToolWindowEP.EP_NAME.extensionList.first { it.id == "LoreMaster" }

        assertEquals("loremaster.syncview.SyncToolWindowFactory", factory.factoryClass)
    }

    fun `test Local and the storage are rows, and each shows only the actions that are about it`() {
        if (!engineConfigured()) return
        writeProject("""{ "loreMaster.outputs": [ { "platform": "github-pages", "repo": "acme/docs" } ] }""")
        val view = SyncViewComponent(project)
        try {
            waitFor { view.storages.isNotEmpty() }

            assertEquals(listOf("GitHub Pages · acme/docs"), view.storages.map { it.label })
            assertEquals("acme/docs", view.storages.single().detail)
            val row = SyncViewActions(view, project).rowGroup().getChildren(null).toList()

            view.selectLocation(LOCAL_INDEX)
            assertEquals(listOf("Show File Names", "Flat View", "Refresh Storage"), visible(row))

            view.selectLocation(0)
            assertEquals(
                listOf(
                    "Check Remote Status", "Show File Names", "Flat View", "Hide Ignored", "Refresh Storage", "Sync Storage",
                    "Generate and Sync", "Edit Storage", "Remove Storage",
                ),
                visible(row),
            )
        } finally {
            view.dispose()
        }
    }

    fun `test a display choice is made on one row and remembered for it alone`() {
        if (!engineConfigured()) return
        writeProject("""{ "loreMaster.outputs": [ { "platform": "github-pages", "repo": "acme/docs" } ] }""")
        val view = SyncViewComponent(project)
        try {
            waitFor { view.storages.isNotEmpty() }
            val flat = SyncViewActions(view, project).rowGroup().getChildren(null).first { it.templateText == "Flat View" } as ToggleAction

            view.selectLocation(0)
            val event = TestActionEvent.createTestEvent(flat)
            assertFalse(flat.isSelected(event))
            flat.setSelected(event, true)

            val store = project.getService(LocationDisplayStore::class.java)
            assertTrue(store.displayOf(0).flat)
            assertFalse(store.displayOf(LOCAL_INDEX).flat)
            assertTrue(flat.isSelected(event))
        } finally {
            view.dispose()
        }
    }

    fun `test a project with no storage says so instead of listing Local alone`() {
        if (!engineConfigured()) return
        writeProject("""{ "loreMaster.outputs": [] }""")
        val view = SyncViewComponent(project)
        try {
            waitFor { view.storages.isEmpty() && view.selectedLocation == null }
            PlatformTestUtil.dispatchAllEventsInIdeEventQueue()

            assertTrue(view.storages.isEmpty())
        } finally {
            view.dispose()
        }
    }

    fun `test the real engine names and summarises the storages the view reads`() {
        val binary = System.getenv("LORE_MASTER_ENGINE")?.takeIf { it.isNotBlank() }?.let(Path::of)
        if (binary == null || !Files.exists(binary)) return
        writeProject(
            """{ "loreMaster.outputs": [
                 { "platform": "confluence", "baseUrl": "", "space": "", "parentPageId": "", "titlePrefix": "" },
                 { "platform": "github-wiki", "repo": "acme/docs" } ] }""",
        )
        val process = ProcessBuilder(binary.toString()).redirectError(ProcessBuilder.Redirect.DISCARD).start()
        try {
            EngineRpcClient(process.inputStream, process.outputStream).use { engine ->
                val locations = readLocations(engine, project.basePath!!)

                assertEquals(listOf(1), locations.storages.map { it.index })
                assertEquals("GitHub Wiki · acme/docs", locations.storages.single().label)
                assertEquals("wiki", locations.storages.single().icon)
            }
        } finally {
            process.destroy()
        }
    }
}

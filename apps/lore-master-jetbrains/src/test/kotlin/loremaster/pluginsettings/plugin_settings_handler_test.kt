package loremaster.pluginsettings

import com.intellij.openapi.actionSystem.ActionManager
import com.intellij.testFramework.fixtures.BasePlatformTestCase
import loremaster.configdescription.ChoiceDescriptor
import loremaster.configdescription.ConfigDescription
import loremaster.configdescription.FIELD_BOOL
import loremaster.configdescription.FIELD_CHOICE
import loremaster.configdescription.FIELD_NUMBER
import loremaster.configdescription.FIELD_PATH
import loremaster.configdescription.FieldDescriptor
import loremaster.configdescription.SectionDescriptor
import loremaster.configdescription.describeConfig
import loremaster.engineprocess.EngineRpcClient
import loremaster.projectsettings.OpenProjectSettingsHandler
import java.nio.file.Files
import java.nio.file.Path

/** The settings page and the project settings action, loaded in a headless IDE. */
class PluginSettingsHandlerTest : BasePlatformTestCase() {
    private val described = ConfigDescription(
        sections = listOf(
            SectionDescriptor(
                id = "export", label = "Export", scope = "user",
                settings = listOf(
                    FieldDescriptor(
                        key = "export.pageSize", label = "PDF page size", type = FIELD_CHOICE, default = "A4",
                        choices = listOf(ChoiceDescriptor("A4", "A4"), ChoiceDescriptor("Letter", "Letter")),
                    ),
                    FieldDescriptor(key = "export.pngWidth", label = "PNG width", type = FIELD_NUMBER, default = "1000", min = 320, max = 4000),
                    FieldDescriptor(key = "export.folder", label = "Export folder", type = FIELD_PATH, pathKind = "folder"),
                    FieldDescriptor(key = "generateBeforeSync", label = "Generate before every sync", type = FIELD_BOOL, shells = listOf("vscode")),
                ),
            ),
            SectionDescriptor(id = "project", label = "Project", scope = "project", settings = listOf(FieldDescriptor(key = "ignore", label = "Ignore"))),
        ),
    )

    fun `test the page is drawn from the engine's description and keeps what the user set`() {
        val settings = PluginSettingsStore.current()
        settings.set("export.pageSize", "Letter")
        settings.set("export.pngWidth", "1400")
        val page = PluginSettingsHandler { described }
        try {
            assertNotNull(page.createComponent())
            page.reset()
            assertFalse(page.isModified)
        } finally {
            page.disposeUIResources()
        }
        assertEquals("Letter", PluginSettingsStore.current().value("export.pageSize"))
        assertEquals("1400", PluginSettingsStore.current().value("export.pngWidth"))
    }

    fun `test the page still opens when the engine cannot be reached`() {
        val page = PluginSettingsHandler { null }
        try {
            assertNotNull(page.createComponent())
        } finally {
            page.disposeUIResources()
        }
    }

    fun `test only the user's settings this editor honours are shown`() {
        val sections = userSections(described)

        assertEquals(listOf("Export"), sections.map { it.label })
        assertEquals(listOf("export.pageSize", "export.pngWidth", "export.folder"), sections.single().settings.map { it.key })
    }

    fun `test a setting left on its default is not kept, and one the user changed is`() {
        val settings = PluginSettings()
        settings.set("export.pageSize", "Letter")
        settings.set("export.pngWidth", "")

        assertEquals("Letter", settings.value("export.pageSize"))
        assertEquals("", settings.value("export.pngWidth"))
        assertEquals("Letter", effectiveValue(described, settings, "export.pageSize"))
        assertEquals("1000", effectiveValue(described, settings, "export.pngWidth"))
        assertEquals(1000, effectiveNumber(described, settings, "export.pngWidth"))
        settings.set("export.pngWidth", "99999")
        assertEquals(4000, effectiveNumber(described, settings, "export.pngWidth"))
    }

    fun `test settings kept by an earlier version move into the described settings`() {
        val old = PluginSettings(pageSize = "Letter", pngWidth = 1400, exportFolder = "dist", watchDebounceSeconds = 2, browserPath = "")

        val moved = old.migrated()

        assertEquals("Letter", moved.value("export.pageSize"))
        assertEquals("1400", moved.value("export.pngWidth"))
        assertEquals("dist", moved.value("export.folder"))
        assertEquals("", moved.value("watchDebounceSeconds"))
        assertEquals("", moved.pageSize)
        assertEquals(0, moved.pngWidth)
    }

    fun `test the real engine's settings page has the export and watch settings`() {
        val binary = System.getenv("LORE_MASTER_ENGINE")?.takeIf { it.isNotBlank() }?.let(Path::of)
        if (binary == null || !Files.exists(binary)) return
        val process = ProcessBuilder(binary.toString()).redirectError(ProcessBuilder.Redirect.DISCARD).start()
        try {
            EngineRpcClient(process.inputStream, process.outputStream).use { engine ->
                val keys = userSections(describeConfig(engine)).flatMap { section -> section.settings.map { it.key } }

                assertTrue(keys.containsAll(listOf("export.pageSize", "export.pngWidth", "export.browserPath", "watchDebounceSeconds")))
                assertFalse("generateBeforeSync" in keys)
            }
        } finally {
            process.destroy()
        }
    }

    fun `test the project settings action creates the file the engine reads`() {
        val action = ActionManager.getInstance().getAction("loremaster.openProjectSettings")

        assertTrue(action is OpenProjectSettingsHandler)
        val file = Path.of(project.basePath!!, ".vscode", "settings.json")
        Files.deleteIfExists(file)
        val event = com.intellij.testFramework.TestActionEvent.createTestEvent(action)
        (action as OpenProjectSettingsHandler).actionPerformed(event)

        assertTrue(Files.exists(file))
    }
}

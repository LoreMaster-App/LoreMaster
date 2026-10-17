package loremaster.pluginsettings

import com.intellij.openapi.actionSystem.ActionManager
import com.intellij.testFramework.fixtures.BasePlatformTestCase
import loremaster.projectsettings.OpenProjectSettingsHandler
import java.nio.file.Files
import java.nio.file.Path

/** The settings page and the project settings action, loaded in a headless IDE. */
class PluginSettingsHandlerTest : BasePlatformTestCase() {
    fun `test the settings page builds and keeps what the user typed`() {
        val settings = PluginSettingsStore.current()
        settings.pageSize = "Letter"
        settings.pngWidth = 1400
        val page = PluginSettingsHandler()
        try {
            assertNotNull(page.createComponent())
            page.reset()
            assertFalse(page.isModified)
        } finally {
            page.disposeUIResources()
        }
        assertEquals("Letter", PluginSettingsStore.current().pageSize)
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

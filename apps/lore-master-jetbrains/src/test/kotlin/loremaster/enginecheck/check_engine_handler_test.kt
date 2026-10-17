package loremaster.enginecheck

import com.intellij.openapi.actionSystem.ActionManager
import com.intellij.openapi.application.ApplicationManager
import com.intellij.testFramework.fixtures.BasePlatformTestCase
import loremaster.engineprocess.EngineProcessStore
import loremaster.engineprocess.METHOD_PING
import loremaster.engineprocess.PingResult
import java.nio.file.Files
import java.nio.file.Path
import java.util.concurrent.TimeUnit

/** The plugin loaded in a real (headless) IDE: the action is registered and the store reaches the engine. */
class CheckEngineHandlerTest : BasePlatformTestCase() {
    fun `test the action is registered under Tools`() {
        val action = ActionManager.getInstance().getAction("loremaster.checkEngine")

        assertTrue(action is CheckEngineHandler)
        assertEquals("LoreMaster: Check Engine", action.templatePresentation.text)
    }

    fun `test the export action is on the editor, project and tools menus`() {
        val action = ActionManager.getInstance().getAction("loremaster.exportMarkdown")

        assertTrue(action is loremaster.exportmarkdown.ExportMarkdownHandler)
        for (group in listOf("ToolsMenu", "EditorPopupMenu", "ProjectViewPopupMenu")) {
            val children = (ActionManager.getInstance().getAction(group) as com.intellij.openapi.actionSystem.ActionGroup)
                .getChildren(null)
            assertTrue("not in $group", children.contains(action))
        }
    }

    fun `test the store starts the real engine and it answers ping`() {
        val binary = System.getenv("LORE_MASTER_ENGINE")?.takeIf { it.isNotBlank() }?.let(Path::of)
        if (binary == null || !Files.exists(binary)) {
            return
        }

        val store = ApplicationManager.getApplication().getService(EngineProcessStore::class.java)
        val answer = store.connection().request(METHOD_PING, null, PingResult::class.java).get(15, TimeUnit.SECONDS)

        assertEquals("pong", answer.pong)
    }
}

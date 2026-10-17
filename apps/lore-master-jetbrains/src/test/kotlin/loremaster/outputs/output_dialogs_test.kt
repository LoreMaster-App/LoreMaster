package loremaster.outputs

import com.intellij.openapi.actionSystem.ActionManager
import com.intellij.testFramework.PlatformTestUtil
import com.intellij.testFramework.fixtures.BasePlatformTestCase
import loremaster.configdescription.ConfigDescription
import loremaster.configdescription.PlatformDescriptor
import loremaster.connection.Credential
import loremaster.connection.CredentialKind
import loremaster.connection.openSession
import loremaster.engineprocess.EngineRpcClient
import loremaster.testing.FakeConfluenceFixture
import java.nio.file.Files
import java.nio.file.Path

/** The picker dialogs built in a headless IDE; the parent browser against the real engine and a fake Confluence. */
class OutputDialogsTest : BasePlatformTestCase() {
    fun `test the add output action is registered and its dialog builds`() {
        assertTrue(ActionManager.getInstance().getAction("loremaster.addOutput") is AddOutputHandler)

        val engine = EngineRpcClient(java.io.ByteArrayInputStream(ByteArray(0)), java.io.ByteArrayOutputStream())
        val description = ConfigDescription(platforms = listOf(PlatformDescriptor(id = "site", label = "Site")))
        val dialog = AddOutputDialogComponent(project, engine, description)
        try {
            assertFalse(dialog.isDisposed)
            assertEquals("site", dialog.platform.id)
        } finally {
            dialog.close(0)
            engine.close()
        }
    }

    fun `test the parent browser starts at the home page's children and defaults to the home page`() {
        val binary = System.getenv("LORE_MASTER_ENGINE")?.takeIf { it.isNotBlank() }?.let(Path::of)
        if (binary == null || !Files.exists(binary)) return
        FakeConfluenceFixture().use { site ->
            site.addPage("ENG", site.root.id, "Guides")
            site.addPage("ENG", site.root.id, "Reference")
            val process = ProcessBuilder(binary.toString()).redirectError(ProcessBuilder.Redirect.DISCARD).start()
            try {
                EngineRpcClient(process.inputStream, process.outputStream).use { engine ->
                    val session = openSession(engine, site.address, Credential(CredentialKind.PAT, token = "t"))
                    val space = listSpaces(engine, session.sessionId).single()
                    val home = RemotePage(space.homepageId!!, "Engineering home")
                    val dialog = ParentPageDialogComponent(project, engine, session.sessionId, space, home)
                    try {
                        repeat(60) {
                            PlatformTestUtil.dispatchAllEventsInIdeEventQueue()
                            if (dialog.shownPages.isEmpty()) Thread.sleep(100)
                        }

                        assertEquals(listOf("Guides", "Reference"), dialog.shownPages.map { it.title }.sorted())
                        assertEquals(home, dialog.chosen)
                    } finally {
                        dialog.close(0)
                    }
                }
            } finally {
                process.destroy()
            }
        }
    }

    fun `test the dialog draws the fields of whichever platform the engine describes`() {
        val binary = System.getenv("LORE_MASTER_ENGINE")?.takeIf { it.isNotBlank() }?.let(Path::of)
        if (binary == null || !Files.exists(binary)) return
        val process = ProcessBuilder(binary.toString()).redirectError(ProcessBuilder.Redirect.DISCARD).start()
        try {
            EngineRpcClient(process.inputStream, process.outputStream).use { engine ->
                val dialog = AddOutputDialogComponent(project, engine, loremaster.configdescription.describeConfig(engine))
                try {
                    dialog.choosePlatform("confluence")
                    assertTrue(dialog.fieldKeys.containsAll(listOf("name", "baseUrl", "space", "parentPageId", "titlePrefix", "direction", "roots")))
                    assertFalse("repo" in dialog.fieldKeys)

                    dialog.choosePlatform("github-pages")
                    assertTrue(dialog.fieldKeys.containsAll(listOf("name", "repo", "branch", "path", "template")))
                    assertFalse("baseUrl" in dialog.fieldKeys)

                    dialog.setValue("repo", "acme/docs")
                    dialog.setValue("name", "Docs site")
                    val output = dialog.output()
                    assertEquals("github-pages", output.get("platform").asString)
                    assertEquals("acme/docs", output.get("repo").asString)
                    assertEquals("Docs site", output.get("name").asString)
                    assertEquals("", output.get("baseUrl").asString)
                } finally {
                    dialog.close(0)
                }
            }
        } finally {
            process.destroy()
        }
    }
}

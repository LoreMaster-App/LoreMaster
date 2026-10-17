package loremaster.outputs

import loremaster.connection.Credential
import loremaster.connection.CredentialKind
import loremaster.connection.closeSession
import loremaster.connection.openSession
import loremaster.engineprocess.EngineRpcClient
import loremaster.projectsettings.readProjectSettings
import loremaster.sync.RunSync
import loremaster.sync.SyncChoice
import loremaster.sync.SyncOutcome
import loremaster.testing.FakeConfluenceFixture
import java.nio.file.Files
import java.nio.file.Path
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs
import kotlin.test.assertTrue

/** Finding a space and a parent page, then saving the output, through the real engine against a fake Confluence. */
class OutputsUseCaseTest {
    private fun withWorld(body: (EngineRpcClient, Path, FakeConfluenceFixture) -> Unit) {
        val binary = System.getenv("LORE_MASTER_ENGINE")?.takeIf { it.isNotBlank() }?.let(Path::of)
        if (binary == null || !Files.exists(binary)) return
        FakeConfluenceFixture().use { site ->
            val root = Files.createTempDirectory("lore-master-outputs")
            Files.writeString(root.resolve("README.md"), "# Home\n")
            val process = ProcessBuilder(binary.toString()).redirectError(ProcessBuilder.Redirect.DISCARD).start()
            try {
                EngineRpcClient(process.inputStream, process.outputStream).use { body(it, root, site) }
            } finally {
                process.destroy()
                root.toFile().deleteRecursively()
            }
        }
    }

    @Test
    fun `spaces, a page search and a page's children are listed through a session`() = withWorld { engine, _, site ->
        site.addPage("ENG", site.root.id, "Guides")
        site.addPage("ENG", site.root.id, "Reference")
        val session = openSession(engine, site.address, Credential(CredentialKind.PAT, token = "t"))
        try {
            val space = listSpaces(engine, session.sessionId).single()
            assertEquals("ENG", space.key)
            assertEquals(site.root.id, space.homepageId)

            assertEquals(listOf("Guides", "Reference"), pageChildren(engine, session.sessionId, site.root.id).map { it.title }.sorted())
            assertEquals(listOf("Guides"), searchPages(engine, session.sessionId, "ENG", "guid").map { it.title })
        } finally {
            closeSession(engine, session.sessionId)
        }
    }

    @Test
    fun `a saved output is read back and a sync can use it`() = withWorld { engine, root, site ->
        val index = addConfluenceOutput(engine, root.toString(), ConfluenceOutputChoice(site.address, "ENG", site.root.id, "ENG"))

        assertEquals(0, index)
        val output = readProjectSettings(engine, root.toString()).settings.outputs.single()
        assertEquals("confluence", output.platform)
        assertEquals(site.root.id, output.parentPageId)
        assertTrue(Files.readString(root.resolve(".vscode/settings.json")).contains("loreMaster.outputs"))

        val outcome = RunSync(engine, { Credential(CredentialKind.PAT, token = "t") }, { SyncChoice() }).run(root.toString(), output, index)
        assertIs<SyncOutcome.Done>(outcome)
        assertEquals(2, site.pages.size)
    }

    @Test
    fun `a second output is added to the first, not instead of it`() = withWorld { engine, root, site ->
        addConfluenceOutput(engine, root.toString(), ConfluenceOutputChoice(site.address, "ENG", site.root.id, "ENG"))

        val guides = site.addPage("ENG", site.root.id, "Guides")
        val index = addConfluenceOutput(engine, root.toString(), ConfluenceOutputChoice(site.address, "ENG", guides.id, "OPS"))

        assertEquals(1, index)
        assertEquals(listOf("ENG", "OPS"), readProjectSettings(engine, root.toString()).settings.outputs.map { it.titlePrefix })
    }
}

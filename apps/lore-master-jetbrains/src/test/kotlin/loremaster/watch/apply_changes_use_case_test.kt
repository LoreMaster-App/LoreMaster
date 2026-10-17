package loremaster.watch

import loremaster.connection.Credential
import loremaster.connection.CredentialKind
import loremaster.engineprocess.EngineRpcClient
import loremaster.projectsettings.readProjectSettings
import loremaster.sync.RunSync
import loremaster.sync.SyncChoice
import loremaster.testing.FakeConfluenceFixture
import java.nio.file.Files
import java.nio.file.Path
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

/** A batch of changes through the real engine against a fake Confluence: only what changed is synced. */
class ApplyChangesUseCaseTest {
    private fun withWorld(body: (EngineRpcClient, Path, FakeConfluenceFixture, (List<String>?) -> Unit) -> Unit) {
        val binary = System.getenv("LORE_MASTER_ENGINE")?.takeIf { it.isNotBlank() }?.let(Path::of)
        if (binary == null || !Files.exists(binary)) return
        FakeConfluenceFixture().use { site ->
            val root = Files.createTempDirectory("lore-master-watch")
            Files.writeString(root.resolve("README.md"), "# Home\n")
            Files.writeString(root.resolve("setup.md"), "# Setup\n")
            Files.createDirectories(root.resolve(".vscode"))
            Files.writeString(
                root.resolve(".vscode/settings.json"),
                """{ "loreMaster.outputs": [ { "platform": "confluence", "baseUrl": "${site.address}", "space": "ENG",
                      "parentPageId": "${site.root.id}", "titlePrefix": "ENG" } ] }""",
            )
            val process = ProcessBuilder(binary.toString()).redirectError(ProcessBuilder.Redirect.DISCARD).start()
            try {
                EngineRpcClient(process.inputStream, process.outputStream).use { engine ->
                    val output = readProjectSettings(engine, root.toString()).settings.outputs.single()
                    val flow = RunSync(engine, { Credential(CredentialKind.PAT, token = "t") }, { SyncChoice() })
                    body(engine, root, site) { scope -> flow.run(root.toString(), output, 0, scope) }
                }
            } finally {
                process.destroy()
                root.toFile().deleteRecursively()
            }
        }
    }

    @Test
    fun `a changed page is synced alone and a file nothing reads is left`() = withWorld { engine, root, site, sync ->
        sync(null)
        val versionsBefore = site.pages.associate { it.title to it.version }

        Files.writeString(root.resolve("setup.md"), "# Setup\n\nMore.\n")
        val seen = mutableListOf<List<String>?>()
        val applied = applyChanges(engine, root.toString(), listOf("setup.md", "notes.png")) { scope -> seen.add(scope); sync(scope) }

        assertEquals(listOf<List<String>?>(listOf("setup.md")), seen)
        assertEquals(listOf("setup.md"), applied.touched)
        assertEquals(versionsBefore.getValue("ENG: Home"), site.pages.first { it.title == "ENG: Home" }.version)
        assertTrue(site.pages.first { it.title == "ENG: Setup" }.version > versionsBefore.getValue("ENG: Setup"))
    }

    @Test
    fun `a change to nothing the sync reads does nothing`() = withWorld { engine, root, _, sync ->
        sync(null)
        var called = false

        val applied = applyChanges(engine, root.toString(), listOf("src/App.java")) { called = true }

        assertEquals(false, called)
        assertTrue(applied.log.single().startsWith("nothing to sync"))
    }

    @Test
    fun `a changed settings file syncs everything`() = withWorld { engine, root, _, _ ->
        val seen = mutableListOf<List<String>?>()

        applyChanges(engine, root.toString(), listOf(".vscode/settings.json")) { seen.add(it) }

        assertEquals(listOf<List<String>?>(null), seen)
    }
}

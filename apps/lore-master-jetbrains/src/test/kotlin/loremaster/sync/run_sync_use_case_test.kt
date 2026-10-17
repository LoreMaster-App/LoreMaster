package loremaster.sync

import loremaster.connection.Credential
import loremaster.connection.CredentialKind
import loremaster.engineprocess.EngineRpcClient
import loremaster.projectsettings.readProjectSettings
import loremaster.testing.FakeConfluenceFixture
import java.nio.file.Files
import java.nio.file.Path
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs
import kotlin.test.assertTrue

/** A whole sync through the real engine against a fake Confluence, driven the way the action drives it. */
class RunSyncUseCaseTest {
    private fun withEngine(body: (EngineRpcClient) -> Unit) {
        val binary = System.getenv("LORE_MASTER_ENGINE")?.takeIf { it.isNotBlank() }?.let(Path::of)
        if (binary == null || !Files.exists(binary)) {
            return
        }
        val process = ProcessBuilder(binary.toString()).redirectError(ProcessBuilder.Redirect.DISCARD).start()
        try {
            EngineRpcClient(process.inputStream, process.outputStream).use(body)
        } finally {
            process.destroy()
        }
    }

    private fun workspace(site: FakeConfluenceFixture): Path {
        val root = Files.createTempDirectory("lore-master-sync")
        Files.writeString(root.resolve("README.md"), "# Home\n\nSee [setup](setup.md).\n")
        Files.writeString(root.resolve("setup.md"), "# Setup\n")
        Files.createDirectories(root.resolve(".vscode"))
        Files.writeString(
            root.resolve(".vscode/settings.json"),
            """{ "loreMaster.outputs": [ { "platform": "confluence", "baseUrl": "${site.address}", "space": "ENG",
                  "parentPageId": "${site.root.id}", "titlePrefix": "ENG" } ] }""",
        )

        return root
    }

    @Test
    fun `the first sync creates the pages and the second finds nothing to do`() = withEngine { engine ->
        FakeConfluenceFixture().use { site ->
            val root = workspace(site)
            try {
                val output = readProjectSettings(engine, root.toString()).settings.outputs.single()
                assertEquals("confluence", output.platform)
                assertEquals(site.address, output.baseUrl)

                val seen = mutableListOf<SyncPlanResult>()
                val flow = RunSync(engine, { Credential(CredentialKind.PAT, token = "a-token") }, { seen.add(it); SyncChoice() })

                val first = flow.run(root.toString(), output, 0)
                assertIs<SyncOutcome.Done>(first)
                assertEquals(2, seen.single().counts["create"])
                assertEquals(listOf("ENG: Home", "ENG: Setup"), first.report.pages.filter { it.outcome == "written" }.map { it.title }.sorted())
                assertEquals(3, site.pages.size)

                seen.clear()
                val second = flow.run(root.toString(), output, 0)
                assertIs<SyncOutcome.Done>(second)
                assertEquals(2, seen.single().counts["unchanged"])
                assertEquals(3, site.pages.size)
            } finally {
                root.toFile().deleteRecursively()
            }
        }
    }

    @Test
    fun `declining to sign in or to go ahead changes nothing`() = withEngine { engine ->
        FakeConfluenceFixture().use { site ->
            val root = workspace(site)
            try {
                val output = readProjectSettings(engine, root.toString()).settings.outputs.single()

                assertEquals(SyncOutcome.Cancelled, RunSync(engine, { null }, { SyncChoice() }).run(root.toString(), output, 0))
                assertEquals(SyncOutcome.Cancelled, RunSync(engine, { Credential(CredentialKind.PAT, token = "t") }, { null }).run(root.toString(), output, 0))
                assertEquals(1, site.pages.size)
            } finally {
                root.toFile().deleteRecursively()
            }
        }
    }

    @Test
    fun `a scope limits the plan to the selected files`() = withEngine { engine ->
        FakeConfluenceFixture().use { site ->
            val root = workspace(site)
            try {
                val output = readProjectSettings(engine, root.toString()).settings.outputs.single()
                val seen = mutableListOf<SyncPlanResult>()
                val flow = RunSync(engine, { Credential(CredentialKind.PAT, token = "t") }, { seen.add(it); null })

                flow.run(root.toString(), output, 0, scope = listOf("setup.md"))

                assertTrue(seen.single().actions.any { it.path == "setup.md" })
            } finally {
                root.toFile().deleteRecursively()
            }
        }
    }
}

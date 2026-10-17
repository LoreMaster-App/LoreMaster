package loremaster.rollback

import loremaster.connection.Credential
import loremaster.connection.CredentialKind
import loremaster.connection.closeSession
import loremaster.connection.openSession
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

/** Undoing a sync through the real engine against a fake Confluence. */
class RollbackUseCaseTest {
    @Test
    fun `a sync can be previewed and then rolled back, and leaves the list`() {
        val binary = System.getenv("LORE_MASTER_ENGINE")?.takeIf { it.isNotBlank() }?.let(Path::of)
        if (binary == null || !Files.exists(binary)) return
        FakeConfluenceFixture().use { site ->
            val root = Files.createTempDirectory("lore-master-rollback")
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
                    RunSync(engine, { Credential(CredentialKind.PAT, token = "t") }, { SyncChoice() }).run(root.toString(), output, 0)
                    assertEquals(3, site.pages.size)

                    val journal = listJournals(engine, root.toString()).single()
                    assertEquals(2, journal.created)
                    assertTrue(journalLabel(journal).contains("2 created"))

                    val session = openSession(engine, site.address, Credential(CredentialKind.PAT, token = "t"))
                    val preview = rollbackSync(engine, session.sessionId, root.toString(), journal.id, dryRun = true, force = false)
                    assertEquals(2, preview.pages.count { it.result == "would-trash" }, describeRollback(preview))
                    assertEquals(3, site.pages.size, "a dry run changes nothing")

                    val done = rollbackSync(engine, session.sessionId, root.toString(), journal.id, dryRun = false, force = false)
                    assertTrue(done.complete, describeRollback(done))
                    assertEquals(1, site.pages.size)
                    assertEquals(emptyList(), listJournals(engine, root.toString()))
                    closeSession(engine, session.sessionId)
                }
            } finally {
                process.destroy()
                root.toFile().deleteRecursively()
            }
        }
    }

    @Test
    fun `the outcome is described in a line and the pages left alone are named`() {
        val result = SyncRollbackResult(
            pages = listOf(
                RollbackPage("1", "ENG: Home", "created", "trashed"),
                RollbackPage("2", "ENG: Setup", "updated", "skipped", "edited since the sync"),
            ),
            files = listOf(RollbackFile("a.md", "restored")),
        )

        assertEquals("1 trashed, 1 skipped, 1 restored\nENG: Setup: skipped (edited since the sync)", describeRollback(result))
        assertTrue(needsForce(result))
        assertEquals("Nothing to undo.", describeRollback(SyncRollbackResult()))
    }
}

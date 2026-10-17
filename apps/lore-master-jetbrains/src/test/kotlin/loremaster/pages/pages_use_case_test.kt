package loremaster.pages

import loremaster.engineprocess.EngineRpcClient
import loremaster.generators.describeRuns
import loremaster.generators.runGenerators
import loremaster.syncview.readPageTree
import loremaster.projectsettings.readProjectSettings
import java.nio.file.Files
import java.nio.file.Path
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

/** Pages, the page tree and the generators through the real engine, on a project with a github-pages output. */
class PagesUseCaseTest {
    private fun withProject(body: (EngineRpcClient, Path) -> Unit) {
        val binary = System.getenv("LORE_MASTER_ENGINE")?.takeIf { it.isNotBlank() }?.let(Path::of)
        if (binary == null || !Files.exists(binary)) {
            return
        }
        val root = Files.createTempDirectory("lore-master-pages")
        Files.writeString(root.resolve("README.md"), "# Home\n\nSee [guide](docs/guide.md).\n")
        Files.createDirectories(root.resolve("docs"))
        Files.writeString(root.resolve("docs/guide.md"), "# Guide\n\nHello.\n")
        Files.createDirectories(root.resolve(".vscode"))
        Files.writeString(root.resolve(".vscode/settings.json"), """{ "loreMaster.outputs": [ { "platform": "github-pages", "repo": "acme/docs" } ] }""")
        val process = ProcessBuilder(binary.toString()).redirectError(ProcessBuilder.Redirect.DISCARD).start()
        try {
            EngineRpcClient(process.inputStream, process.outputStream).use { body(it, root) }
        } finally {
            process.destroy()
            root.toFile().deleteRecursively()
        }
    }

    @Test
    fun `the site is built into a folder`() = withProject { engine, root ->
        val out = root.resolveSibling("lore-master-site-${System.nanoTime()}")
        try {
            val result = buildPages(engine, root.toString(), 0, out.toString())

            assertTrue(result.errors.isNullOrEmpty(), "errors: ${result.errors}")
            assertTrue(result.files >= 2, "files: ${result.files}")
            assertTrue(Files.exists(out.resolve("index.html")))
        } finally {
            out.toFile().deleteRecursively()
        }
    }

    @Test
    fun `the page tree lists the files the output would publish`() = withProject { engine, root ->
        val output = readProjectSettings(engine, root.toString()).settings.outputs.single()
        assertEquals("github-pages", output.platform)

        val tree = readPageTree(engine, root.toString(), 0)

        assertEquals(listOf("README.md", "docs/guide.md"), tree.nodes.map { it.path }.sorted())
    }

    @Test
    fun `no generators configured is said plainly`() = withProject { engine, root ->
        assertEquals("No generator is configured (loreMaster.generators).", describeRuns(runGenerators(engine, root.toString(), prepare = false)))
    }
}

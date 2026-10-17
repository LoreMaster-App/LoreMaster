package loremaster.exportmarkdown

import loremaster.engineprocess.EngineRpcClient
import java.nio.file.Files
import java.nio.file.Path
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

/** The export against the real engine. Needs LORE_MASTER_ENGINE (a built engine); without it nothing runs. */
class ExportMarkdownUseCaseTest {
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

    @Test
    fun `the engine lists its formats`() = withEngine { engine ->
        val ids = listExportFormats(engine).map { it.id }

        assertTrue(ids.containsAll(listOf("html", "txt", "docx", "odt", "pdf", "png")), "got $ids")
    }

    @Test
    fun `a document is exported next to itself, from the editor's unsaved text`() = withEngine { engine ->
        val folder = Files.createTempDirectory("lore-master-export")
        try {
            val file = folder.resolve("note.md")
            Files.writeString(file, "# On disk\n")

            val result = exportMarkdown(engine, ExportRunParams(file.toString(), "txt", content = "# Unsaved title\n\nBody"))

            assertEquals("note.txt", result.name)
            assertTrue(Files.readString(Path.of(result.path)).contains("Unsaved title"))
        } finally {
            folder.toFile().deleteRecursively()
        }
    }
}

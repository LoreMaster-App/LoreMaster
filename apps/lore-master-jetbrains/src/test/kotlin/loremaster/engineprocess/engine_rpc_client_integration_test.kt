package loremaster.engineprocess

import java.nio.file.Files
import java.nio.file.Path
import java.util.concurrent.TimeUnit
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

/**
 * The client against the real engine. Set LORE_MASTER_ENGINE to a built binary
 * (`go build -o <path> ./apps/lore-master-engine`); without it the test does nothing.
 */
class EngineRpcClientIntegrationTest {
    @Test
    fun `the real engine answers ping`() {
        val binary = System.getenv("LORE_MASTER_ENGINE")?.takeIf { it.isNotBlank() }?.let(Path::of)
        if (binary == null || !Files.exists(binary)) {
            return
        }

        val process = ProcessBuilder(binary.toString()).redirectError(ProcessBuilder.Redirect.DISCARD).start()
        try {
            EngineRpcClient(process.inputStream, process.outputStream).use { client ->
                val answer = client.request(METHOD_PING, null, PingResult::class.java).get(10, TimeUnit.SECONDS)

                assertEquals("pong", answer.pong)
                assertTrue(answer.version.isNotEmpty())
            }
        } finally {
            process.destroy()
        }
    }
}

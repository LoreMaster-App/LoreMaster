package loremaster.connection

import com.sun.net.httpserver.HttpServer
import loremaster.engineprocess.EngineRpcClient
import loremaster.engineprocess.EngineRpcError
import java.net.InetSocketAddress
import java.nio.file.Files
import java.nio.file.Path
import java.util.concurrent.ExecutionException
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith

/** session/open against the real engine and a fake Data Center site that accepts one personal access token. */
class SessionUseCaseTest {
    private val secret = "s3cret-token"

    private fun withSite(body: (String) -> Unit) {
        val site = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        site.createContext("/") { exchange ->
            val answer = when (exchange.requestURI.path) {
                "/rest/applinks/1.0/manifest" -> 200 to "<manifest><typeId>confluence</typeId><version>8.5.4</version></manifest>"
                "/rest/api/user/current" ->
                    if (exchange.requestHeaders.getFirst("Authorization") == "Bearer $secret") {
                        200 to """{"type":"known","username":"ada","userKey":"k1","displayName":"Ada Lovelace"}"""
                    } else {
                        401 to "{}"
                    }
                else -> 404 to ""
            }
            val bytes = answer.second.toByteArray()
            exchange.sendResponseHeaders(answer.first, if (bytes.isEmpty()) -1 else bytes.size.toLong())
            if (bytes.isNotEmpty()) exchange.responseBody.use { it.write(bytes) }
            exchange.close()
        }
        site.start()
        try {
            body("http://127.0.0.1:${site.address.port}")
        } finally {
            site.stop(0)
        }
    }

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
    fun `a good token opens a session and closing it is accepted`() = withEngine { engine ->
        withSite { address ->
            val session = openSession(engine, address, Credential(CredentialKind.PAT, token = secret))

            assertEquals("datacenter", session.edition)
            assertEquals("Ada Lovelace", session.user.displayName)
            closeSession(engine, session.sessionId)
        }
    }

    @Test
    fun `a refused token fails with the engine's unauthorized code`() = withEngine { engine ->
        withSite { address ->
            val failure = assertFailsWith<ExecutionException> {
                openSession(engine, address, Credential(CredentialKind.PAT, token = "wrong"))
            }

            assertEquals(-32001, (failure.cause as EngineRpcError).code)
        }
    }
}

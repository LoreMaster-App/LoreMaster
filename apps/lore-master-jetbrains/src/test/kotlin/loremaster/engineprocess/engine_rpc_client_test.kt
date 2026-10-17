package loremaster.engineprocess

import com.google.gson.JsonParser
import java.io.PipedInputStream
import java.io.PipedOutputStream
import java.util.concurrent.ExecutionException
import java.util.concurrent.LinkedBlockingQueue
import java.util.concurrent.TimeUnit
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith

/** The client against a scripted engine on in-memory pipes. */
class EngineRpcClientTest {
    private class Wire {
        val toClient = PipedOutputStream()
        val clientIn = PipedInputStream(toClient, 1 shl 16)
        val fromClient = PipedInputStream(1 shl 16)
        val clientOut = PipedOutputStream(fromClient)
    }

    @Test
    fun `a response is matched to its request and decoded`() {
        val wire = Wire()
        EngineRpcClient(wire.clientIn, wire.clientOut).use { client ->
            val reply = client.request(METHOD_PING, null, PingResult::class.java)
            assertEquals(1, requestId(wire))
            writeFrame(wire.toClient, """{"jsonrpc":"2.0","id":1,"result":{"pong":"pong","version":"1.2.3"}}""")

            assertEquals(PingResult("pong", "1.2.3"), reply.get(5, TimeUnit.SECONDS))
        }
    }

    @Test
    fun `an error response fails the request with the engine's code`() {
        val wire = Wire()
        EngineRpcClient(wire.clientIn, wire.clientOut).use { client ->
            val reply = client.request("nope", null, PingResult::class.java)
            requestId(wire)
            writeFrame(wire.toClient, """{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"no such method"}}""")

            val failure = assertFailsWith<ExecutionException> { reply.get(5, TimeUnit.SECONDS) }
            assertEquals(-32601, (failure.cause as EngineRpcError).code)
        }
    }

    @Test
    fun `a notification reaches the listener`() {
        val wire = Wire()
        val seen = LinkedBlockingQueue<String>()
        EngineRpcClient(wire.clientIn, wire.clientOut, onNotification = { method, _ -> seen.add(method) }).use {
            writeFrame(wire.toClient, """{"jsonrpc":"2.0","method":"host/progress","params":{}}""")

            assertEquals("host/progress", seen.poll(5, TimeUnit.SECONDS))
        }
    }

    @Test
    fun `a request from the engine is refused when there is no handler, so the engine can fall back`() {
        val wire = Wire()
        EngineRpcClient(wire.clientIn, wire.clientOut).use {
            writeFrame(wire.toClient, """{"jsonrpc":"2.0","id":7,"method":"host/renderDiagram","params":{}}""")

            val answer = JsonParser.parseString(readFrame(wire.fromClient)!!).asJsonObject
            assertEquals(7, answer.get("id").asInt)
            assertEquals(-32601, answer.getAsJsonObject("error").get("code").asInt)
        }
    }

    @Test
    fun `a request from the engine is answered by the handler, and a failure becomes an error`() {
        val wire = Wire()
        val handler: EngineRequestHandler = { method, _ -> if (method == "host/ok") mapOf("svg" to "<svg/>") else error("boom") }
        EngineRpcClient(wire.clientIn, wire.clientOut, onRequest = handler).use {
            writeFrame(wire.toClient, """{"jsonrpc":"2.0","id":1,"method":"host/ok"}""")
            writeFrame(wire.toClient, """{"jsonrpc":"2.0","id":2,"method":"host/bad"}""")

            val answers = (1..2).map { JsonParser.parseString(readFrame(wire.fromClient)!!).asJsonObject }.associateBy { it.get("id").asInt }
            assertEquals("<svg/>", answers.getValue(1).getAsJsonObject("result").get("svg").asString)
            assertEquals("boom", answers.getValue(2).getAsJsonObject("error").get("message").asString)
        }
    }

    private fun requestId(wire: Wire): Int {
        val body = readFrame(wire.fromClient)!!

        return JsonParser.parseString(body).asJsonObject.get("id").asInt
    }
}

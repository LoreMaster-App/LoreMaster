package loremaster.engineprocess

import com.google.gson.Gson
import com.google.gson.JsonElement
import com.google.gson.JsonObject
import java.io.InputStream
import java.io.OutputStream
import java.util.concurrent.CompletableFuture
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.Executors
import java.util.concurrent.atomic.AtomicLong

private const val METHOD_NOT_FOUND = -32601
private const val INTERNAL_ERROR = -32603

/** The engine failed a request. */
class EngineRpcError(val code: Int, message: String) : RuntimeException(message)

/**
 * Answers a request the engine makes of its host (the engine is the client there). Return the result, or throw
 * [EngineRpcError] to refuse; a method this does not know is refused by the caller.
 */
typealias EngineRequestHandler = (method: String, params: JsonElement?) -> Any?

/**
 * A JSON-RPC 2.0 client over the engine's stdio. Requests are matched to responses by id, so several can be in
 * flight at once (the engine answers them concurrently). Notifications from the engine go to [onNotification]; requests
 * from the engine go to [onRequest], and one that has no handler is answered "method not found" so the engine can
 * fall back (it draws Mermaid itself when the host will not).
 */
class EngineRpcClient(
    private val input: InputStream,
    private val output: OutputStream,
    private val onNotification: (method: String, params: JsonElement?) -> Unit = { _, _ -> },
    private val onRequest: EngineRequestHandler? = null,
) : AutoCloseable {
    private val gson = Gson()
    private val nextId = AtomicLong(1)
    private val pending = ConcurrentHashMap<Long, CompletableFuture<JsonElement?>>()
    private val answers = Executors.newCachedThreadPool { task -> Thread(task, "lore-master-engine-request").apply { isDaemon = true } }
    private val reader = Thread(::readLoop, "lore-master-engine-reader").apply { isDaemon = true }

    init {
        reader.start()
    }

    /** Calls one engine method and decodes its result as [type]. */
    fun <R> request(method: String, params: Any? = null, type: Class<R>): CompletableFuture<R> {
        val id = nextId.getAndIncrement()
        val reply = CompletableFuture<JsonElement?>()
        pending[id] = reply

        val message = JsonObject().apply {
            addProperty("jsonrpc", "2.0")
            addProperty("id", id)
            addProperty("method", method)
            if (params != null) {
                add("params", gson.toJsonTree(params))
            }
        }
        send(message)

        return reply.thenApply { gson.fromJson(it, type) }
    }

    override fun close() {
        failPending("the engine connection closed")
        answers.shutdownNow()
        runCatching { output.close() }
    }

    private fun send(message: JsonObject) = synchronized(output) { writeFrame(output, gson.toJson(message)) }

    private fun readLoop() {
        while (true) {
            val body = runCatching { readFrame(input) }.getOrNull() ?: break
            val message = gson.fromJson(body, JsonObject::class.java)
            val id = message.get("id")
            val method = message.get("method")?.asString
            when {
                method != null && (id == null || id.isJsonNull) -> onNotification(method, message.get("params"))
                method != null -> runCatching { answers.execute { answer(id, method, message.get("params")) } }
                id != null -> finish(id.asLong, message)
            }
        }
        failPending("the engine stopped")
    }

    private fun finish(id: Long, message: JsonObject) {
        val reply = pending.remove(id) ?: return
        val error = message.getAsJsonObject("error")
        if (error != null) {
            reply.completeExceptionally(EngineRpcError(error.get("code").asInt, error.get("message").asString))
        } else {
            reply.complete(message.get("result"))
        }
    }

    private fun answer(id: JsonElement, method: String, params: JsonElement?) {
        val response = JsonObject().apply {
            addProperty("jsonrpc", "2.0")
            add("id", id)
        }
        val handler = onRequest
        if (handler == null) {
            response.add("error", errorObject(METHOD_NOT_FOUND, "the host does not answer $method"))
        } else {
            try {
                response.add("result", gson.toJsonTree(handler(method, params)))
            } catch (refusal: EngineRpcError) {
                response.add("error", errorObject(refusal.code, refusal.message ?: "refused"))
            } catch (failure: Exception) {
                response.add("error", errorObject(INTERNAL_ERROR, failure.message ?: failure.javaClass.simpleName))
            }
        }
        runCatching { send(response) }
    }

    private fun errorObject(code: Int, message: String) = JsonObject().apply {
        addProperty("code", code)
        addProperty("message", message)
    }

    private fun failPending(reason: String) {
        pending.values.forEach { it.completeExceptionally(IllegalStateException(reason)) }
        pending.clear()
    }
}

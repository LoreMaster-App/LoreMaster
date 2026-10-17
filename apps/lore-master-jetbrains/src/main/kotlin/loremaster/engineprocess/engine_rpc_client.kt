package loremaster.engineprocess

import com.google.gson.Gson
import com.google.gson.JsonElement
import com.google.gson.JsonObject
import java.io.InputStream
import java.io.OutputStream
import java.util.concurrent.CompletableFuture
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.atomic.AtomicLong

/** The engine failed a request. */
class EngineRpcError(val code: Int, message: String) : RuntimeException(message)

/**
 * A JSON-RPC 2.0 client over the engine's stdio. Requests are matched to responses by id, so several can be in
 * flight at once (the engine answers them concurrently). Notifications from the engine go to [onNotification].
 */
class EngineRpcClient(
    private val input: InputStream,
    private val output: OutputStream,
    private val onNotification: (method: String, params: JsonElement?) -> Unit = { _, _ -> },
) : AutoCloseable {
    private val gson = Gson()
    private val nextId = AtomicLong(1)
    private val pending = ConcurrentHashMap<Long, CompletableFuture<JsonElement?>>()
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
        synchronized(output) { writeFrame(output, gson.toJson(message)) }

        return reply.thenApply { gson.fromJson(it, type) }
    }

    override fun close() {
        failPending("the engine connection closed")
        runCatching { output.close() }
    }

    private fun readLoop() {
        while (true) {
            val body = runCatching { readFrame(input) }.getOrNull() ?: break
            val message = gson.fromJson(body, JsonObject::class.java)
            val id = message.get("id")
            if (id == null || id.isJsonNull) {
                onNotification(message.get("method").asString, message.get("params"))
                continue
            }
            val reply = pending.remove(id.asLong) ?: continue
            val error = message.getAsJsonObject("error")
            if (error != null) {
                reply.completeExceptionally(EngineRpcError(error.get("code").asInt, error.get("message").asString))
            } else {
                reply.complete(message.get("result"))
            }
        }
        failPending("the engine stopped")
    }

    private fun failPending(reason: String) {
        pending.values.forEach { it.completeExceptionally(IllegalStateException(reason)) }
        pending.clear()
    }
}

package loremaster.testing

import com.google.gson.Gson
import com.google.gson.JsonObject
import com.sun.net.httpserver.HttpExchange
import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import java.net.URLDecoder

/**
 * An in-process Confluence Data Center (REST v1), faithful to the one dialect the engine uses: pages are matched by title and
 * parent, never by body. Seeded with one page, the space's home, which outputs use as their parent. Mirrors the fake in
 * sync.integration.spec.ts of the VS Code extension.
 */
class FakeConfluenceFixture : AutoCloseable {
    class Page(val id: String, var title: String, val spaceKey: String, var parentId: String, var version: Int = 1, var marked: Boolean = false) {
        var property: Pair<Any?, Int>? = null
    }

    val pages = mutableListOf<Page>()
    val root: Page = store("ENG", "", "Engineering Home")
    private val gson = Gson()
    private val server: HttpServer = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)

    val address: String get() = "http://127.0.0.1:${server.address.port}"

    init {
        server.createContext("/") { exchange -> exchange.use(::handle) }
        server.start()
    }

    private fun store(space: String, parent: String, title: String): Page = Page("p${pages.size + 1}", title, space, parent).also(pages::add)

    private fun find(id: String) = pages.firstOrNull { it.id == id }

    private fun taken(space: String, title: String) = pages.any { it.spaceKey == space && it.title.equals(title, ignoreCase = true) }

    private fun below(start: String, rootId: String): Boolean {
        var id = start
        repeat(pages.size + 1) {
            val page = find(id)
            if (page == null || page.parentId.isEmpty()) return false
            if (page.parentId == rootId) return true
            id = page.parentId
        }

        return false
    }

    private fun wire(page: Page): Map<String, Any?> = buildMap {
        put("id", page.id)
        put("title", page.title)
        put("space", mapOf("id" to 1, "key" to page.spaceKey))
        put("version", mapOf("number" to page.version))
        if (page.parentId.isNotEmpty()) put("ancestors", listOf(mapOf("id" to page.parentId)))
        put("_links", mapOf("webui" to "/display/${page.spaceKey}/${page.id}", "base" to ""))
    }

    private fun cqlValue(cql: String, prefix: String): String {
        val index = cql.indexOf(prefix)
        if (index < 0) return ""
        val rest = cql.substring(index + prefix.length)
        if (!rest.startsWith("\"")) return ""
        val value = StringBuilder()
        var i = 1
        while (i < rest.length) {
            when {
                rest[i] == '\\' && i + 1 < rest.length -> value.append(rest[++i])
                rest[i] == '"' -> break
                else -> value.append(rest[i])
            }
            i++
        }

        return value.toString()
    }

    private fun send(exchange: HttpExchange, status: Int, body: Any) {
        val bytes = gson.toJson(body).toByteArray()
        exchange.responseHeaders.add("Content-Type", "application/json")
        exchange.sendResponseHeaders(status, bytes.size.toLong())
        exchange.responseBody.write(bytes)
    }

    private fun list(exchange: HttpExchange, results: List<Any?>) =
        send(exchange, 200, mapOf("results" to results, "size" to results.size, "limit" to 250, "_links" to emptyMap<String, Any>()))

    @Synchronized
    private fun handle(exchange: HttpExchange) {
        val path = exchange.requestURI.path
        val method = exchange.requestMethod
        val query = (exchange.requestURI.rawQuery ?: "").split("&").filter { it.isNotEmpty() }
            .associate { it.substringBefore("=") to URLDecoder.decode(it.substringAfter("=", ""), "UTF-8") }
        val raw = exchange.requestBody.readBytes().toString(Charsets.UTF_8)
        val body = if (raw.isBlank()) JsonObject() else gson.fromJson(raw, JsonObject::class.java)
        val id = Regex("^/rest/api/content/([^/]+)").find(path)?.groupValues?.get(1).orEmpty()

        when {
            path == "/rest/api/content" && method == "POST" -> {
                val space = body.getAsJsonObject("space").get("key").asString
                val title = body.get("title").asString
                if (taken(space, title)) return send(exchange, 400, mapOf("message" to "A page with this title already exists: $title"))
                val parent = body.getAsJsonArray("ancestors")?.lastOrNull()?.asJsonObject?.get("id")?.asString.orEmpty()
                send(exchange, 200, wire(store(space, parent, title)))
            }
            path == "/rest/api/content/search" && method == "GET" -> {
                val cql = query["cql"].orEmpty()
                if (cql.contains("label = \"lore-master\"")) {
                    val ancestor = cqlValue(cql, "ancestor = ")
                    list(exchange, pages.filter { it.marked && below(it.id, ancestor) }.map(::wire))
                } else {
                    val space = cqlValue(cql, "space = ")
                    val title = cqlValue(cql, "title = ")
                    list(exchange, pages.filter { it.spaceKey == space && it.title.equals(title, ignoreCase = true) }.map(::wire))
                }
            }
            Regex("^/rest/api/content/[^/]+/child/page$").matches(path) && method == "GET" ->
                list(exchange, pages.filter { it.parentId == id }.map(::wire))
            Regex("^/rest/api/content/[^/]+/label$").matches(path) && method == "POST" -> {
                find(id)?.marked = true
                send(exchange, 200, emptyMap<String, Any>())
            }
            Regex("^/rest/api/content/[^/]+/property/[^/]+$").matches(path) && method == "GET" -> {
                val property = find(id)?.property
                if (property == null) send(exchange, 404, mapOf("message" to "No property found"))
                else send(exchange, 200, mapOf("value" to property.first, "version" to mapOf("number" to property.second)))
            }
            Regex("^/rest/api/content/[^/]+/property$").matches(path) && method == "POST" -> {
                val page = find(id) ?: return send(exchange, 404, mapOf("message" to "No content found"))
                page.property = gson.fromJson(body.get("value"), Any::class.java) to 1
                send(exchange, 200, emptyMap<String, Any>())
            }
            Regex("^/rest/api/content/[^/]+/property/[^/]+$").matches(path) && method == "PUT" -> {
                val page = find(id) ?: return send(exchange, 404, mapOf("message" to "No content found"))
                page.property = gson.fromJson(body.get("value"), Any::class.java) to body.getAsJsonObject("version").get("number").asInt
                send(exchange, 200, emptyMap<String, Any>())
            }
            Regex("^/rest/api/content/[^/]+$").matches(path) && method == "PUT" -> {
                val page = find(id) ?: return send(exchange, 404, mapOf("message" to "No content found"))
                val number = body.getAsJsonObject("version").get("number").asInt
                if (number != page.version + 1) return send(exchange, 409, mapOf("message" to "Version must be incremented on update"))
                page.title = body.get("title").asString
                page.parentId = body.getAsJsonArray("ancestors")?.lastOrNull()?.asJsonObject?.get("id")?.asString ?: page.parentId
                page.version = number
                send(exchange, 200, wire(page))
            }
            Regex("^/rest/api/content/[^/]+$").matches(path) && method == "DELETE" -> {
                // Confluence moves the page to the trash; for the fake it is gone from every listing.
                val page = find(id) ?: return send(exchange, 404, mapOf("message" to "No content found"))
                pages.remove(page)
                exchange.sendResponseHeaders(204, -1)
            }
            Regex("^/rest/api/content/[^/]+$").matches(path) && method == "GET" ->
                find(id)?.let { send(exchange, 200, wire(it)) } ?: send(exchange, 404, mapOf("message" to "No content found"))
            path == "/rest/api/space" && method == "GET" ->
                list(exchange, listOf(mapOf("id" to 1, "key" to "ENG", "name" to "Engineering", "type" to "global", "homepage" to mapOf("id" to root.id))))
            path == "/rest/applinks/1.0/manifest" ->
                exchange.responseHeaders.add("Content-Type", "application/xml").also {
                    val bytes = "<manifest><typeId>confluence</typeId><version>8.5.4</version></manifest>".toByteArray()
                    exchange.sendResponseHeaders(200, bytes.size.toLong())
                    exchange.responseBody.write(bytes)
                }
            path == "/rest/api/user/current" && method == "GET" ->
                if (exchange.requestHeaders.getFirst("Authorization") != null) {
                    send(exchange, 200, mapOf("type" to "known", "username" to "ada", "userKey" to "k1", "displayName" to "Ada Lovelace"))
                } else {
                    send(exchange, 401, mapOf("message" to "Unauthorized"))
                }
            else -> send(exchange, 404, mapOf("message" to "unexpected $method $path"))
        }
    }

    override fun close() = server.stop(0)
}

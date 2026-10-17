package loremaster.outputs

import com.google.gson.Gson
import com.google.gson.JsonArray
import com.google.gson.JsonObject
import loremaster.engineprocess.EngineRpcClient
import java.util.concurrent.TimeUnit

private const val METHOD_SETTINGS_READ = "settings/read"
private const val METHOD_SETTINGS_SAVE = "settings/save"
private val gson = Gson()

/** What the picker decided: where the pages of the project go. */
data class ConfluenceOutputChoice(
    val baseUrl: String,
    val space: String,
    val parentPageId: String,
    val titlePrefix: String,
)

/** The output as `loreMaster.outputs` holds it, with the defaults the engine would apply written out. */
fun outputObject(choice: ConfluenceOutputChoice): JsonObject = JsonObject().apply {
    addProperty("platform", "confluence")
    addProperty("baseUrl", choice.baseUrl)
    addProperty("space", choice.space)
    addProperty("parentPageId", choice.parentPageId)
    addProperty("titlePrefix", choice.titlePrefix)
    addProperty("direction", "to-platform")
    add(
        "content",
        JsonArray().apply {
            add(
                JsonObject().apply {
                    addProperty("type", "markdown")
                    add("roots", JsonArray().apply { add(".") })
                    add("excludes", JsonArray())
                    addProperty("template", "default")
                },
            )
        },
    )
    addProperty("mermaidMode", "image")
    addProperty("titleCollision", "adopt")
    addProperty("linkMode", "title")
}

/** An output with nowhere to go yet: what the engine reads back for a project that has no configuration. */
private fun isPlaceholder(output: com.google.gson.JsonElement): Boolean {
    val fields = output.asJsonObject
    return listOf("baseUrl", "space", "parentPageId", "repo").all { fields.get(it)?.asString.isNullOrBlank() }
}

/**
 * Adds a Confluence output to the project's `loreMaster.outputs`, keeping everything else the settings hold. The engine reads the
 * whole configuration, we add one output to it, and the engine validates and writes it back (to `.vscode/settings.json`, keeping
 * the author's comments). Returns the index of the new output.
 */
fun addConfluenceOutput(engine: EngineRpcClient, workspaceRoot: String, choice: ConfluenceOutputChoice): Int {
    val read = engine.request(METHOD_SETTINGS_READ, mapOf("workspaceRoot" to workspaceRoot), JsonObject::class.java).get(30, TimeUnit.SECONDS)
    val settings = read.getAsJsonObject("settings")
    // A project with no configuration reads back with one empty placeholder output; it is replaced, not kept beside the new one.
    val outputs = JsonArray().also { kept ->
        settings.getAsJsonArray("outputs")?.toList().orEmpty().filterNot(::isPlaceholder).forEach(kept::add)
        settings.add("outputs", kept)
    }
    outputs.add(outputObject(choice))
    engine.request(METHOD_SETTINGS_SAVE, mapOf("workspaceRoot" to workspaceRoot, "settings" to gson.toJsonTree(settings)), Any::class.java)
        .get(30, TimeUnit.SECONDS)

    return outputs.size() - 1
}

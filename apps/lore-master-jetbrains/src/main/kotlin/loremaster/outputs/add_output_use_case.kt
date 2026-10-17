package loremaster.outputs

import com.google.gson.Gson
import com.google.gson.JsonArray
import com.google.gson.JsonElement
import com.google.gson.JsonObject
import loremaster.configdescription.ConfigDescription
import loremaster.engineprocess.EngineRpcClient
import java.util.concurrent.TimeUnit

private const val METHOD_SETTINGS_READ = "settings/read"
private const val METHOD_SETTINGS_SAVE = "settings/save"
private val gson = Gson()

/**
 * An output with nowhere to go yet: what the engine reads back for a project that has no configuration. It is a platform's output
 * whose required settings are all empty; a platform with nothing required (a site on this repository's origin) is never one.
 */
private fun isPlaceholder(description: ConfigDescription, output: JsonElement): Boolean {
    val fields = output.asJsonObject
    val required = description.platforms.firstOrNull { it.id == fields.get("platform")?.asString }?.fields.orEmpty().filter { it.required }

    return required.isNotEmpty() && required.all { fields.get(it.key)?.asString.isNullOrBlank() }
}

/**
 * Adds [output] to the project's `loreMaster.outputs`, keeping everything else the settings hold. The engine reads the whole
 * configuration, we add one output to it, and the engine validates and writes it back (to `.vscode/settings.json`, keeping the
 * author's comments). Returns the index of the new output.
 */
fun addOutput(engine: EngineRpcClient, workspaceRoot: String, description: ConfigDescription, output: JsonObject): Int {
    val read = engine.request(METHOD_SETTINGS_READ, mapOf("workspaceRoot" to workspaceRoot), JsonObject::class.java).get(30, TimeUnit.SECONDS)
    val settings = read.getAsJsonObject("settings")
    // A project with no configuration reads back with one empty placeholder output; it is replaced, not kept beside the new one.
    val outputs = JsonArray().also { kept ->
        settings.getAsJsonArray("outputs")?.toList().orEmpty().filterNot { isPlaceholder(description, it) }.forEach(kept::add)
        settings.add("outputs", kept)
    }
    outputs.add(output)
    engine.request(METHOD_SETTINGS_SAVE, mapOf("workspaceRoot" to workspaceRoot, "settings" to gson.toJsonTree(settings)), Any::class.java)
        .get(30, TimeUnit.SECONDS)

    return outputs.size() - 1
}

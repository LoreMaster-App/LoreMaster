package loremaster.outputs

import com.google.gson.JsonArray
import com.google.gson.JsonObject
import loremaster.configdescription.ConfigDescription
import loremaster.configdescription.FIELD_CHOICE
import loremaster.configdescription.FIELD_LIST
import loremaster.configdescription.FIELD_NUMBER
import loremaster.configdescription.FieldDescriptor
import loremaster.configdescription.PlatformDescriptor

private fun textArray(items: List<String>): JsonArray = JsonArray().also { array -> items.forEach(array::add) }

/** The Markdown content every new output starts with: the whole workspace. */
private fun defaultContent(roots: List<String>, excludes: List<String>): JsonArray = JsonArray().apply {
    add(
        JsonObject().apply {
            addProperty("type", "markdown")
            add("roots", textArray(roots))
            add("excludes", textArray(excludes))
            addProperty("template", "default")
        },
    )
}

/** The content of an output being edited with its Markdown entry's folders and excluded paths set and the rest kept. */
private fun contentWith(existing: JsonArray?, roots: List<String>, excludes: List<String>): JsonArray {
    val entries = existing?.deepCopy() ?: return defaultContent(roots, excludes)
    val markdown = entries.map { it.asJsonObject }.firstOrNull { it.get("type")?.asString == "markdown" } ?: return defaultContent(roots, excludes)
    markdown.add("roots", textArray(roots))
    markdown.add("excludes", textArray(excludes))

    return entries
}

/** Every setting an output of any platform can hold: the platforms' own fields and the ones every output has. */
private fun allFields(description: ConfigDescription): List<FieldDescriptor> =
    (description.outputFields + description.platforms.flatMap { it.fields }).distinctBy { it.key }

/**
 * A new output of [platform]: every setting of every platform present and empty, a choice on its default, and the whole workspace's
 * Markdown as content. What the platform does not use is left empty for the engine to ignore.
 */
fun blankOutput(description: ConfigDescription, platform: PlatformDescriptor): JsonObject = JsonObject().apply {
    addProperty("platform", platform.id)
    for (field in allFields(description)) {
        when {
            field.type == FIELD_LIST || field.type == FIELD_NUMBER || field.key == "name" -> Unit
            field.type == FIELD_CHOICE -> addProperty(field.key, field.default)
            else -> addProperty(field.key, "")
        }
    }
    add("content", defaultContent(listOf("."), emptyList()))
}

/** The fields a form for [platform] shows: its own, then the ones every output has. */
fun formFields(description: ConfigDescription, platform: PlatformDescriptor): List<FieldDescriptor> = platform.fields + description.outputFields

/** Comma separated text as a list of non-blank trimmed entries. */
fun splitList(text: String): List<String> = text.split(',').map { it.trim() }.filter { it.isNotEmpty() }

/**
 * The output the [values] of a form describe, by field key: the platform's settings and the shared ones as written (trimmed), the
 * content's folders and excluded paths from `roots` and `excludes`. A blank name is left out so the engine makes one up. When
 * [base] is given (an output being edited) everything the form does not show is kept as it was.
 */
fun outputFrom(description: ConfigDescription, platform: PlatformDescriptor, values: Map<String, String>, base: JsonObject? = null): JsonObject {
    val output = base?.deepCopy() ?: blankOutput(description, platform)
    for (field in formFields(description, platform)) {
        // A choice left blank, or not given, takes its default, as in a new output.
        val text = values[field.key].orEmpty().trim().ifEmpty { if (field.type == FIELD_CHOICE) field.default else "" }
        if (field.type == FIELD_LIST) {
            output.remove(field.key)
            val items = splitList(text)
            if (items.isNotEmpty()) output.add(field.key, JsonArray().also { array -> items.forEach(array::add) })
        } else if (field.type == FIELD_NUMBER) {
            // The engine wants a number or nothing, never text.
            output.remove(field.key)
            text.toIntOrNull()?.let { output.addProperty(field.key, it) }
        } else if (field.key == "name" && text.isEmpty()) {
            output.remove("name")
        } else {
            output.addProperty(field.key, text)
        }
    }
    val roots = splitList(values["roots"].orEmpty()).ifEmpty { listOf(".") }
    output.add("content", contentWith(output.getAsJsonArray("content"), roots, splitList(values["excludes"].orEmpty())))

    return output
}

/** What is wrong with the values, by field key; empty means the output can be saved. Only what the description states is checked. */
fun validateOutputValues(description: ConfigDescription, platform: PlatformDescriptor, values: Map<String, String>): Map<String, String> {
    val errors = linkedMapOf<String, String>()
    for (field in formFields(description, platform)) {
        val text = values[field.key].orEmpty().trim()
        when {
            text.isEmpty() && field.required -> errors[field.key] = "${field.label} is required."
            text.isNotEmpty() && field.pattern.isNotEmpty() && !Regex(field.pattern).matches(text) ->
                errors[field.key] = field.patternHelp.ifBlank { "${field.label} is not valid." }
        }
    }

    return errors
}

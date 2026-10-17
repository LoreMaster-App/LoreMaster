package loremaster.pluginsettings

import loremaster.configdescription.ConfigDescription
import loremaster.configdescription.FieldDescriptor

/** The engine's description of a setting, wherever it is grouped. */
fun settingDescribed(description: ConfigDescription, key: String): FieldDescriptor? =
    description.sections.asSequence().flatMap { it.settings.asSequence() }.firstOrNull { it.key == key }

/** What a described setting is now: the user's value, else the default the engine states. */
fun effectiveValue(description: ConfigDescription, settings: PluginSettings, key: String): String =
    settings.value(key).ifBlank { settingDescribed(description, key)?.default.orEmpty() }

/** A described number setting as the engine bounds it; the user's value when it is a number in range, else the default. */
fun effectiveNumber(description: ConfigDescription, settings: PluginSettings, key: String): Int? {
    val field = settingDescribed(description, key) ?: return null
    val chosen = settings.value(key).toIntOrNull() ?: field.default.toIntOrNull() ?: return null

    return chosen.coerceIn(field.min ?: Int.MIN_VALUE, field.max ?: Int.MAX_VALUE)
}

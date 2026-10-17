package loremaster.pluginsettings

import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.components.PersistentStateComponent
import com.intellij.openapi.components.Service
import com.intellij.openapi.components.State
import com.intellij.openapi.components.Storage

/**
 * What the user sets in Settings > Tools > LoreMaster. Blank means "the default".
 *
 * Only [enginePath] belongs to this plugin: the shell must find the engine before it can ask anything. Every other setting is one
 * the engine describes (`config/describe`), kept by its key in [values] ("export.pageSize", "watchDebounceSeconds").
 */
data class PluginSettings(
    var enginePath: String = "",
    var values: MutableMap<String, String> = mutableMapOf(),
    // The settings as they were kept before the page was built from the engine's description; read once, then cleared.
    var browserPath: String = "",
    var exportFolder: String = "",
    var pageSize: String = "",
    var pngWidth: Int = 0,
    var docxTemplate: String = "",
    var odtTemplate: String = "",
    var watchDebounceSeconds: Int = 0,
) {
    /** The value the user set for a described setting; blank when they left it on the default. */
    fun value(key: String): String = values[key].orEmpty()

    /** Sets a described setting; blank puts it back to the default. */
    fun set(key: String, value: String) {
        if (value.isBlank()) values.remove(key) else values[key] = value
    }

    /** The same settings with any value kept in the old fields moved into [values]. */
    fun migrated(): PluginSettings {
        val old = mapOf(
            "export.browserPath" to browserPath,
            "export.folder" to exportFolder,
            "export.pageSize" to pageSize.takeIf { it != "A4" }.orEmpty(),
            "export.pngWidth" to pngWidth.takeIf { it > 0 && it != 1000 }?.toString().orEmpty(),
            "export.docxTemplate" to docxTemplate,
            "export.odtTemplate" to odtTemplate,
            "watchDebounceSeconds" to watchDebounceSeconds.takeIf { it > 0 && it != 2 }?.toString().orEmpty(),
        )
        val moved = old.filterValues { it.isNotBlank() }.filterKeys { it !in values }

        return copy(values = (values + moved).toMutableMap(), browserPath = "", exportFolder = "", pageSize = "", pngWidth = 0, docxTemplate = "", odtTemplate = "", watchDebounceSeconds = 0)
    }
}

/** The IDE-wide preferences of the plugin, kept in loremaster.xml. Connections and secrets are not here. */
@Service(Service.Level.APP)
@State(name = "LoreMasterSettings", storages = [Storage("loremaster.xml")])
class PluginSettingsStore : PersistentStateComponent<PluginSettings> {
    private var state = PluginSettings()

    override fun getState(): PluginSettings = state

    override fun loadState(loaded: PluginSettings) {
        state = loaded.migrated()
    }

    companion object {
        fun current(): PluginSettings = ApplicationManager.getApplication().getService(PluginSettingsStore::class.java).state
    }
}

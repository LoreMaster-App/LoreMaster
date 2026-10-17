package loremaster.pluginsettings

import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.components.PersistentStateComponent
import com.intellij.openapi.components.Service
import com.intellij.openapi.components.State
import com.intellij.openapi.components.Storage

/** What the user sets in Settings > Tools > LoreMaster. Blank means "the default". */
data class PluginSettings(
    var enginePath: String = "",
    var browserPath: String = "",
    var exportFolder: String = "",
    var pageSize: String = "A4",
    var pngWidth: Int = 1000,
    var docxTemplate: String = "",
    var odtTemplate: String = "",
)

/** The IDE-wide preferences of the plugin, kept in loremaster.xml. Connections and secrets are not here. */
@Service(Service.Level.APP)
@State(name = "LoreMasterSettings", storages = [Storage("loremaster.xml")])
class PluginSettingsStore : PersistentStateComponent<PluginSettings> {
    private var state = PluginSettings()

    override fun getState(): PluginSettings = state

    override fun loadState(loaded: PluginSettings) {
        state = loaded
    }

    companion object {
        fun current(): PluginSettings = ApplicationManager.getApplication().getService(PluginSettingsStore::class.java).state
    }
}

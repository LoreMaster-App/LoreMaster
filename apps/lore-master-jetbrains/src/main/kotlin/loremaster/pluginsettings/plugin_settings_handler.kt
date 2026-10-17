package loremaster.pluginsettings

import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.fileChooser.FileChooserDescriptorFactory
import com.intellij.openapi.options.BoundSearchableConfigurable
import com.intellij.openapi.progress.ProgressManager
import com.intellij.openapi.ui.DialogPanel
import com.intellij.ui.dsl.builder.COLUMNS_LARGE
import com.intellij.ui.dsl.builder.Cell
import com.intellij.ui.dsl.builder.Row
import com.intellij.ui.dsl.builder.bindIntText
import com.intellij.ui.dsl.builder.bindItem
import com.intellij.ui.dsl.builder.bindSelected
import com.intellij.ui.dsl.builder.bindText
import com.intellij.ui.dsl.builder.columns
import com.intellij.ui.dsl.builder.panel
import loremaster.configdescription.ConfigDescription
import loremaster.configdescription.FIELD_BOOL
import loremaster.configdescription.FIELD_CHOICE
import loremaster.configdescription.FIELD_NUMBER
import loremaster.configdescription.FIELD_PATH
import loremaster.configdescription.FieldDescriptor
import loremaster.configdescription.SectionDescriptor
import loremaster.configdescription.describeConfig
import loremaster.engineprocess.EngineProcessStore

/** This plugin's name in the engine's descriptions of which editors honour a setting. */
private const val THIS_SHELL = "jetbrains"

/** The settings of the user (not the project's) that this plugin acts on, in the sections the engine groups them in. */
fun userSections(description: ConfigDescription): List<SectionDescriptor> = description.sections
    .filter { it.scope == "user" }
    .map { section -> section.copy(settings = section.settings.filter { it.shells.isEmpty() || THIS_SHELL in it.shells }) }
    .filter { it.settings.isNotEmpty() }

/**
 * Settings > Tools > LoreMaster. The engine path is this plugin's own (the engine must be found before it can be asked anything);
 * every other setting is drawn from what the engine describes, so a setting the engine adds appears here without a change.
 */
class PluginSettingsHandler(private val describe: () -> ConfigDescription?) : BoundSearchableConfigurable("LoreMaster", "loremaster.settings") {
    constructor() : this(::askTheEngine)

    override fun createPanel(): DialogPanel {
        val settings = PluginSettingsStore.current()
        val description = describe()

        return panel {
            group("Engine") {
                row("Engine path:") {
                    textField().columns(COLUMNS_LARGE).bindText(settings::enginePath)
                        .comment("A locally built lore-master-engine. Blank uses the one bundled with the plugin (or the LORE_MASTER_ENGINE variable).")
                }
            }
            if (description == null) {
                row { comment("The engine did not start, so its settings cannot be shown. Check the engine path above.") }

                return@panel
            }
            for (section in userSections(description)) {
                group(section.label) {
                    for (field in section.settings) row(field.label + ":") { control(field, settings) }
                }
            }
        }
    }

    /** The control for one described setting, kept in [PluginSettings.values] under its key. */
    private fun Row.control(field: FieldDescriptor, settings: PluginSettings) {
        val cell: Cell<*> = when (field.type) {
            FIELD_BOOL -> checkBox("").bindSelected(
                { settings.value(field.key).ifBlank { field.default } == "true" },
                { settings.set(field.key, if (it == (field.default == "true")) "" else it.toString()) },
            )
            FIELD_CHOICE -> comboBox(field.choices.map { it.value }).bindItem(
                { settings.value(field.key).ifBlank { field.default.ifBlank { field.choices.firstOrNull()?.value } } },
                { settings.set(field.key, if (it == field.default) "" else it.orEmpty()) },
            )
            FIELD_NUMBER -> intTextField((field.min ?: 0)..(field.max ?: Int.MAX_VALUE)).bindIntText(
                { settings.value(field.key).toIntOrNull() ?: field.default.toIntOrNull() ?: 0 },
                { settings.set(field.key, if (it == field.default.toIntOrNull()) "" else it.toString()) },
            )
            FIELD_PATH -> textFieldWithBrowseButton(
                (if (field.pathKind == "folder") FileChooserDescriptorFactory.singleDir() else FileChooserDescriptorFactory.singleFile())
                    .withTitle(field.label),
            ).columns(COLUMNS_LARGE).bindText({ settings.value(field.key) }, { settings.set(field.key, it) })
            else -> textField().columns(COLUMNS_LARGE).bindText({ settings.value(field.key) }, { settings.set(field.key, it) })
        }
        if (field.help.isNotBlank()) cell.comment(field.help)
    }
}

/** Asks the engine what can be configured, off the UI thread behind a progress dialog; null when the engine cannot be reached. */
private fun askTheEngine(): ConfigDescription? = runCatching {
    ProgressManager.getInstance().runProcessWithProgressSynchronously<ConfigDescription, Exception>(
        { describeConfig(ApplicationManager.getApplication().getService(EngineProcessStore::class.java).connection()) },
        "Loading the LoreMaster settings",
        true,
        null,
    )
}.getOrNull()


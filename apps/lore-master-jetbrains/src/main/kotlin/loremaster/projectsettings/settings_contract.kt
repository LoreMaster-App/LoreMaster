package loremaster.projectsettings

/** The engine's `settings/read`: the project's configuration as the engine understands it, defaults applied. */
const val METHOD_SETTINGS_READ = "settings/read"

data class SettingsReadParams(val workspaceRoot: String)

data class SettingsReadResult(
    val exists: Boolean = false,
    val firstSync: Boolean = false,
    val settings: ProjectSettings = ProjectSettings(),
    /** What the engine calls each output, in the order of the outputs: its name, or one made up from where it goes. */
    val displayNames: List<String> = emptyList(),
    /** Where each output goes (the site address, the repository), in the same order. */
    val summaries: List<String> = emptyList(),
)

/** The parts of the configuration this plugin shows; the engine sends more. */
data class ProjectSettings(
    val outputs: List<OutputSettings> = emptyList(),
    val generators: List<GeneratorSettings> = emptyList(),
)

/** One place the lore goes. [platform] is the id of one of the engine's platforms (see `config/describe`). */
data class OutputSettings(
    val name: String = "",
    val platform: String = "",
    val baseUrl: String = "",
    val space: String = "",
    val parentPageId: String = "",
    val titlePrefix: String = "",
    val repo: String = "",
    val branch: String = "",
    /** What the engine calls this output; filled in when the settings are read. */
    val displayName: String = "",
    /** Where it goes; filled in when the settings are read. */
    val summary: String = "",
) {
    /** What the output is called in a picker: the engine's name for it. */
    fun label(): String = displayName.ifBlank { name.ifBlank { platform } }
}

data class GeneratorSettings(val type: String = "", val output: String = "", val title: String = "")

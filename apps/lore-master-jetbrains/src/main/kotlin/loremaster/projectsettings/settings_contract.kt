package loremaster.projectsettings

/** The engine's `settings/read`: the project's configuration as the engine understands it, defaults applied. */
const val METHOD_SETTINGS_READ = "settings/read"

data class SettingsReadParams(val workspaceRoot: String)

data class SettingsReadResult(
    val exists: Boolean = false,
    val firstSync: Boolean = false,
    val settings: ProjectSettings = ProjectSettings(),
)

/** The parts of the configuration this plugin shows; the engine sends more. */
data class ProjectSettings(
    val outputs: List<OutputSettings> = emptyList(),
    val generators: List<GeneratorSettings> = emptyList(),
)

/** One place the lore goes. [platform] is confluence, github-pages or github-wiki. */
data class OutputSettings(
    val name: String = "",
    val platform: String = "",
    val baseUrl: String = "",
    val space: String = "",
    val parentPageId: String = "",
    val titlePrefix: String = "",
    val repo: String = "",
    val branch: String = "",
) {
    /** What the output is called in a picker: its name, else where it goes. */
    fun label(): String = name.ifBlank {
        when (platform) {
            "confluence" -> "Confluence ${space.ifBlank { baseUrl }}"
            else -> "$platform ${repo.ifBlank { branch }}".trim()
        }
    }
}

data class GeneratorSettings(val type: String = "", val output: String = "", val title: String = "")

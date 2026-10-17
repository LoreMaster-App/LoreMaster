package loremaster.syncview

import loremaster.configdescription.ConfigDescription
import loremaster.configdescription.describeConfig
import loremaster.configdescription.platformOf
import loremaster.engineprocess.EngineRpcClient
import loremaster.projectsettings.OutputSettings
import loremaster.projectsettings.readProjectSettings
import loremaster.projectsettings.readRawOutputs

/** A storage of the project as the Sync view lists it: its place in the settings, what the engine calls it, and where it goes. */
data class StorageLocation(val index: Int, val output: OutputSettings, val label: String, val detail: String, val icon: String)

/** Everything the view shows: the storages that are set up, and the engine's description they were read with. */
data class Locations(val storages: List<StorageLocation>, val description: ConfigDescription)

/**
 * A storage shows in the view once it is set up: every setting its platform's description requires is filled (the blank scaffold of
 * a project with no configuration is hidden). One the description does not know is shown, so a project written for a newer
 * LoreMaster is not hidden from its user. [value] reads a setting of the output by key.
 */
fun isSetUp(description: ConfigDescription, platformId: String, value: (String) -> String): Boolean {
    val platform = description.platformOf(platformId) ?: return true

    return platform.fields.none { it.required && value(it.key).isBlank() }
}

/** The project's storages that are set up, named by the engine, and the description they were judged by. */
fun readLocations(engine: EngineRpcClient, workspaceRoot: String): Locations {
    val description = describeConfig(engine)
    val outputs = readProjectSettings(engine, workspaceRoot).settings.outputs
    val raw = readRawOutputs(engine, workspaceRoot)

    val storages = outputs.mapIndexedNotNull { index, output ->
        val held = raw.getOrNull(index)
        if (!isSetUp(description, output.platform) { key -> held?.get(key)?.takeIf { it.isJsonPrimitive }?.asString.orEmpty() }) return@mapIndexedNotNull null

        StorageLocation(index, output, output.label(), output.summary, description.platformOf(output.platform)?.icon.orEmpty())
    }

    return Locations(storages, description)
}

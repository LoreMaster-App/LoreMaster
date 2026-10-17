package loremaster.projectsettings

import loremaster.engineprocess.EngineRpcClient
import java.util.concurrent.TimeUnit

/** Reads the project's outputs and generators through the engine, which layers the user, workspace and folder settings. */
fun readProjectSettings(engine: EngineRpcClient, workspaceRoot: String): SettingsReadResult {
    val read = engine.request(METHOD_SETTINGS_READ, SettingsReadParams(workspaceRoot), SettingsReadResult::class.java)
        .get(30, TimeUnit.SECONDS)
    val named = read.settings.outputs.mapIndexed { index, output ->
        output.copy(displayName = read.displayNames.getOrElse(index) { "" }, summary = read.summaries.getOrElse(index) { "" })
    }

    return read.copy(settings = read.settings.copy(outputs = named))
}

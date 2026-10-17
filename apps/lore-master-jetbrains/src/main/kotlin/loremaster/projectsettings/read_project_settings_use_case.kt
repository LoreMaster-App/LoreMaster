package loremaster.projectsettings

import loremaster.engineprocess.EngineRpcClient
import java.util.concurrent.TimeUnit

/** Reads the project's outputs and generators through the engine, which layers the user, workspace and folder settings. */
fun readProjectSettings(engine: EngineRpcClient, workspaceRoot: String): SettingsReadResult =
    engine.request(METHOD_SETTINGS_READ, SettingsReadParams(workspaceRoot), SettingsReadResult::class.java)
        .get(30, TimeUnit.SECONDS)

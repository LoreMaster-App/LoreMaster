package loremaster.configdescription

import loremaster.engineprocess.EngineRpcClient
import java.util.Collections
import java.util.WeakHashMap
import java.util.concurrent.TimeUnit

private val known: MutableMap<EngineRpcClient, ConfigDescription> = Collections.synchronizedMap(WeakHashMap())

/**
 * Everything the engine says can be configured. Asked once per engine connection and remembered; a failed ask is not, so the next
 * one tries again. This is where the plugin learns the platforms, their fields and the settings: it keeps no list of its own.
 */
fun describeConfig(engine: EngineRpcClient): ConfigDescription =
    known[engine] ?: engine.request(METHOD_CONFIG_DESCRIBE, emptyMap<String, Any>(), ConfigDescription::class.java)
        .get(30, TimeUnit.SECONDS)
        .also { known[engine] = it }

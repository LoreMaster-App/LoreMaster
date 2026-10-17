package loremaster.engineprocess

import com.google.gson.JsonElement
import com.intellij.ide.plugins.PluginManagerCore
import com.intellij.openapi.Disposable
import com.intellij.openapi.components.Service
import com.intellij.openapi.extensions.PluginId
import loremaster.pluginsettings.PluginSettingsStore
import java.nio.file.Files
import java.nio.file.Path
import java.util.concurrent.CopyOnWriteArrayList

/** The id in plugin.xml. */
const val PLUGIN_ID = "app.loremaster.jetbrains"

/** The environment variable that points at a locally built engine, for development. */
const val ENGINE_PATH_VARIABLE = "LORE_MASTER_ENGINE"

/**
 * Owns the one engine process of the IDE: started on first use, restarted if it has died, killed when the IDE
 * (or the plugin) goes away.
 */
@Service(Service.Level.APP)
class EngineProcessStore : Disposable {
    private var process: Process? = null
    private var client: EngineRpcClient? = null
    private val listeners = CopyOnWriteArrayList<(String, JsonElement?) -> Unit>()

    /** Hears the engine's notifications (`host/progress`, ...) until the returned function is called. */
    fun listen(listener: (method: String, params: JsonElement?) -> Unit): () -> Unit {
        listeners.add(listener)

        return { listeners.remove(listener) }
    }

    /** The connection to a running engine, starting one if there is none. */
    @Synchronized
    fun connection(): EngineRpcClient {
        val running = client
        if (running != null && process?.isAlive == true) {
            return running
        }
        stop()

        val binary = locateBinary()
        check(Files.isRegularFile(binary)) { "the engine was not found at $binary" }
        // A zip written on Windows carries no execute bit.
        binary.toFile().setExecutable(true)
        val started = ProcessBuilder(binary.toString()).redirectError(ProcessBuilder.Redirect.DISCARD).start()
        process = started

        val notify: (String, JsonElement?) -> Unit = { method, params -> listeners.forEach { it(method, params) } }

        return EngineRpcClient(started.inputStream, started.outputStream, onNotification = notify).also { client = it }
    }

    override fun dispose() = stop()

    @Synchronized
    private fun stop() {
        runCatching { client?.close() }
        process?.destroy()
        client = null
        process = null
    }

    private fun locateBinary(): Path {
        val pluginPath = PluginManagerCore.getPlugin(PluginId.getId(PLUGIN_ID))?.pluginPath
            ?: error("the LoreMaster plugin is not installed")

        return resolveEngineBinary(enginePlatform(System.getProperty("os.name"), System.getProperty("os.arch")), pluginPath, PluginSettingsStore.current().enginePath.ifBlank { System.getenv(ENGINE_PATH_VARIABLE) })
    }
}

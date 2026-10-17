package loremaster.engineprocess

import com.intellij.ide.plugins.PluginManagerCore
import com.intellij.openapi.Disposable
import com.intellij.openapi.components.Service
import com.intellij.openapi.extensions.PluginId
import com.intellij.openapi.util.SystemInfo
import java.nio.file.Files
import java.nio.file.Path

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
        val started = ProcessBuilder(binary.toString()).redirectError(ProcessBuilder.Redirect.DISCARD).start()
        process = started

        return EngineRpcClient(started.inputStream, started.outputStream).also { client = it }
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

        return resolveEngineBinary(SystemInfo.isWindows, pluginPath, System.getenv(ENGINE_PATH_VARIABLE))
    }
}

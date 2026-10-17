package loremaster.watch

import com.intellij.notification.NotificationType
import com.intellij.openapi.Disposable
import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.components.Service
import com.intellij.openapi.diagnostic.Logger
import com.intellij.openapi.project.Project
import com.intellij.openapi.vfs.VirtualFileManager
import com.intellij.openapi.vfs.newvfs.BulkFileListener
import com.intellij.openapi.vfs.newvfs.events.VFileEvent
import loremaster.engineprocess.EngineProcessStore
import loremaster.engineprogress.TaskReport
import loremaster.engineprogress.notify
import loremaster.pluginsettings.PluginSettingsStore
import loremaster.projectsettings.readProjectSettings
import java.nio.file.Files
import java.nio.file.Path
import java.security.MessageDigest
import java.util.concurrent.ConcurrentHashMap

private val LOG = Logger.getInstance("loremaster.watch")

/**
 * Watch mode of one project: while on, a change to a Markdown file, the settings, or a file a generator reads is synced on its
 * own once nothing has changed for a moment. Only the generators that read what changed run, only the pages that changed sync.
 */
@Service(Service.Level.PROJECT)
class WatchSessionStore(private val project: Project) : Disposable {
    private var batcher: ChangeBatcher? = null
    private var connection: com.intellij.util.messages.MessageBusConnection? = null

    /** What the files our own batches wrote contained afterwards, to tell that from the user's edits. */
    private val fingerprints = ConcurrentHashMap<String, String>()

    @get:Synchronized
    val active: Boolean get() = batcher != null

    /** Starts watching. Returns why it could not, or null when it is on. */
    @Synchronized
    fun start(): String? {
        if (batcher != null) return null
        val root = project.basePath ?: return "Open a project to watch it."
        val engineStore = ApplicationManager.getApplication().getService(EngineProcessStore::class.java)
        if (readProjectSettings(engineStore.connection(), root).settings.outputs.isEmpty()) {
            return "Set up an output first (LoreMaster: Open Project Settings); watching needs somewhere to send the pages."
        }
        val quietMs = PluginSettingsStore.current().watchDebounceSeconds.coerceIn(1, 300) * 1000L
        val folder = Path.of(root)
        val created = ChangeBatcher(
            quietMs = quietMs,
            maxBackoffMs = maxOf(60_000L, quietMs),
            run = { changed -> runBatch(engineStore, root, folder, changed) },
            onFailure = { error, retryInMs ->
                LOG.info("watch: failed: ${error.message}; trying again in ${retryInMs / 1000}s")
                notify(project, TaskReport("Watch: sync failed, retrying in ${retryInMs / 1000}s: ${error.message}", NotificationType.WARNING))
            },
        )
        batcher = created
        connection = project.messageBus.connect(this).also {
            it.subscribe(VirtualFileManager.VFS_CHANGES, object : BulkFileListener {
                override fun after(events: List<VFileEvent>) = events.forEach { event -> noticeChange(folder, created, event.path) }
            })
        }

        return null
    }

    @Synchronized
    fun stop() {
        connection?.disconnect()
        connection = null
        batcher?.close()
        batcher = null
        fingerprints.clear()
    }

    override fun dispose() = stop()

    private fun runBatch(engineStore: EngineProcessStore, root: String, folder: Path, changed: List<String>) {
        val applied = applyChanges(engineStore.connection(), root, changed) { scope ->
            syncConfluenceOutputs(engineStore.connection(), root, scope)
        }
        applied.log.forEach { LOG.info("watch: $it") }
        applied.touched.forEach { path -> fingerprint(folder.resolve(path))?.let { fingerprints[path] = it } }
    }

    private fun noticeChange(folder: Path, target: ChangeBatcher, absolute: String) {
        val path = workspacePath(folder, Path.of(absolute)) ?: return
        if (!isWatchedPath(path)) return
        val known = fingerprints[path]
        if (known != null) {
            if (fingerprint(folder.resolve(path)) == known) return
            fingerprints.remove(path)
        }
        target.add(listOf(path))
    }

    private fun fingerprint(file: Path): String? = runCatching {
        MessageDigest.getInstance("SHA-256").digest(Files.readAllBytes(file)).joinToString("") { "%02x".format(it) }
    }.getOrNull()
}

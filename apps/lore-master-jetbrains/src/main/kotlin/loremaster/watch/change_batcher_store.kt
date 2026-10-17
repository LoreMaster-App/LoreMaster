package loremaster.watch

import java.util.concurrent.Executors
import java.util.concurrent.ScheduledExecutorService
import java.util.concurrent.ScheduledFuture
import java.util.concurrent.TimeUnit

/**
 * Gathers changed paths and hands them over as one batch once nothing has changed for [quietMs], so a burst of saves is one
 * sync. One batch runs at a time; paths that change meanwhile wait for the next. A batch that fails is put back and tried
 * again after a pause that doubles with each failure, up to [maxBackoffMs], and starts over at the first success.
 */
class ChangeBatcher(
    private val quietMs: Long,
    private val maxBackoffMs: Long,
    private val run: (changed: List<String>) -> Unit,
    private val onFailure: (error: Exception, retryInMs: Long) -> Unit,
    private val scheduler: ScheduledExecutorService = Executors.newSingleThreadScheduledExecutor { task -> Thread(task, "lore-master-watch").apply { isDaemon = true } },
) : AutoCloseable {
    private val pending = linkedSetOf<String>()
    private var timer: ScheduledFuture<*>? = null
    private var running = false
    private var failures = 0
    private var closed = false

    /** Queues changed paths and restarts the quiet period. */
    @Synchronized
    fun add(paths: Collection<String>) {
        if (closed || paths.isEmpty()) return
        pending.addAll(paths)
        if (!running) schedule(quietMs)
    }

    @Synchronized
    override fun close() {
        closed = true
        timer?.cancel(false)
        pending.clear()
        scheduler.shutdownNow()
    }

    @Synchronized
    private fun schedule(delayMs: Long) {
        timer?.cancel(false)
        timer = scheduler.schedule({ fire() }, delayMs, TimeUnit.MILLISECONDS)
    }

    private fun fire() {
        val batch: List<String>
        synchronized(this) {
            if (closed || running || pending.isEmpty()) return
            running = true
            batch = pending.sorted()
            pending.clear()
        }
        try {
            run(batch)
            synchronized(this) {
                failures = 0
                running = false
                if (pending.isNotEmpty() && !closed) schedule(quietMs)
            }
        } catch (error: Exception) {
            synchronized(this) {
                failures += 1
                pending.addAll(batch)
                running = false
                val wait = minOf(maxBackoffMs, quietMs shl minOf(failures, 20))
                if (!closed) schedule(wait)
                runCatching { onFailure(error, wait) }
            }
        }
    }
}

package loremaster.watch

import java.util.concurrent.CountDownLatch
import java.util.concurrent.LinkedBlockingQueue
import java.util.concurrent.TimeUnit
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.test.assertTrue

class ChangeBatcherStoreTest {
    private fun batcher(batches: LinkedBlockingQueue<List<String>>, failFirst: Int = 0, failures: LinkedBlockingQueue<Long> = LinkedBlockingQueue()): ChangeBatcher {
        var attempts = 0

        return ChangeBatcher(
            quietMs = 40,
            maxBackoffMs = 400,
            run = { changed ->
                attempts += 1
                if (attempts <= failFirst) error("down")
                batches.add(changed)
            },
            onFailure = { _, retryInMs -> failures.add(retryInMs) },
        )
    }

    @Test
    fun `a burst of changes is one sorted batch without duplicates`() {
        val batches = LinkedBlockingQueue<List<String>>()
        batcher(batches).use { watcher ->
            watcher.add(listOf("b.md"))
            watcher.add(listOf("a.md", "b.md"))
            Thread.sleep(10)
            watcher.add(listOf("c.md"))

            assertEquals(listOf("a.md", "b.md", "c.md"), batches.poll(3, TimeUnit.SECONDS))
            assertNull(batches.poll(200, TimeUnit.MILLISECONDS))
        }
    }

    @Test
    fun `a failed batch is kept and tried again after a longer pause`() {
        val batches = LinkedBlockingQueue<List<String>>()
        val failures = LinkedBlockingQueue<Long>()
        batcher(batches, failFirst = 2, failures = failures).use { watcher ->
            watcher.add(listOf("a.md"))

            assertEquals(listOf("a.md"), batches.poll(5, TimeUnit.SECONDS))
            val first = failures.poll()
            val second = failures.poll()
            assertTrue(second > first, "$first then $second")
        }
    }

    @Test
    fun `changes that arrive while a batch runs wait for the next one`() {
        val started = CountDownLatch(1)
        val release = CountDownLatch(1)
        val batches = LinkedBlockingQueue<List<String>>()
        ChangeBatcher(40, 400, { changed ->
            batches.add(changed)
            started.countDown()
            release.await(5, TimeUnit.SECONDS)
        }, { _, _ -> }).use { watcher ->
            watcher.add(listOf("a.md"))
            assertTrue(started.await(3, TimeUnit.SECONDS))
            watcher.add(listOf("b.md"))
            release.countDown()

            assertEquals(listOf("a.md"), batches.poll(3, TimeUnit.SECONDS))
            assertEquals(listOf("b.md"), batches.poll(3, TimeUnit.SECONDS))
        }
    }

    @Test
    fun `nothing runs after close`() {
        val batches = LinkedBlockingQueue<List<String>>()
        val watcher = batcher(batches)
        watcher.add(listOf("a.md"))
        watcher.close()

        assertNull(batches.poll(300, TimeUnit.MILLISECONDS))
    }
}

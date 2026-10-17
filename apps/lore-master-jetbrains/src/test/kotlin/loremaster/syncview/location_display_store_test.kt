package loremaster.syncview

import kotlin.test.Test
import kotlin.test.assertEquals

class LocationDisplayStoreTest {
    @Test
    fun `a display is stored as the words that differ from the default`() {
        assertEquals("", encodeDisplay(LocationDisplay()))
        assertEquals("fileNames,flat,hideIgnored", encodeDisplay(LocationDisplay(fileNames = true, flat = true, hideIgnored = true)))
        assertEquals(LocationDisplay(flat = true), decodeDisplay("flat"))
        assertEquals(LocationDisplay(), decodeDisplay("nonsense, "))
    }

    @Test
    fun `each location remembers its own choices and the others stay on the defaults`() {
        val store = LocationDisplayStore()

        store.change(1) { it.copy(flat = true) }
        store.change(1) { it.copy(fileNames = true) }
        store.change(LOCAL_INDEX) { it.copy(hideIgnored = true) }

        assertEquals(LocationDisplay(fileNames = true, flat = true), store.displayOf(1))
        assertEquals(LocationDisplay(hideIgnored = true), store.displayOf(LOCAL_INDEX))
        assertEquals(LocationDisplay(), store.displayOf(0))
    }

    @Test
    fun `going back to the defaults forgets the entry`() {
        val store = LocationDisplayStore()
        store.change(2) { it.copy(flat = true) }

        store.change(2) { it.copy(flat = false) }

        assertEquals(emptyMap(), store.state.choices)
    }
}

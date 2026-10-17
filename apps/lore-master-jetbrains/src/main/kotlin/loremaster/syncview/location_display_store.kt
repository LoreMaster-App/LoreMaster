package loremaster.syncview

import com.intellij.openapi.components.PersistentStateComponent
import com.intellij.openapi.components.Service
import com.intellij.openapi.components.State
import com.intellij.openapi.components.Storage
import com.intellij.openapi.components.StoragePathMacros

/**
 * How one location of the Sync view (Local, or a storage) is shown: pages named by file name instead of title, a flat list instead
 * of a tree, and the files it leaves out hidden instead of listed.
 */
data class LocationDisplay(val fileNames: Boolean = false, val flat: Boolean = false, val hideIgnored: Boolean = false)

private const val FILE_NAMES = "fileNames"
private const val FLAT = "flat"
private const val HIDE_IGNORED = "hideIgnored"

/** A display as the words that differ from the default, comma separated; the form it is stored in. */
fun encodeDisplay(display: LocationDisplay): String = listOfNotNull(
    FILE_NAMES.takeIf { display.fileNames },
    FLAT.takeIf { display.flat },
    HIDE_IGNORED.takeIf { display.hideIgnored },
).joinToString(",")

fun decodeDisplay(text: String): LocationDisplay {
    val words = text.split(',').map { it.trim() }.toSet()

    return LocationDisplay(fileNames = FILE_NAMES in words, flat = FLAT in words, hideIgnored = HIDE_IGNORED in words)
}

/** What the store keeps: for each location (its place in the settings, -1 for Local) its display. */
class LocationDisplayState {
    var choices: MutableMap<String, String> = mutableMapOf()
}

/** Each location of the Sync view remembers its own display choices, for this project. */
@Service(Service.Level.PROJECT)
@State(name = "LoreMasterSyncView", storages = [Storage(StoragePathMacros.WORKSPACE_FILE)])
class LocationDisplayStore : PersistentStateComponent<LocationDisplayState> {
    private var state = LocationDisplayState()

    override fun getState(): LocationDisplayState = state

    override fun loadState(loaded: LocationDisplayState) {
        state = loaded
    }

    /** How a location is shown now. */
    fun displayOf(index: Int): LocationDisplay = decodeDisplay(state.choices[index.toString()].orEmpty())

    /** Changes one location's display. */
    fun change(index: Int, change: (LocationDisplay) -> LocationDisplay) {
        val next = encodeDisplay(change(displayOf(index)))
        if (next.isEmpty()) state.choices.remove(index.toString()) else state.choices[index.toString()] = next
    }
}

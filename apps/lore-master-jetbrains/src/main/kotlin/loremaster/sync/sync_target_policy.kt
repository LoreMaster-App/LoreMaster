package loremaster.sync

import loremaster.projectsettings.OutputSettings

/** Which outputs a sync is for. */
sealed interface SyncTarget {
    /** Every output that can sync; the Sync All action. */
    data object All : SyncTarget

    /** Ask which one when there are several; the Tools menu's Sync. */
    data object Choose : SyncTarget

    /** These outputs, by their place in the settings; the Sync action of a storage's row. */
    data class Only(val indices: List<Int>) : SyncTarget
}

/** What to do about a [SyncTarget] once the outputs that can sync are known. */
sealed interface SyncSelection {
    /** Sync these, in order. */
    data class Run(val outputs: List<IndexedValue<OutputSettings>>) : SyncSelection

    /** More than one can sync and the user is to pick one. */
    data class AskWhich(val outputs: List<IndexedValue<OutputSettings>>) : SyncSelection

    /** Nothing to sync, and why. */
    data class Nothing(val reason: String) : SyncSelection
}

/** The outputs a sync of [target] runs for, out of the [syncable] ones. */
fun selectOutputs(target: SyncTarget, syncable: List<IndexedValue<OutputSettings>>): SyncSelection = when {
    syncable.isEmpty() ->
        SyncSelection.Nothing("No output is ready to sync. Add one (Add Sync Storage) or edit loreMaster.outputs (LoreMaster Config).")
    target is SyncTarget.All -> SyncSelection.Run(syncable)
    target is SyncTarget.Only -> syncable.filter { it.index in target.indices }.let {
        if (it.isEmpty()) SyncSelection.Nothing("That storage is not ready to sync yet: fill in what it is missing (Edit Storage).") else SyncSelection.Run(it)
    }
    syncable.size == 1 -> SyncSelection.Run(syncable)
    else -> SyncSelection.AskWhich(syncable)
}

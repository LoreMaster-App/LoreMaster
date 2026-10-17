package loremaster.sync

import loremaster.projectsettings.OutputSettings
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs

class SyncTargetPolicyTest {
    private fun outputs(vararg names: String) = names.mapIndexed { index, name -> IndexedValue(index, OutputSettings(name = name, displayName = name)) }

    @Test
    fun `sync all runs every output that can sync, in order`() {
        val selection = selectOutputs(SyncTarget.All, outputs("a", "b", "c"))

        assertEquals(listOf(0, 1, 2), (selection as SyncSelection.Run).outputs.map { it.index })
    }

    @Test
    fun `a storage's own sync runs only that storage`() {
        val selection = selectOutputs(SyncTarget.Only(listOf(1)), outputs("a", "b", "c"))

        assertEquals(listOf("b"), (selection as SyncSelection.Run).outputs.map { it.value.name })
    }

    @Test
    fun `a storage that cannot sync yet says so instead of syncing another`() {
        val selection = selectOutputs(SyncTarget.Only(listOf(5)), outputs("a", "b"))

        assertIs<SyncSelection.Nothing>(selection)
        assertEquals(true, selection.reason.contains("not ready"))
    }

    @Test
    fun `choosing asks which only when there is more than one`() {
        assertIs<SyncSelection.Run>(selectOutputs(SyncTarget.Choose, outputs("a")))
        assertEquals(2, (selectOutputs(SyncTarget.Choose, outputs("a", "b")) as SyncSelection.AskWhich).outputs.size)
    }

    @Test
    fun `with nothing set up there is nothing to sync, whatever the target`() {
        for (target in listOf(SyncTarget.All, SyncTarget.Choose, SyncTarget.Only(listOf(0)))) {
            assertIs<SyncSelection.Nothing>(selectOutputs(target, emptyList()))
        }
    }
}

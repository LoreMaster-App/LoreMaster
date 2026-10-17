package loremaster.engineprocess

import java.nio.file.Path
import kotlin.test.Test
import kotlin.test.assertEquals

class EngineBinaryPolicyTest {
    private val plugin = Path.of("plugins", "lore-master")

    @Test
    fun `the bundled binary is in bin, with an exe suffix on windows`() {
        assertEquals(plugin.resolve("bin/lore-master-engine"), resolveEngineBinary(false, plugin))
        assertEquals(plugin.resolve("bin/lore-master-engine.exe"), resolveEngineBinary(true, plugin))
    }

    @Test
    fun `a configured path wins, a blank one does not`() {
        assertEquals(Path.of("/dev/engine"), resolveEngineBinary(false, plugin, " /dev/engine "))
        assertEquals(plugin.resolve("bin/lore-master-engine"), resolveEngineBinary(false, plugin, "   "))
    }
}

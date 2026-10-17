package loremaster.engineprocess

import java.nio.file.Path
import kotlin.test.Test
import kotlin.test.assertEquals

class EngineBinaryPolicyTest {
    private val plugin = Path.of("plugins", "lore-master")

    @Test
    fun `the platform folder follows the JVM's os and arch`() {
        assertEquals("windows-amd64", enginePlatform("Windows 11", "amd64"))
        assertEquals("windows-arm64", enginePlatform("Windows 11", "aarch64"))
        assertEquals("darwin-arm64", enginePlatform("Mac OS X", "aarch64"))
        assertEquals("darwin-amd64", enginePlatform("Mac OS X", "x86_64"))
        assertEquals("linux-amd64", enginePlatform("Linux", "amd64"))
        assertEquals("linux-arm64", enginePlatform("Linux", "aarch64"))
    }

    @Test
    fun `the bundled binary is in bin per platform, with an exe suffix on windows`() {
        assertEquals(plugin.resolve("bin/linux-amd64/lore-master-engine"), resolveEngineBinary("linux-amd64", plugin))
        assertEquals(plugin.resolve("bin/windows-arm64/lore-master-engine.exe"), resolveEngineBinary("windows-arm64", plugin))
    }

    @Test
    fun `a configured path wins, a blank one does not`() {
        assertEquals(Path.of("/dev/engine"), resolveEngineBinary("linux-amd64", plugin, " /dev/engine "))
        assertEquals(plugin.resolve("bin/linux-amd64/lore-master-engine"), resolveEngineBinary("linux-amd64", plugin, "   "))
    }
}

package loremaster.engineprocess

import java.io.ByteArrayInputStream
import java.io.ByteArrayOutputStream
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

class JsonRpcFramingAlgorithmTest {
    @Test
    fun `a written frame is read back, counting bytes not characters`() {
        val out = ByteArrayOutputStream()
        writeFrame(out, """{"text":"áé"}""")
        writeFrame(out, "{}")

        val input = ByteArrayInputStream(out.toByteArray())
        assertEquals("""{"text":"áé"}""", readFrame(input))
        assertEquals("{}", readFrame(input))
        assertNull(readFrame(input))
    }

    @Test
    fun `other headers are skipped and the name is not case sensitive`() {
        val raw = "Content-Type: x\r\ncontent-length: 2\r\n\r\n{}"

        assertEquals("{}", readFrame(ByteArrayInputStream(raw.toByteArray())))
    }

    @Test
    fun `a body cut short ends the stream`() {
        assertNull(readFrame(ByteArrayInputStream("Content-Length: 10\r\n\r\n{}".toByteArray())))
    }
}

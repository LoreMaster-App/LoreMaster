package loremaster.engineprocess

import java.io.InputStream
import java.io.OutputStream

private const val CONTENT_LENGTH = "content-length:"

/** Writes one message with the LSP-style header the engine speaks: `Content-Length: <bytes>\r\n\r\n<body>`. */
fun writeFrame(output: OutputStream, body: String) {
    val bytes = body.toByteArray(Charsets.UTF_8)
    output.write("Content-Length: ${bytes.size}\r\n\r\n".toByteArray(Charsets.US_ASCII))
    output.write(bytes)
    output.flush()
}

/** Reads one message body, or null when the stream ends before a complete message. */
fun readFrame(input: InputStream): String? {
    var length = -1
    while (true) {
        val line = readHeaderLine(input) ?: return null
        if (line.isEmpty()) {
            break
        }
        if (line.lowercase().startsWith(CONTENT_LENGTH)) {
            length = line.substring(CONTENT_LENGTH.length).trim().toInt()
        }
    }
    require(length >= 0) { "a message came without Content-Length" }

    val body = input.readNBytes(length)
    if (body.size < length) {
        return null
    }

    return String(body, Charsets.UTF_8)
}

private fun readHeaderLine(input: InputStream): String? {
    val line = StringBuilder()
    while (true) {
        val next = input.read()
        if (next < 0) {
            return null
        }
        if (next == '\n'.code) {
            return line.toString().trimEnd('\r')
        }
        line.append(next.toChar())
    }
}

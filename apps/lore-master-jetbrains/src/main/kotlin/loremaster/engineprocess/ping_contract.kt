package loremaster.engineprocess

/** The engine's `ping` method: no params, answers [PingResult]. */
const val METHOD_PING = "ping"

/** What `ping` answers. [pong] is always "pong"; [version] is the engine's build version. */
data class PingResult(val pong: String = "", val version: String = "")

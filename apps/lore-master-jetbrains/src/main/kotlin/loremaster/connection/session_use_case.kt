package loremaster.connection

import loremaster.engineprocess.EngineRpcClient
import java.util.concurrent.TimeUnit

private const val SESSION_TIMEOUT_SECONDS = 60L

/** Connects the engine to a site; the credential stays in the engine's memory until [closeSession]. */
fun openSession(engine: EngineRpcClient, baseUrl: String, credential: Credential): SessionOpenResult =
    engine.request(METHOD_SESSION_OPEN, SessionOpenParams(baseUrl, credential), SessionOpenResult::class.java)
        .get(SESSION_TIMEOUT_SECONDS, TimeUnit.SECONDS)

/** Forgets a session. Closing one the engine does not know is not an error. */
fun closeSession(engine: EngineRpcClient, sessionId: String) {
    engine.request(METHOD_SESSION_CLOSE, SessionCloseParams(sessionId), Any::class.java)
        .get(SESSION_TIMEOUT_SECONDS, TimeUnit.SECONDS)
}

package loremaster.connection

/** The engine's `session/open`: connects to a documentation platform and verifies the credential. */
const val METHOD_SESSION_OPEN = "session/open"

/** The engine's `session/close`: forgets a session and its credential. */
const val METHOD_SESSION_CLOSE = "session/close"

/** How the user signs in: "apitoken" (Cloud: email + token), "pat" (Data Center, Server: token), "basic" (user + password). */
object CredentialKind {
    const val API_TOKEN = "apitoken"
    const val PAT = "pat"
    const val BASIC = "basic"
}

/** A credential as the engine takes it. Which fields count depends on [kind]. */
data class Credential(
    val kind: String = CredentialKind.API_TOKEN,
    val email: String? = null,
    val token: String? = null,
    val user: String? = null,
    val password: String? = null,
)

data class SessionOpenParams(val baseUrl: String, val credential: Credential, val edition: String? = null)

data class SessionUser(val displayName: String = "", val accountId: String? = null, val username: String? = null)

/** An open session: [baseUrl] is the normalised address, [edition] cloud, datacenter or server. */
data class SessionOpenResult(
    val sessionId: String = "",
    val baseUrl: String = "",
    val edition: String = "",
    val version: String? = null,
    val user: SessionUser = SessionUser(),
)

data class SessionCloseParams(val sessionId: String)

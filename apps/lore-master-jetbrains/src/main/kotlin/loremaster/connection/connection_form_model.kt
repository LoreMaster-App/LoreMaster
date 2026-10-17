package loremaster.connection

/** What the connection dialog edits. */
class ConnectionForm(
    var baseUrl: String = "",
    var kind: String = CredentialKind.API_TOKEN,
    var email: String = "",
    var token: String = "",
    var user: String = "",
    var password: String = "",
) {
    fun credential() = when (kind) {
        CredentialKind.API_TOKEN -> Credential(kind, email = email.trim(), token = token.trim())
        CredentialKind.PAT -> Credential(kind, token = token.trim())
        else -> Credential(kind, user = user.trim(), password = password)
    }

    /** What is missing, or null when the form can be tried. */
    fun problem(): String? = when {
        baseUrl.isBlank() -> "Enter the Confluence address."
        kind == CredentialKind.API_TOKEN && (email.isBlank() || token.isBlank()) -> "Enter the email and the API token."
        kind == CredentialKind.PAT && token.isBlank() -> "Enter the personal access token."
        kind == CredentialKind.BASIC && (user.isBlank() || password.isEmpty()) -> "Enter the user name and the password."
        else -> null
    }
}

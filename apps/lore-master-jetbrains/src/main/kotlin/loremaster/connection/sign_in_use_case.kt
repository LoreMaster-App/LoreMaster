package loremaster.connection

import com.intellij.openapi.project.Project
import loremaster.engineprogress.onUiThread

/**
 * The credential to sign in to [baseUrl] with: the one in the password safe, else one the user types in a dialog (and which is
 * then saved). Null when the user declines. Safe to call from a background thread.
 */
fun signInCredential(project: Project?, baseUrl: String): Credential? {
    CredentialVaultClient.load(baseUrl)?.let { return it }

    return onUiThread {
        val form = ConnectionForm(baseUrl = baseUrl)
        if (ConnectionDialogComponent(project, form).showAndGet()) form.credential().also { CredentialVaultClient.save(baseUrl, it) } else null
    }
}

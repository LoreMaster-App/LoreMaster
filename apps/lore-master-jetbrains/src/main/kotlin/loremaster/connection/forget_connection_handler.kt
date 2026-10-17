package loremaster.connection

import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.ui.Messages

/** "LoreMaster: Forget Confluence Connection": removes a site's credential from the password safe. */
class ForgetConnectionHandler : AnAction() {
    override fun actionPerformed(event: AnActionEvent) {
        val address = Messages.showInputDialog(event.project, "The Confluence address to forget:", "Forget Connection", null) ?: return
        if (address.isNotBlank()) {
            CredentialVaultClient.forget(address)
        }
    }
}

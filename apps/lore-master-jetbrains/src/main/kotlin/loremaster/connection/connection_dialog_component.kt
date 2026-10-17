package loremaster.connection

import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.DialogPanel
import com.intellij.openapi.ui.DialogWrapper
import com.intellij.ui.SimpleListCellRenderer
import com.intellij.ui.dsl.builder.AlignX
import com.intellij.ui.dsl.builder.bindItem
import com.intellij.ui.dsl.builder.bindText
import com.intellij.ui.dsl.builder.panel
import javax.swing.JComponent

private val KINDS = listOf(
    CredentialKind.API_TOKEN to "Confluence Cloud: email and API token",
    CredentialKind.PAT to "Data Center or Server: personal access token",
    CredentialKind.BASIC to "Server: user name and password",
)

/** Asks for a site and how to sign in. */
class ConnectionDialogComponent(project: Project?, private val form: ConnectionForm) : DialogWrapper(project) {
    private lateinit var content: DialogPanel

    init {
        title = "Connect LoreMaster to Confluence"
        setOKButtonText("Connect")
        init()
    }

    override fun createCenterPanel(): JComponent {
        content = panel {
            row("Address:") {
                textField().bindText(form::baseUrl).align(AlignX.FILL)
                    .comment("For example https://acme.atlassian.net/wiki. Any page address of the site works.")
            }
            row("Sign in with:") {
                comboBox(KINDS.map { it.first }, SimpleListCellRenderer.create("") { id -> KINDS.first { it.first == id }.second })
                    .bindItem({ form.kind }, { form.kind = it ?: CredentialKind.API_TOKEN })
            }
            row("Email:") { textField().bindText(form::email).align(AlignX.FILL) }
            row("User name:") { textField().bindText(form::user).align(AlignX.FILL) }
            row("Password:") { passwordField().bindText(form::password).align(AlignX.FILL) }
            row("Token:") {
                passwordField().bindText(form::token).align(AlignX.FILL)
                    .comment("The API token (Cloud) or personal access token. It is kept in the IDE's password safe.")
            }
        }

        return content
    }

    override fun doOKAction() {
        content.apply()
        val problem = form.problem()
        if (problem != null) {
            setErrorText(problem)

            return
        }
        super.doOKAction()
    }
}

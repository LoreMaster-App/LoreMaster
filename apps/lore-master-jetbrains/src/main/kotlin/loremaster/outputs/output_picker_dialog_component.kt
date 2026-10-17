package loremaster.outputs

import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.DialogWrapper
import com.intellij.ui.components.JBLabel
import com.intellij.ui.components.JBTextField
import com.intellij.util.ui.FormBuilder
import loremaster.connection.closeSession
import loremaster.connection.openSession
import loremaster.connection.signInCredential
import loremaster.engineprocess.EngineRpcClient
import javax.swing.JButton
import javax.swing.JComponent
import javax.swing.SwingUtilities
import javax.swing.JComboBox

/**
 * Sets up where the project's pages go in Confluence: the address, a space from the ones the user can see, the parent page
 * (browsed or searched), and the title prefix (the parent's title unless changed). Signs in with the saved credential, asking
 * once when there is none.
 */
class OutputPickerDialogComponent(private val project: Project?, private val engine: EngineRpcClient) : DialogWrapper(project) {
    private val address = JBTextField()
    private val status = JBLabel(" ")
    private val spaces = JComboBox<Space>()
    private val parentLabel = JBLabel("(the space's home page)")
    private val prefix = JBTextField()
    private val choose = JButton("Choose parent page...")
    private var sessionId: String? = null
    private var connectedAddress = ""
    private var parent: RemotePage? = null

    init {
        title = "LoreMaster: Add a Confluence Output"
        setOKButtonText("Add output")
        init()
        spaces.isEnabled = false
        choose.isEnabled = false
    }

    /** What was picked; call after the dialog closed with OK. */
    fun choice(): ConfluenceOutputChoice {
        val space = spaces.selectedItem as Space
        val parentId = parent?.id ?: space.homepageId.orEmpty()

        return ConfluenceOutputChoice(connectedAddress, space.key, parentId, prefix.text.trim())
    }

    override fun createCenterPanel(): JComponent {
        val connect = JButton("Connect").apply { addActionListener { connect() } }
        choose.addActionListener { chooseParent() }
        spaces.addActionListener { spaceChanged() }

        return FormBuilder.createFormBuilder()
            .addLabeledComponent("Address:", address)
            .addComponent(connect)
            .addComponent(status)
            .addLabeledComponent("Space:", spaces)
            .addLabeledComponent("Parent page:", parentLabel)
            .addComponent(choose)
            .addLabeledComponent("Title prefix:", prefix)
            .panel
    }

    override fun doOKAction() {
        val space = spaces.selectedItem as? Space
        when {
            sessionId == null || space == null -> status.text = "Connect and choose a space first."
            parent == null && space.homepageId.isNullOrBlank() -> status.text = "This space has no home page: choose a parent page."
            prefix.text.isBlank() -> status.text = "Enter a title prefix (every page title starts with it)."
            else -> super.doOKAction()
        }
    }

    override fun dispose() {
        val id = sessionId
        sessionId = null
        if (id != null) {
            ApplicationManager.getApplication().executeOnPooledThread { runCatching { closeSession(engine, id) } }
        }
        super.dispose()
    }

    private fun connect() {
        val where = address.text.trim()
        if (where.isEmpty()) {
            status.text = "Enter the Confluence address."

            return
        }
        status.text = "Connecting..."
        ApplicationManager.getApplication().executeOnPooledThread {
            val outcome = runCatching {
                val credential = signInCredential(project, where) ?: return@runCatching null
                val session = openSession(engine, where, credential)
                session to listSpaces(engine, session.sessionId)
            }
            SwingUtilities.invokeLater {
                outcome.onSuccess { found ->
                    if (found == null) {
                        status.text = "Cancelled."

                        return@onSuccess
                    }
                    sessionId?.let { old -> ApplicationManager.getApplication().executeOnPooledThread { runCatching { closeSession(engine, old) } } }
                    sessionId = found.first.sessionId
                    connectedAddress = found.first.baseUrl.ifBlank { where }
                    status.text = "Connected to $connectedAddress as ${found.first.user.displayName}."
                    spaces.removeAllItems()
                    found.second.forEach(spaces::addItem)
                    spaces.isEnabled = found.second.isNotEmpty()
                    choose.isEnabled = found.second.isNotEmpty()
                }.onFailure { status.text = "Could not connect: ${it.cause?.message ?: it.message}" }
            }
        }
    }

    private fun spaceChanged() {
        val space = spaces.selectedItem as? Space ?: return
        parent = null
        parentLabel.text = if (space.homepageId.isNullOrBlank()) "(choose one)" else "(the space's home page)"
        if (prefix.text.isBlank() || prefix.text == space.name) prefix.text = space.key
    }

    private fun chooseParent() {
        val id = sessionId ?: return
        val space = spaces.selectedItem as? Space ?: return
        ApplicationManager.getApplication().executeOnPooledThread {
            val home = space.homepageId?.takeIf { it.isNotBlank() }?.let { runCatching { RemotePage(it, "${space.name} home") }.getOrNull() }
            SwingUtilities.invokeLater {
                val dialog = ParentPageDialogComponent(project, engine, id, space, home)
                if (dialog.showAndGet()) {
                    dialog.chosen?.let {
                        parent = it
                        parentLabel.text = it.title
                        prefix.text = it.title
                    }
                }
            }
        }
    }
}

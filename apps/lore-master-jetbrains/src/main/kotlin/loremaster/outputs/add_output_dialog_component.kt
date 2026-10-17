package loremaster.outputs

import com.google.gson.JsonObject
import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.DialogWrapper
import com.intellij.ui.SimpleListCellRenderer
import com.intellij.ui.components.JBCheckBox
import com.intellij.ui.components.JBLabel
import com.intellij.ui.components.JBTextField
import com.intellij.util.ui.FormBuilder
import com.intellij.util.ui.JBUI
import loremaster.configdescription.ChoiceDescriptor
import loremaster.configdescription.ConfigDescription
import loremaster.configdescription.FIELD_BOOL
import loremaster.configdescription.FIELD_CHOICE
import loremaster.configdescription.FieldDescriptor
import loremaster.configdescription.LOOKUP_PAGE
import loremaster.configdescription.LOOKUP_SPACE
import loremaster.configdescription.PlatformDescriptor
import loremaster.connection.closeSession
import loremaster.connection.openSession
import loremaster.connection.signInCredential
import loremaster.engineprocess.EngineRpcClient
import java.awt.BorderLayout
import javax.swing.DefaultComboBoxModel
import javax.swing.JButton
import javax.swing.JComboBox
import javax.swing.JComponent
import javax.swing.JPanel
import javax.swing.SwingUtilities

/**
 * Adds an output, as the engine describes it: the dialog offers the engine's platforms and draws the chosen one's fields, so a
 * platform the engine learns appears here without a change. A field with a space or page lookup is picked from the platform (after
 * signing in at the platform's connection field), and a lookup can fill another field (the parent page's title becomes the title
 * prefix); every other field is typed or chosen. Given [existing] it edits that output instead: its platform stays, its settings fill the
 * form, and whatever the form does not show is kept.
 */
class AddOutputDialogComponent(
    private val project: Project?,
    private val engine: EngineRpcClient,
    private val description: ConfigDescription,
    private val existing: JsonObject? = null,
) : DialogWrapper(project) {
    private val platforms = JComboBox(DefaultComboBoxModel(description.platforms.toTypedArray()))
    private val fieldsPanel = JPanel(BorderLayout())
    private val status = JBLabel(" ")
    private val readers = linkedMapOf<String, () -> String>()
    private val setters = mutableMapOf<String, (String) -> Unit>()
    private var remembered = mapOf<String, String>()
    private var sessionId: String? = null
    private var spaceBox: JComboBox<Space>? = null
    private var onConnected: (List<Space>) -> Unit = {}
    private var chosenPage: RemotePage? = null

    /** What a lookup field held before it could be picked from the platform (an output being edited, not yet signed in). */
    private val pending = mutableMapOf<String, String>()

    /** The platform whose fields are shown. */
    val platform: PlatformDescriptor get() = platforms.selectedItem as PlatformDescriptor

    /** The keys of the fields shown now. */
    val fieldKeys: List<String> get() = readers.keys.toList()

    init {
        title = if (existing == null) "LoreMaster: Add an Output" else "LoreMaster: Edit an Output"
        setOKButtonText(if (existing == null) "Add output" else "Save")
        platforms.renderer = SimpleListCellRenderer.create("") { it.label }
        existing?.get("platform")?.asString?.let { id -> description.platforms.firstOrNull { it.id == id }?.let { platforms.selectedItem = it } }
        platforms.isEnabled = existing == null
        platforms.addActionListener { drawFields() }
        init()
        drawFields()
        existing?.let { fillFrom(it) }
    }

    /** Shows what an output being edited holds: its settings by key, and the Markdown content's folders and excluded paths. */
    private fun fillFrom(output: JsonObject) {
        for (key in readers.keys.toList()) {
            val value = output.get(key) ?: continue
            setValue(key, if (value.isJsonArray) value.asJsonArray.joinToString(", ") { it.asString } else value.asString)
        }
        val markdown = output.getAsJsonArray("content")?.map { it.asJsonObject }?.firstOrNull { it.get("type")?.asString == "markdown" }
        for (key in listOf("roots", "excludes")) {
            markdown?.getAsJsonArray(key)?.let { setValue(key, it.joinToString(", ") { item -> item.asString }) }
        }
    }

    /** Sets what a field shows, as the user typing it would; for tests and for a lookup that fills it. */
    fun setValue(key: String, value: String) {
        setters[key]?.invoke(value)
    }

    /** Chooses which platform's fields are shown. */
    fun choosePlatform(id: String) {
        platforms.selectedItem = description.platforms.first { it.id == id }
    }

    /** The output the form describes; call after the dialog closed with OK. */
    fun output(): JsonObject = outputFrom(description, platform, values(), existing)

    private fun values(): Map<String, String> = readers.mapValues { it.value() }

    override fun createCenterPanel(): JComponent = FormBuilder.createFormBuilder()
        .addLabeledComponent("Add to:", platforms)
        .addComponentFillVertically(fieldsPanel, 0)
        .addComponent(status)
        .panel

    override fun doOKAction() {
        val errors = validateOutputValues(description, platform, values())
        if (errors.isEmpty()) super.doOKAction() else status.text = errors.values.first()
    }

    override fun dispose() {
        val id = sessionId
        sessionId = null
        if (id != null) ApplicationManager.getApplication().executeOnPooledThread { runCatching { closeSession(engine, id) } }
        super.dispose()
    }

    /** Draws the chosen platform's fields, keeping what was typed into any field the new platform shares. */
    private fun drawFields() {
        remembered = remembered + values()
        readers.clear()
        setters.clear()
        spaceBox = null
        onConnected = {}
        chosenPage = null
        pending.clear()
        status.text = " "

        val chosen = platform
        val builder = FormBuilder.createFormBuilder()
        val advanced = FormBuilder.createFormBuilder()
        for (field in formFields(description, chosen)) {
            (if (field.advanced) advanced else builder).addLabeledComponent(field.label + if (field.required) " *" else "", control(field, chosen))
            if (field.help.isNotBlank()) (if (field.advanced) advanced else builder).addTooltip(field.help)
        }
        for (field in description.contentFields) {
            advanced.addLabeledComponent(field.label, control(field, chosen))
        }
        val more = advanced.panel.apply { isVisible = false }
        val toggle = JBCheckBox("Show advanced settings").apply { addActionListener { more.isVisible = isSelected; refitWindow() } }
        builder.addComponent(toggle).addComponent(more)

        fieldsPanel.removeAll()
        fieldsPanel.add(builder.panel, BorderLayout.CENTER)
        fieldsPanel.revalidate()
        fieldsPanel.repaint()
        remembered.forEach { (key, value) -> if (value.isNotBlank()) setters[key]?.invoke(value) }
    }

    private fun refitWindow() {
        window?.let { it.pack(); it.revalidate() }
    }

    /** The control for one field, registering how to read and set it. */
    private fun control(field: FieldDescriptor, chosen: PlatformDescriptor): JComponent {
        val connection = chosen.connection
        return when {
            connection != null && connection.field == field.key -> addressControl(field)
            field.lookup?.kind == LOOKUP_SPACE -> spaceControl(field)
            field.lookup?.kind == LOOKUP_PAGE -> pageControl(field)
            field.type == FIELD_CHOICE -> choiceControl(field)
            field.type == FIELD_BOOL -> JBCheckBox().also {
                readers[field.key] = { it.isSelected.toString() }
                setters[field.key] = { value -> it.isSelected = value == "true" }
                it.isSelected = field.default == "true"
            }
            else -> JBTextField().also {
                it.emptyText.text = field.placeholder.ifBlank { field.default }
                readers[field.key] = { it.text }
                setters[field.key] = { value -> it.text = value }
            }
        }
    }

    private fun choiceControl(field: FieldDescriptor): JComponent {
        val box = JComboBox(DefaultComboBoxModel(field.choices.toTypedArray()))
        box.renderer = SimpleListCellRenderer.create("") { it.label }
        box.selectedItem = field.choices.firstOrNull { it.value == field.default } ?: field.choices.firstOrNull()
        readers[field.key] = { (box.selectedItem as? ChoiceDescriptor)?.value.orEmpty() }
        setters[field.key] = { value -> field.choices.firstOrNull { it.value == value }?.let { box.selectedItem = it } }

        return box
    }

    /** The site address with a Connect button: signs in and lists the spaces for the space lookup. */
    private fun addressControl(field: FieldDescriptor): JComponent {
        val address = JBTextField().also { it.emptyText.text = field.placeholder }
        readers[field.key] = { address.text }
        setters[field.key] = { address.text = it }
        val connect = JButton("Connect").apply { addActionListener { connect(address.text.trim()) } }

        return JPanel(BorderLayout(JBUI.scale(6), 0)).apply {
            add(address, BorderLayout.CENTER)
            add(connect, BorderLayout.EAST)
        }
    }

    private fun spaceControl(field: FieldDescriptor): JComponent {
        val box = JComboBox<Space>().apply { isEnabled = false }
        spaceBox = box
        readers[field.key] = { (box.selectedItem as? Space)?.key ?: pending[field.key].orEmpty() }
        setters[field.key] = { value ->
            val found = (0 until box.itemCount).map { box.getItemAt(it) }.firstOrNull { it.key == value }
            if (found != null) box.selectedItem = found else pending[field.key] = value
        }
        box.addActionListener { chosenPage = null }
        onConnected = { found ->
            box.removeAllItems()
            found.forEach(box::addItem)
            box.isEnabled = found.isNotEmpty()
            pending[field.key]?.let { kept -> found.firstOrNull { it.key == kept }?.let { box.selectedItem = it } }
        }

        return box
    }

    private fun pageControl(field: FieldDescriptor): JComponent {
        val label = JBLabel("(the space's home page)")
        val choose = JButton("Choose parent page...").apply { addActionListener { chooseParent(field, label) } }
        readers[field.key] = { chosenPage?.id ?: pending[field.key]?.takeIf { it.isNotBlank() } ?: (spaceBox?.selectedItem as? Space)?.homepageId.orEmpty() }
        setters[field.key] = { value ->
            pending[field.key] = value
            if (value.isNotBlank()) label.text = "(page $value)"
        }

        return JPanel(BorderLayout(JBUI.scale(6), 0)).apply {
            add(label, BorderLayout.CENTER)
            add(choose, BorderLayout.EAST)
        }
    }

    private fun connect(where: String) {
        if (where.isEmpty()) {
            status.text = "Enter the address first."

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
                    val connection = platform.connection
                    if (connection != null) setters[connection.field]?.invoke(found.first.baseUrl.ifBlank { where })
                    status.text = "Connected to ${found.first.baseUrl.ifBlank { where }} as ${found.first.user.displayName}."
                    onConnected(found.second)
                }.onFailure { status.text = "Could not connect: ${it.cause?.message ?: it.message}" }
            }
        }
    }

    private fun chooseParent(field: FieldDescriptor, label: JBLabel) {
        val id = sessionId
        val space = spaceBox?.selectedItem as? Space
        if (id == null || space == null) {
            status.text = "Connect and choose a space first."

            return
        }
        ApplicationManager.getApplication().executeOnPooledThread {
            val home = space.homepageId?.takeIf { it.isNotBlank() }?.let { RemotePage(it, "${space.name} home") }
            SwingUtilities.invokeLater {
                val dialog = ParentPageDialogComponent(project, engine, id, space, home)
                if (dialog.showAndGet()) {
                    dialog.chosen?.let { page ->
                        chosenPage = page
                        label.text = page.title
                        for (fill in field.lookup?.fills.orEmpty()) {
                            setters[fill.field]?.invoke(if (fill.from == "id") page.id else page.title)
                        }
                    }
                }
            }
        }
    }
}


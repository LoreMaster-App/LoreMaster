package loremaster.pluginsettings

import com.intellij.openapi.options.BoundSearchableConfigurable
import com.intellij.openapi.ui.DialogPanel
import com.intellij.ui.dsl.builder.COLUMNS_LARGE
import com.intellij.ui.dsl.builder.bindIntText
import com.intellij.ui.dsl.builder.bindItem
import com.intellij.ui.dsl.builder.bindText
import com.intellij.ui.dsl.builder.columns
import com.intellij.ui.dsl.builder.panel

private val PAGE_SIZES = listOf("A4", "A3", "A5", "Letter", "Legal")

/** Settings > Tools > LoreMaster. */
class PluginSettingsHandler : BoundSearchableConfigurable("LoreMaster", "loremaster.settings") {
    override fun createPanel(): DialogPanel {
        val settings = PluginSettingsStore.current()

        return panel {
            group("Engine") {
                row("Engine path:") {
                    textField().columns(COLUMNS_LARGE).bindText(settings::enginePath)
                        .comment("A locally built lore-master-engine. Blank uses the one bundled with the plugin (or the LORE_MASTER_ENGINE variable).")
                }
            }
            group("Export") {
                row("Folder:") {
                    textField().columns(COLUMNS_LARGE).bindText(settings::exportFolder)
                        .comment("Where exports go. Blank writes next to the document; a relative folder is under the project, mirroring the document's place.")
                }
                row("PDF page size:") { comboBox(PAGE_SIZES).bindItem({ settings.pageSize }, { settings.pageSize = it ?: "A4" }) }
                row("PNG width (pixels):") { intTextField(200..8000).bindIntText(settings::pngWidth) }
                row("Browser:") {
                    textField().columns(COLUMNS_LARGE).bindText(settings::browserPath)
                        .comment("The Chrome, Edge or Chromium that prints PDF and PNG. Blank looks for one.")
                }
                row("Word template:") {
                    textField().columns(COLUMNS_LARGE).bindText(settings::docxTemplate)
                        .comment("A .docx whose styles, headers, footers and page setup every Word export takes.")
                }
                row("OpenDocument template:") {
                    textField().columns(COLUMNS_LARGE).bindText(settings::odtTemplate)
                        .comment("An .odt (or .ott) used the same way.")
                }
            }
        }
    }
}

package loremaster.syncview

import com.intellij.openapi.project.Project
import com.intellij.openapi.wm.ToolWindow
import com.intellij.openapi.wm.ToolWindowFactory
import com.intellij.ui.content.ContentFactory

/**
 * The LoreMaster tool window, the Sync view: Local and each storage with their pages and the actions for them, the same layout as
 * the VS Code extension's Sync view. Double-click a page to open its file.
 */
class SyncToolWindowFactory : ToolWindowFactory {
    override fun createToolWindowContent(project: Project, toolWindow: ToolWindow) {
        val view = SyncViewComponent(project)
        val content = ContentFactory.getInstance().createContent(view, "Sync", false)
        content.setDisposer(view)
        toolWindow.contentManager.addContent(content)
    }
}

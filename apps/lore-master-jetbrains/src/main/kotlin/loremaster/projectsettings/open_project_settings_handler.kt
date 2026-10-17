package loremaster.projectsettings

import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.application.runWriteAction
import com.intellij.openapi.fileEditor.FileEditorManager
import com.intellij.openapi.vfs.LocalFileSystem
import com.intellij.openapi.vfs.VfsUtil
import java.nio.file.Files
import java.nio.file.Path

/**
 * "LoreMaster: Open Project Settings": the project's `.vscode/settings.json`. The engine reads the `loreMaster.*` keys there
 * (outputs, ignore, generators, ...), the same file VS Code uses, so a team can mix both editors. Created as `{}` when missing.
 */
class OpenProjectSettingsHandler : AnAction() {
    override fun actionPerformed(event: AnActionEvent) {
        val project = event.project ?: return
        val base = project.basePath ?: return
        val file = Path.of(base, ".vscode", "settings.json")
        if (!Files.exists(file)) {
            Files.createDirectories(file.parent)
            Files.writeString(file, "{\n}\n")
        }
        val virtual = runWriteAction { LocalFileSystem.getInstance().refreshAndFindFileByNioFile(file) } ?: return
        VfsUtil.markDirtyAndRefresh(false, false, false, virtual)
        FileEditorManager.getInstance(project).openFile(virtual, true)
    }
}

package loremaster.exportmarkdown

import java.nio.file.Path

/**
 * The folder an export goes into, given the export folder setting. Blank means next to the document (null: the engine's
 * default). Otherwise the document's place under the project is mirrored below that folder, so exporting a whole folder keeps
 * its layout and two files with the same name do not collide.
 */
fun exportDestination(documentPath: Path, projectRoot: Path?, setting: String): Path? {
    val folder = setting.trim()
    if (folder.isEmpty()) {
        return null
    }
    val configured = Path.of(folder)
    val base = if (configured.isAbsolute) configured else (projectRoot ?: documentPath.parent).resolve(configured)
    if (projectRoot == null) {
        return base
    }
    val relative = projectRoot.relativize(documentPath.parent)

    return if (relative.startsWith("..") || relative.isAbsolute) base else base.resolve(relative)
}

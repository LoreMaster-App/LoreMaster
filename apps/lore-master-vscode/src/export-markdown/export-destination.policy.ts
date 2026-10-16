import * as path from 'node:path'

/**
 * The folder an export goes into, given the `loreMaster.export.folder` setting. Empty means next to
 * the document (undefined: the engine's default). Otherwise the document's place under the workspace
 * is mirrored below that folder, so exporting a whole folder keeps its layout and two files with
 * the same name do not collide.
 */
export function exportDestination (documentPath: string, workspaceRoot: string | undefined, setting: string): string | undefined {
  const folder = setting.trim()
  if (folder === '') {
    return undefined
  }
  const base = path.isAbsolute(folder) ? folder : path.join(workspaceRoot ?? path.dirname(documentPath), folder)
  if (workspaceRoot === undefined) {
    return base
  }
  const relative = path.relative(workspaceRoot, path.dirname(documentPath))

  return relative.startsWith('..') || path.isAbsolute(relative) ? base : path.join(base, relative)
}

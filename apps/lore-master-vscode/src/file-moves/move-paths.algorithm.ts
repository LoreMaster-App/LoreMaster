/**
 * A file system path as a workspace-relative, '/'-separated path, or undefined when it is
 * outside the workspace folder. The comparison ignores case, which is how Windows and macOS
 * volumes behave.
 */
export function workspaceRelative (root: string, fsPath: string): string | undefined {
  const base = root.replaceAll('\\', '/').replace(/\/+$/, '')
  const target = fsPath.replaceAll('\\', '/')
  const prefix = `${base}/`
  if (!target.toLowerCase().startsWith(prefix.toLowerCase())) {
    return undefined
  }

  return target.slice(prefix.length)
}

/** Where a workspace-relative path is after the moves: a moved file, or a file inside a moved folder. */
export function afterMoves (path: string, moves: { from: string; to: string }[]): string {
  for (const move of moves) {
    if (path === move.from) {
      return move.to
    }
    if (path.startsWith(`${move.from}/`)) {
      return move.to + path.slice(move.from.length)
    }
  }

  return path
}

/** Whether a path may be a Markdown file or a folder holding some: anything else cannot matter. */
export function mayHoldMarkdown (relativePath: string, isFolder: boolean): boolean {
  return isFolder || relativePath.toLowerCase().endsWith('.md')
}

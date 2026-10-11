import { isAbsolute, relative, sep } from 'node:path'

/** Folders whose changes never matter: dependencies, build output and tool caches. Hidden
 *  folders are skipped too. Mirrors the command line's look at the workspace. */
const SKIPPED_FOLDERS = new Set(['node_modules', 'vendor', 'bin', 'obj', 'dist', 'build', 'out', 'target', 'venv', 'env', '__pycache__', 'coverage', 'testdata'])

/** The files that can matter: Markdown, the settings, and what the generators read. */
const WATCHED_EXTENSIONS = new Set([
  '.md', '.xml', '.go', '.mod', '.yaml', '.yml', '.json', '.ts', '.tsx', '.mts', '.cts', '.js', '.jsx',
  '.py', '.toml', '.cfg', '.cs', '.csproj', '.props', '.dart',
])

/** The configuration: the editor's settings, and the deprecated yaml until it is gone. */
const SETTINGS_FILES = new Set(['.lore-master.yaml', '.vscode/settings.json'])

/** The workspace-relative, '/'-separated path of a file, or undefined when it is outside the folder. */
export function workspacePath (folder: string, fsPath: string): string | undefined {
  const relativePath = relative(folder, fsPath)
  if (relativePath === '' || relativePath.startsWith('..') || isAbsolute(relativePath)) {
    return undefined
  }

  return relativePath.split(sep).join('/')
}

/** Whether a change to this file (workspace-relative, '/'-separated) can matter to the sync. */
export function isWatchedPath (path: string): boolean {
  if (SETTINGS_FILES.has(path)) {
    return true
  }
  const segments = path.split('/')
  if (segments.slice(0, -1).some(segment => segment.startsWith('.') || SKIPPED_FOLDERS.has(segment.toLowerCase()))) {
    return false
  }
  const name = segments.at(-1) ?? ''
  const dot = name.lastIndexOf('.')

  return !name.startsWith('.') && dot > 0 && WATCHED_EXTENSIONS.has(name.slice(dot).toLowerCase())
}

// Stands in for the 'vscode' module in unit tests: that module exists only
// inside the extension host. Extend it as the extension reaches for more API.
const registered = new Map<string, (...arguments_: unknown[]) => unknown>()

export const commands = {
  registerCommand (id: string, handler: (...arguments_: unknown[]) => unknown): { dispose: () => void } {
    registered.set(id, handler)

    return { dispose: () => { registered.delete(id) } }
  },
  getCommands (): Promise<string[]> {
    return Promise.resolve(registered.keys().toArray())
  },
  executeCommand (id: string, ...arguments_: unknown[]): Promise<unknown> {
    return Promise.resolve(registered.get(id)?.(...arguments_))
  },
}

// Each window prompt is a replaceable function so a test can script what the user does
// (reassign it), then restore it. They return "cancelled" by default.
export const window = {
  showInformationMessage (_message: string): Promise<undefined> {
    return Promise.resolve(undefined)
  },
  showErrorMessage (_message: string): Promise<undefined> {
    return Promise.resolve(undefined)
  },
  showInputBox (_options?: unknown): Promise<string | undefined> {
    return Promise.resolve(undefined)
  },
  showQuickPick (_items: unknown, _options?: unknown): Promise<unknown> {
    return Promise.resolve(undefined)
  },
  createOutputChannel (name: string): { name: string; appendLine: () => void; append: () => void; clear: () => void; show: () => void; hide: () => void; dispose: () => void } {
    return { name, appendLine () {}, append () {}, clear () {}, show () {}, hide () {}, dispose () {} }
  },
  activeTextEditor: undefined as { document: { uri: unknown } } | undefined,
}

export const workspace = {
  workspaceFolders: undefined as { uri: { fsPath: string }; name: string }[] | undefined,
  getWorkspaceFolder (_uri: unknown): { uri: { fsPath: string } } | undefined {
    return undefined
  },
  getConfiguration (_section?: string): { get: <T>(key: string, defaultValue?: T) => T | undefined } {
    return { get: <T>(_key: string, defaultValue?: T) => defaultValue }
  },
}

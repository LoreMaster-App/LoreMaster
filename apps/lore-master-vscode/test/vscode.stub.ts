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
  registerTreeDataProvider (_viewId: string, _provider: unknown): { dispose: () => void } {
    return { dispose () {} }
  },
  activeTextEditor: undefined as { document: { uri: unknown } } | undefined,
}

/** How a tree item can expand; matches vscode's enum values. */
export const TreeItemCollapsibleState = { None: 0, Collapsed: 1, Expanded: 2 } as const

/** A minimal vscode.TreeItem: holds the label and whatever the provider sets on it. */
export class TreeItem {
  command?:  { command: string; title: string }
  iconPath?: unknown

  constructor (public label: string, public collapsibleState: number = TreeItemCollapsibleState.None) {}
}

/** A minimal vscode.ThemeIcon: keeps the icon id. */
export class ThemeIcon {
  constructor (public id: string) {}
}

// A minimal EventEmitter matching vscode's: `event` registers a listener, `fire` notifies.
export class EventEmitter<T> {
  private listeners: ((event: T) => unknown)[] = []

  event = (listener: (event: T) => unknown): { dispose: () => void } => {
    this.listeners.push(listener)

    return { dispose: () => { this.listeners = this.listeners.filter(each => each !== listener) } }
  }

  fire (data: T): void {
    for (const listener of this.listeners) {
      listener(data)
    }
  }

  dispose (): void {
    this.listeners = []
  }
}

export const authentication = {
  registerAuthenticationProvider (_id: string, _label: string, _provider: unknown, _options?: unknown): { dispose: () => void } {
    return { dispose () {} }
  },
}

// A minimal vscode.env.clipboard: writeText stores, readText returns what was stored, so a
// test can assert what a command copied.
let clipboardContent = ''
export const env = {
  clipboard: {
    writeText (text: string): Promise<void> {
      clipboardContent = text

      return Promise.resolve()
    },
    readText (): Promise<string> {
      return Promise.resolve(clipboardContent)
    },
  },
}

export const workspace = {
  workspaceFolders: undefined as { uri: { fsPath: string }; name: string }[] | undefined,
  getWorkspaceFolder (_uri: unknown): { uri: { fsPath: string } } | undefined {
    return undefined
  },
  getConfiguration (_section?: string): { get: <T>(key: string, defaultValue?: T) => T | undefined } {
    return { get: <T>(_key: string, defaultValue?: T) => defaultValue }
  },
  onDidChangeWorkspaceFolders (_listener: () => unknown): { dispose: () => void } {
    return { dispose () {} }
  },
}

/** A minimal vscode.Uri: keeps the fsPath, which is all the extension reads. */
export const Uri = {
  file (fsPath: string): { fsPath: string; scheme: string } {
    return { fsPath, scheme: 'file' }
  },
}

/** Combines disposables, like vscode.Disposable.from. */
export const Disposable = {
  from (...items: { dispose: () => void }[]): { dispose: () => void } {
    return { dispose () { for (const item of items) item.dispose() } }
  },
}

/** A minimal vscode.McpStdioServerDefinition: records how the server is launched. */
export class McpStdioServerDefinition {
  cwd?: { fsPath: string }

  constructor (
    public label: string,
    public command: string,
    public args: string[] = [],
    public env: Record<string, string | number | null> = {},
    public version?: string,
  ) {}
}

// The MCP provider API. registerMcpServerDefinitionProvider keeps the provider so a test
// can fetch it by id and exercise provideMcpServerDefinitions.
interface McpServerDefinitionProvider {
  onDidChangeMcpServerDefinitions?: (listener: () => unknown) => { dispose: () => void }
  provideMcpServerDefinitions (token?: unknown): unknown
  resolveMcpServerDefinition?(server: unknown, token?: unknown): unknown
}

const mcpProviders = new Map<string, McpServerDefinitionProvider>()

export const lm = {
  registerMcpServerDefinitionProvider (id: string, provider: McpServerDefinitionProvider): { dispose: () => void } {
    mcpProviders.set(id, provider)

    return { dispose: () => { mcpProviders.delete(id) } }
  },
}

/** Test helper: the provider registered under an id, or undefined. */
export function mcpProvider (id: string): McpServerDefinitionProvider | undefined {
  return mcpProviders.get(id)
}

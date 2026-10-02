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

export const window = {
  showInformationMessage (_message: string): Promise<undefined> {
    return Promise.resolve(undefined)
  },
}

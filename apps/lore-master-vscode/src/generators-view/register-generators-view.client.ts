import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import { RUN_GENERATORS_COMMAND, runGeneratorsCommand } from '../generators-command'
import { ADD_GENERATOR_COMMAND, addGeneratorCommand, type GeneratorsViewDeps } from './add-generator.handler'
import { EDIT_GENERATOR_COMMAND, editGeneratorCommand } from './edit-generator.handler'
import { GENERATORS_VIEW_ID, type GeneratorNode, GeneratorsViewProvider } from './generators-view.client'
import { REMOVE_GENERATOR_COMMAND, removeGeneratorCommand } from './remove-generator.handler'
import { RUN_GENERATOR_COMMAND, runGeneratorCommand } from './run-generator.handler'

/** The command id contributed in package.json. */
export const REFRESH_GENERATORS_COMMAND = 'loreMaster.refreshGenerators'

/** How long to wait for a burst of changes to the settings file to settle before re-reading it. */
const REFRESH_DEBOUNCE_MS = 300

/**
 * Registers the Generators view and its commands, and keeps it current: it re-reads when
 * .lore-master.yaml changes, by hand or through another command. The Command Palette's
 * "Run generators" is registered here too, so what it ran shows on the rows.
 */
export function registerGeneratorsView (deps: { engine: EngineClient; output: vscode.OutputChannel }): vscode.Disposable {
  const provider = new GeneratorsViewProvider(deps.engine)
  const viewDeps: GeneratorsViewDeps = { ...deps, provider }

  let timer: ReturnType<typeof setTimeout> | undefined
  const watcher = vscode.workspace.createFileSystemWatcher('**/.lore-master.yaml')
  const refreshSoon = (): void => {
    clearTimeout(timer)
    timer = setTimeout(() => provider.refresh(), REFRESH_DEBOUNCE_MS)
  }

  return vscode.Disposable.from(
    vscode.window.registerTreeDataProvider(GENERATORS_VIEW_ID, provider),
    vscode.commands.registerCommand(REFRESH_GENERATORS_COMMAND, () => provider.refresh()),
    vscode.commands.registerCommand(ADD_GENERATOR_COMMAND, () => addGeneratorCommand(viewDeps)),
    vscode.commands.registerCommand(RUN_GENERATOR_COMMAND, (node: GeneratorNode | undefined) => runGeneratorCommand(viewDeps, node)),
    vscode.commands.registerCommand(EDIT_GENERATOR_COMMAND, (node: GeneratorNode | undefined) => editGeneratorCommand(viewDeps, node)),
    vscode.commands.registerCommand(REMOVE_GENERATOR_COMMAND, (node: GeneratorNode | undefined) => removeGeneratorCommand(viewDeps, node)),
    vscode.commands.registerCommand(RUN_GENERATORS_COMMAND, async () => {
      const result = await runGeneratorsCommand(deps)
      if (result) {
        provider.recordRun(result)
      }
    }),
    watcher.onDidCreate(refreshSoon),
    watcher.onDidChange(refreshSoon),
    watcher.onDidDelete(refreshSoon),
    watcher,
    vscode.workspace.onDidChangeWorkspaceFolders(() => provider.refresh()),
    { dispose: () => clearTimeout(timer) },
  )
}

import * as vscode from 'vscode'
import { GENERATE_AND_SYNC_COMMAND } from '../generate-and-sync'
import { runAndReport } from '../generators-command'
import { generatorTypeOf, readGeneratorSettings, type GeneratorNode, type GeneratorsViewDeps } from '../generators-view'
import { pickWorkspaceFolder } from '../workspace-files'
import type { GeneratorFormInit, GeneratorFormReply, GeneratorFormRequest } from './generator-form.contract'
import { renderGeneratorFormPage } from './generator-form-page.mapper'
import { allKinds, generatorToValues, kindsFor } from './generator-form-values.mapper'
import { saveGenerator } from './save-generator.use-case'

/** The command ids contributed in package.json. */
export const ADD_GENERATOR_COMMAND = 'loreMaster.addGenerator'
export const EDIT_GENERATOR_COMMAND = 'loreMaster.editGenerator'

const RUN_NOW = 'Run now'
const GENERATE_AND_SYNC = 'Generate and sync'

function nonce (): string {
  const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  let value = ''
  for (let index = 0; index < 32; index += 1) {
    value += alphabet.charAt(Math.floor(Math.random() * alphabet.length))
  }

  return value
}

function messageOf (error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

/** Opens the form to add a generator to the chosen workspace folder. */
export async function addGeneratorForm (deps: GeneratorsViewDeps): Promise<void> {
  const folder = await pickWorkspaceFolder()
  if (!folder) {
    await vscode.window.showInformationMessage('LoreMaster: open a folder to add a generator.')

    return
  }
  try {
    const settings = await readGeneratorSettings(deps.engine, folder)
    open(deps, folder, {
      mode:         'create',
      values:       { kind: '', output: '', reads: '', command: '', title: '' },
      kinds:        allKinds(),
      takenOutputs: settings.generators.map(generator => generator.output),
    }, undefined)
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${messageOf(error)}`)
  }
}

/** Opens the form on the generator a row stands for. */
export async function editGeneratorForm (deps: GeneratorsViewDeps, node: GeneratorNode | undefined): Promise<void> {
  if (!node) {
    return
  }
  const folder = await pickWorkspaceFolder()
  if (!folder) {
    return
  }
  try {
    const settings = await readGeneratorSettings(deps.engine, folder)
    const generator = settings.generators[node.index]
    if (!generator) {
      await vscode.window.showErrorMessage('LoreMaster: that generator is no longer in your LoreMaster settings.')

      return
    }
    open(deps, folder, {
      mode:         'edit',
      values:       generatorToValues(generator),
      kinds:        kindsFor(generator),
      takenOutputs: settings.generators.filter((_, position) => position !== node.index).map(other => other.output),
    }, node.index)
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${messageOf(error)}`)
  }
}

/** One form panel: saving is driven by its messages until it closes. */
function open (deps: GeneratorsViewDeps, folder: string, init: GeneratorFormInit, index: number | undefined): void {
  const panel = vscode.window.createWebviewPanel(
    'loreMasterGeneratorForm',
    init.mode === 'edit' ? 'LoreMaster: edit generator' : 'LoreMaster: add generator',
    vscode.ViewColumn.Active,
    { enableScripts: true, retainContextWhenHidden: true },
  )
  panel.webview.html = renderGeneratorFormPage(nonce(), init.mode === 'edit' ? 'Edit generator' : 'Add a generator')
  const reply = (message: GeneratorFormReply): void => { void panel.webview.postMessage(message) }

  panel.webview.onDidReceiveMessage(async (request: GeneratorFormRequest) => {
    switch (request.type) {
      case 'ready': {
        reply({ type: 'init', init })
        break
      }
      case 'cancel': {
        panel.dispose()
        break
      }
      case 'save': {
        const result = await saveGenerator({ engine: deps.engine, workspaceRoot: folder, index }, request.values)
        if (!result.saved) {
          reply({ type: 'invalid', errors: result.errors, message: result.message })

          break
        }
        panel.dispose()
        deps.provider.refresh()
        if (index !== undefined) {
          break
        }
        const label = generatorTypeOf(result.generator.type).label
        const answer = await vscode.window.showInformationMessage(
          `LoreMaster: added the ${label} generator → ${result.generator.output}.${result.notSynced ? ` ${result.notSynced}` : ''}`,
          RUN_NOW, GENERATE_AND_SYNC,
        )
        if (answer === RUN_NOW) {
          deps.provider.recordRun(await runAndReport(deps, folder, [result.index]))
        } else if (answer === GENERATE_AND_SYNC) {
          await vscode.commands.executeCommand(GENERATE_AND_SYNC_COMMAND)
          deps.provider.refresh()
        }
        break
      }
      default: {
        break
      }
    }
  })
}

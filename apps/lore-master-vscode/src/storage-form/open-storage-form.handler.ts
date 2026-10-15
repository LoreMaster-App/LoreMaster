import * as vscode from 'vscode'
import { connectConfluence } from '../connection-setup'
import type { EngineClient } from '../engine-process'
import { type Output, SETTINGS_READ_METHOD, type Settings, type SettingsReadResult } from '../engine-protocol'
import type { ConnectionStore } from '../secret-storage'
import { isConfiguredOutput, REFRESH_STORAGES_COMMAND, storageDescription, storageLabel, type StorageNode } from '../sidebar'
import { pickWorkspaceFolder } from '../workspace-files'
import { saveStorage } from './save-storage.use-case'
import type { ConnectionChoice, FormReply, FormRequest, StorageFormInit } from './storage-form.contract'
import { renderStorageFormPage } from './storage-form-page.mapper'
import { emptyValues, outputToValues } from './storage-form-values.mapper'
import { createStorageLookup } from './storage-lookup.client'

export interface StorageFormDeps {
  engine:      EngineClient
  connections: ConnectionStore
}

function choices (connections: ConnectionStore): ConnectionChoice[] {
  return connections.list().map(meta => ({ baseUrl: meta.baseUrl, label: meta.displayName ? `${meta.baseUrl} (${meta.displayName})` : meta.baseUrl }))
}

function messageOf (error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

function nonce (): string {
  const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  let value = ''
  for (let index = 0; index < 32; index += 1) {
    value += alphabet.charAt(Math.floor(Math.random() * alphabet.length))
  }

  return value
}

/** Opens the form to add a storage to the chosen workspace folder. */
export async function addStorageForm (deps: StorageFormDeps): Promise<void> {
  const folder = await pickWorkspaceFolder()
  if (!folder) {
    await vscode.window.showInformationMessage('LoreMaster: open a folder first.')

    return
  }
  let read: SettingsReadResult
  try {
    read = await deps.engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot: folder })
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${messageOf(error)}`)

    return
  }
  const values = { ...emptyValues(), ignore: (read.settings.ignore ?? []).join(', ') }
  const metas = deps.connections.list()
  if (metas.length === 1) {
    values.connection = metas[0].baseUrl
  }
  open(deps, folder, { mode: 'create', values, connections: choices(deps.connections) }, undefined)
}

/** Opens the form on a storage: the row it was invoked on, or the one the user picks. */
export async function editStorageForm (deps: StorageFormDeps, node: StorageNode | undefined): Promise<void> {
  const folder = await pickWorkspaceFolder()
  if (!folder) {
    await vscode.window.showInformationMessage('LoreMaster: open a folder first.')

    return
  }
  let settings: Settings
  try {
    const read = await deps.engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot: folder })
    settings = read.settings
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${messageOf(error)}`)

    return
  }
  const target = node ?? await pickStorage(settings)
  if (!target) {
    return
  }
  open(deps, folder, { mode: 'edit', values: outputToValues(target.output, settings), connections: choices(deps.connections) }, target.index)
}

async function pickStorage (settings: Settings): Promise<{ index: number, output: Output } | undefined> {
  const options = settings.outputs.map((output, index) => ({ index, output })).filter(each => isConfiguredOutput(each.output))
  if (options.length === 0) {
    await vscode.window.showInformationMessage('LoreMaster: no storage to edit yet. Add one with "Add sync storage".')

    return undefined
  }
  if (options.length === 1) {
    return options[0]
  }
  const picked = await vscode.window.showQuickPick(
    options.map(each => ({ label: storageLabel(each.output), description: storageDescription(each.output), each })),
    { title: 'LoreMaster: which storage?' },
  )

  return picked?.each
}

/** One form panel: lookups and saving are driven by its messages until it closes. */
function open (deps: StorageFormDeps, folder: string, init: StorageFormInit, index: number | undefined): void {
  const panel = vscode.window.createWebviewPanel(
    'loreMasterStorageForm',
    init.mode === 'edit' ? 'LoreMaster: edit storage' : 'LoreMaster: add storage',
    vscode.ViewColumn.Active,
    { enableScripts: true, retainContextWhenHidden: true },
  )
  panel.webview.html = renderStorageFormPage(nonce(), init.mode === 'edit' ? 'Edit storage' : 'Add a storage')
  const lookup = createStorageLookup(deps)
  const reply = (message: FormReply): void => { void panel.webview.postMessage(message) }

  panel.onDidDispose(() => { void lookup.dispose() })
  panel.webview.onDidReceiveMessage(async (request: FormRequest) => {
    switch (request.type) {
      case 'ready': {
        reply({ type: 'init', init })
        break
      }
      case 'cancel': {
        panel.dispose()
        break
      }
      case 'listSpaces': {
        try {
          reply({ type: 'spaces', spaces: await lookup.listSpaces(request.connection) })
        } catch (error) {
          reply({ type: 'lookupError', message: messageOf(error) })
        }
        break
      }
      case 'searchPages': {
        try {
          reply({ type: 'pages', pages: await lookup.searchPages(request.connection, request.space, request.query) })
        } catch (error) {
          reply({ type: 'lookupError', message: messageOf(error) })
        }
        break
      }
      case 'addConnection': {
        const meta = await connectConfluence({ engine: deps.engine, store: deps.connections })
        reply({ type: 'connections', connections: choices(deps.connections), selected: meta?.baseUrl ?? '' })
        break
      }
      case 'save': {
        const result = await saveStorage({ engine: deps.engine, workspaceRoot: folder, index, connectionCount: deps.connections.list().length }, request.values)
        if (result.saved) {
          panel.dispose()
          await vscode.commands.executeCommand(REFRESH_STORAGES_COMMAND)
          await vscode.window.showInformationMessage('LoreMaster: storage saved.')
        } else {
          reply({ type: 'invalid', errors: result.errors, message: result.message })
        }
        break
      }
      default: {
        break
      }
    }
  })
}

import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import { type Output, SETTINGS_READ_METHOD, type Settings, type SettingsReadResult } from '../engine-protocol'
import { isConfiguredOutput, REFRESH_STORAGES_COMMAND, storageDescription, storageLabel, type StorageNode } from '../sidebar'
import { editStorage } from './edit-storage.use-case'
import { fieldsFor, type StorageField } from './storage-field.config'

/** The command id contributed in package.json. */
export const EDIT_STORAGE_COMMAND = 'loreMaster.editStorage'

/**
 * Changes one setting of a storage, one pick at a time: which storage (unless the row it was
 * invoked on says), which setting, then its new value. The engine validates and saves it, so
 * a refused value shows the engine's reason and writes nothing.
 */
export async function editStorageCommand (deps: { engine: EngineClient }, node: StorageNode | undefined): Promise<void> {
  const folder = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath
  if (!folder) {
    await vscode.window.showInformationMessage('LoreMaster: open a folder first.')

    return
  }

  const target = node ?? await pickStorage(deps.engine, folder)
  if (!target) {
    return
  }
  const settings = await readSettings(deps.engine, folder)
  const field = await pickField(target.output, settings)
  if (!field) {
    return
  }
  const value = await askValue(field, field.read(target.output, settings))
  if (value === undefined) {
    return
  }

  try {
    await editStorage({ engine: deps.engine, workspaceRoot: folder, index: target.index }, field, value)
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${error instanceof Error ? error.message : String(error)}`)

    return
  }
  await vscode.commands.executeCommand(REFRESH_STORAGES_COMMAND)
}

async function pickStorage (engine: EngineClient, folder: string): Promise<{ index: number; output: Output } | undefined> {
  let read: SettingsReadResult
  try {
    read = await engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot: folder })
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${error instanceof Error ? error.message : String(error)}`)

    return undefined
  }
  const choices = read.settings.outputs.map((output, index) => ({ index, output })).filter(each => isConfiguredOutput(each.output))
  if (choices.length === 0) {
    await vscode.window.showInformationMessage('LoreMaster: no storage to edit yet. Add one with "Add sync storage".')

    return undefined
  }
  if (choices.length === 1) {
    return choices[0]
  }

  const picked = await vscode.window.showQuickPick(
    choices.map(each => ({ label: storageLabel(each.output), description: storageDescription(each.output), each })),
    { title: 'LoreMaster: edit which storage?' },
  ) as { each: { index: number; output: Output } } | undefined

  return picked?.each
}

/** The settings as saved, or undefined when they cannot be read (the picks still work; a field then shows no current value). */
async function readSettings (engine: EngineClient, folder: string): Promise<Settings | undefined> {
  try {
    return (await engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot: folder })).settings
  } catch {
    return undefined
  }
}

async function pickField (output: Output, settings: Settings | undefined): Promise<StorageField | undefined> {
  const picked = await vscode.window.showQuickPick(
    fieldsFor(output).map(field => ({ label: field.label, description: field.read(output, settings) || '(default)', field })),
    { title: `LoreMaster: change a setting of ${storageLabel(output)}` },
  ) as { field: StorageField } | undefined

  return picked?.field
}

function askValue (field: StorageField, current: string): Promise<string | undefined> {
  if (field.kind === 'enum') {
    return pickOption(field, current)
  }

  return Promise.resolve(vscode.window.showInputBox({ title: `LoreMaster: ${field.label}`, prompt: field.hint, value: current }))
}

async function pickOption (field: StorageField, current: string): Promise<string | undefined> {
  const picked = await vscode.window.showQuickPick(
    (field.options ?? []).map(option => ({ label: option, description: option === current ? 'current' : '' })),
    { title: `LoreMaster: ${field.label}` },
  ) as { label: string } | undefined

  return picked?.label
}

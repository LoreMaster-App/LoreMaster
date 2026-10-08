import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import { type Output, SETTINGS_READ_METHOD, SETTINGS_SAVE_METHOD } from '../engine-protocol'
import { editStorageCommand } from './edit-storage.handler'

const confluence: Output = {
  platform:       'confluence',
  baseUrl:        'https://x.atlassian.net/wiki',
  space:          'ENG',
  parentPageId:   '1',
  titlePrefix:    'ENG',
  direction:      'to-platform',
  content:        [{ type: 'markdown', roots: ['.'], template: 'default' }],
  mermaidMode:    'image',
  titleCollision: 'fail',
  linkMode:       'title',
}
const second: Output = { ...confluence, space: 'OPS' }
const scaffold: Output = { ...confluence, baseUrl: '', space: '' }

interface Prompts {
  quickPicks: ((items: { label: string }[]) => { label: string } | undefined)[]
  inputs:     (string | undefined)[]
  messages:   string[]
  executed:   string[]
}

const window = vscode.window as unknown as Record<string, unknown>
const original = { ...window }
const commands = vscode.commands as unknown as Record<string, unknown>
const originalExecute = commands.executeCommand

function script (prompts: Prompts): void {
  window.showQuickPick = (items: { label: string }[]) => Promise.resolve(prompts.quickPicks.shift()?.(items))
  window.showInputBox = () => Promise.resolve(prompts.inputs.shift())
  window.showInformationMessage = (message: string) => {
    prompts.messages.push(message)

    return Promise.resolve(undefined)
  }
  window.showErrorMessage = (message: string) => {
    prompts.messages.push(message)

    return Promise.resolve(undefined)
  }
  commands.executeCommand = (id: string) => {
    prompts.executed.push(id)

    return Promise.resolve(undefined)
  }
}

function engine (outputs: Output[], requests: { method: string; params: unknown }[], refuse?: string): EngineClient {
  return {
    request (method: string, params?: unknown) {
      requests.push({ method, params })
      if (method === SETTINGS_READ_METHOD) {
        return Promise.resolve({ exists: true, firstSync: false, settings: { version: 1, outputs } } as never)
      }
      if (method === SETTINGS_SAVE_METHOD && refuse) {
        return Promise.reject(new Error(refuse))
      }

      return Promise.resolve(null as never)
    },
  } as unknown as EngineClient
}

const pick = (label: string) => (items: { label: string }[]) => items.find(item => item.label === label)

const saved = (requests: { method: string; params: unknown }[]): { settings: { outputs: Output[] } }[] =>
  requests.filter(request => request.method === SETTINGS_SAVE_METHOD).map(request => request.params as { settings: { outputs: Output[] } })

function setFolders (folders: { uri: { fsPath: string }; name: string }[] | undefined): void {
  (vscode.workspace as { workspaceFolders: unknown }).workspaceFolders = folders
}

describe('editStorageCommand', () => {
  let prompts: Prompts
  beforeEach(() => {
    prompts = { quickPicks: [], inputs: [], messages: [], executed: [] }
    script(prompts)
    setFolders([{ uri: { fsPath: '/w' }, name: 'w' }])
  })
  afterEach(() => {
    Object.assign(window, original)
    commands.executeCommand = originalExecute
    setFolders(undefined)
  })

  it('asks which setting, then its value from the allowed ones, for the row it was invoked on', async () => {
    const requests: { method: string; params: unknown }[] = []
    prompts.quickPicks.push(pick('Direction'), pick('two-way'))

    await editStorageCommand({ engine: engine([confluence, second], requests) }, { index: 1, output: second })

    const [save] = saved(requests)
    expect(save.settings.outputs[1].direction).toBe('two-way')
    expect(save.settings.outputs[0].direction).toBe('to-platform')
    expect(prompts.executed).toEqual(['loreMaster.refreshStorages'])
  })

  it('asks for text in an input box, starting from the current value', async () => {
    const requests: { method: string; params: unknown }[] = []
    prompts.quickPicks.push(pick('Title prefix'))
    prompts.inputs.push('HANDBOOK')

    await editStorageCommand({ engine: engine([confluence], requests) }, { index: 0, output: confluence })

    expect(saved(requests)[0].settings.outputs[0].titlePrefix).toBe('HANDBOOK')
  })

  it('asks which storage when invoked from the palette with several, and goes straight to the one when there is one', async () => {
    const requests: { method: string; params: unknown }[] = []
    prompts.quickPicks.push(items => items.find(item => item.label.includes('OPS')), pick('Links between pages'), pick('id'))

    await editStorageCommand({ engine: engine([confluence, second, scaffold], requests) }, undefined)
    expect(saved(requests)[0].settings.outputs[1].linkMode).toBe('id')

    const solo: { method: string; params: unknown }[] = []
    prompts.quickPicks.push(pick('Mermaid diagrams'), pick('code'))
    await editStorageCommand({ engine: engine([confluence, scaffold], solo) }, undefined)
    expect(saved(solo)[0].settings.outputs[0].mermaidMode).toBe('code')
  })

  it('writes nothing when any prompt is cancelled', async () => {
    const requests: { method: string; params: unknown }[] = []

    await editStorageCommand({ engine: engine([confluence], requests) }, { index: 0, output: confluence })
    expect(saved(requests)).toEqual([])

    prompts.quickPicks.push(pick('Direction'), () => undefined)
    await editStorageCommand({ engine: engine([confluence], requests) }, { index: 0, output: confluence })
    expect(saved(requests)).toEqual([])

    prompts.quickPicks.push(pick('Title prefix'))
    prompts.inputs.push(undefined)
    await editStorageCommand({ engine: engine([confluence], requests) }, { index: 0, output: confluence })
    expect(saved(requests)).toEqual([])
    expect(prompts.executed).toEqual([])
  })

  it('shows the engine\'s reason when it refuses the value, and does not refresh', async () => {
    prompts.quickPicks.push(pick('Mermaid diagrams'), pick('code'))

    await editStorageCommand({ engine: engine([confluence], [], 'mermaidMode "x" is not valid') }, { index: 0, output: confluence })

    expect(prompts.messages).toEqual(['LoreMaster: mermaidMode "x" is not valid'])
    expect(prompts.executed).toEqual([])
  })

  it('says so when there is no folder or no storage to edit', async () => {
    setFolders(undefined)
    await editStorageCommand({ engine: engine([confluence], []) }, undefined)
    expect(prompts.messages.at(-1)).toContain('open a folder')

    setFolders([{ uri: { fsPath: '/w' }, name: 'w' }])
    await editStorageCommand({ engine: engine([scaffold], []) }, undefined)
    expect(prompts.messages.at(-1)).toContain('no storage to edit')
  })
})

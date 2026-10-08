import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import { COPY_MCP_CONFIG_COMMAND } from '../mcp-server'
import { pickWorkspaceFolder } from '../workspace-files'
import { AGENT_TARGETS, type AgentTarget } from './agent-target.config'
import { createAgent, type CreateAgentResult, type FileStore } from './create-agent.use-case'
import { createWorkspaceFileStore } from './workspace-file-store.client'

/** The command id contributed in package.json. */
export const CREATE_AGENT_COMMAND = 'loreMaster.createAgent'

export interface CreateAgentCommandDeps {
  engine: EngineClient
  /** The files of a workspace folder; the VS Code file system unless a test supplies one. */
  files?: (folder: string) => FileStore
}

const OPEN = 'Open'
const COPY_MCP = 'Copy MCP Server Config'

/**
 * Creates an AI agent that writes documentation the way LoreMaster expects, for the tools the
 * user picks. The instructions come from the engine, built from the live rules and this
 * workspace's settings; running the command again refreshes the files it wrote. It ends by
 * pointing at the MCP server, which gives the agent the place and validate tools.
 */
export async function createAgentCommand (deps: CreateAgentCommandDeps): Promise<void> {
  const folder = await pickWorkspaceFolder()
  if (!folder) {
    await vscode.window.showInformationMessage('LoreMaster: open a folder to create its documentation agent.')

    return
  }
  const targets = await pickTargets()
  if (targets === undefined || targets.length === 0) {
    return
  }

  let result: CreateAgentResult
  try {
    result = await createAgent({
      engine:           deps.engine,
      files:            (deps.files ?? createWorkspaceFileStore)(folder),
      workspaceRoot:    folder,
      confirmOverwrite: async path => await vscode.window.showWarningMessage(`${path} exists and was not written by LoreMaster. Replace it?`, { modal: true }, 'Replace') === 'Replace',
    }, targets)
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${error instanceof Error ? error.message : String(error)}`)

    return
  }

  await report(folder, result)
}

async function pickTargets (): Promise<AgentTarget[] | undefined> {
  const picked = await vscode.window.showQuickPick(
    AGENT_TARGETS.map(target => ({ label: target.label, description: target.description, picked: target.kind === 'file', target })),
    { title: 'LoreMaster: create a documentation agent for…', canPickMany: true, placeHolder: 'Pick the tools that should get the agent' },
  ) as { target: AgentTarget }[] | undefined

  return picked?.map(item => item.target)
}

async function report (folder: string, result: CreateAgentResult): Promise<void> {
  const lines = result.outcomes.map(({ target, outcome }) => `${outcome} ${target.path}`)
  const note = result.hasConfig ? '' : ' There is no .lore-master.yaml yet, so the instructions are general; run it again after the first sync.'
  const written = result.outcomes.find(entry => entry.outcome !== 'skipped' && entry.outcome !== 'unchanged')
  const answer = await vscode.window.showInformationMessage(
    `LoreMaster agent: ${lines.join('; ')}.${note} For the place and validate tools, add the LoreMaster MCP server.`,
    ...(written ? [OPEN] : []),
    COPY_MCP,
  )

  if (answer === OPEN && written) {
    await vscode.commands.executeCommand('vscode.open', vscode.Uri.joinPath(vscode.Uri.file(folder), ...written.target.path.split('/')))
  } else if (answer === COPY_MCP) {
    await vscode.commands.executeCommand(COPY_MCP_CONFIG_COMMAND)
  }
}

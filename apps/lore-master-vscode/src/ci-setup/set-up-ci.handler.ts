import * as vscode from 'vscode'
import { createWorkspaceFileStore } from '../agent-creation'
import type { EngineClient } from '../engine-process'
import { pickWorkspaceFolder } from '../workspace-files'
import { CI_PROVIDERS, type CiProvider } from './ci-provider.config'
import { detectCiProviders } from './detect-ci-providers.policy'
import { buildCiSetupGuide, planCiSetup, writePipeline } from './set-up-ci.use-case'

/** The command id contributed in package.json. */
export const SET_UP_CI_COMMAND = 'loreMaster.setUpCi'

export interface SetUpCiCommandDeps {
  engine:  EngineClient
  /** The extension's version, which the pipeline pins the engine to. */
  version: string
}

const COPY = 'Copy pipeline'

/**
 * Guides adding LoreMaster to the repository's CI. It reads the settings configured locally,
 * works out which CI the repository uses (and asks when it cannot tell), opens a guide written
 * from those settings and offers the pipeline file to copy or to add.
 */
export async function setUpCiCommand (deps: SetUpCiCommandDeps): Promise<void> {
  const folder = await pickWorkspaceFolder()
  if (!folder) {
    await vscode.window.showInformationMessage('LoreMaster: open a folder to add it to your CI pipeline.')

    return
  }

  const files = createWorkspaceFileStore(folder)
  try {
    const plan = await planCiSetup({ engine: deps.engine, files, workspaceRoot: folder, version: deps.version })
    if (plan.kind === 'no-outputs') {
      await vscode.window.showInformationMessage('LoreMaster: this folder has no outputs yet. Add a storage first, then add it to your CI.')

      return
    }

    const provider = await chooseProvider(folder)
    if (!provider) {
      return
    }

    const { guide, pipeline } = buildCiSetupGuide(provider, plan.needs)
    const document = await vscode.workspace.openTextDocument({ language: 'markdown', content: guide })
    await vscode.window.showTextDocument(document, { preview: false })

    const add = `Add ${pipeline.path}`
    const answer = await vscode.window.showInformationMessage(
      `LoreMaster: the ${provider.label} guide is open. The pipeline is ready to copy or add.`,
      COPY,
      add,
    )

    if (answer === COPY) {
      await vscode.env.clipboard.writeText(pipeline.content)
      await vscode.window.showInformationMessage('LoreMaster: the pipeline is on your clipboard.')
    } else if (answer === add) {
      const outcome = await writePipeline(files, pipeline, async path =>
        await vscode.window.showWarningMessage(`${path} exists with other content. Replace it?`, { modal: true }, 'Replace') === 'Replace')
      await vscode.window.showInformationMessage(`LoreMaster: ${outcome} ${pipeline.path}. Create the secrets from the guide, then commit it.`)
      if (outcome === 'created' || outcome === 'updated') {
        await vscode.commands.executeCommand('vscode.open', vscode.Uri.joinPath(vscode.Uri.file(folder), ...pipeline.path.split('/')))
      }
    }
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${error instanceof Error ? error.message : String(error)}`)
  }
}

/** The CI the repository uses, confirmed when there is one and asked when there are none or several. */
async function chooseProvider (folder: string): Promise<CiProvider | undefined> {
  const found = await findPipelineFiles(folder)
  const detected = detectCiProviders(found)

  const picked = await vscode.window.showQuickPick(
    CI_PROVIDERS.map(provider => ({
      label:       provider.label,
      description: detected.includes(provider) ? 'found in this repository' : undefined,
      picked:      detected.length === 1 && detected[0] === provider,
      provider,
    })).sort((a, b) => Number(b.description !== undefined) - Number(a.description !== undefined)),
    {
      title:       'LoreMaster: which CI should run it?',
      placeHolder: detected.length === 0 ? 'No pipeline found in this repository: pick the CI you use' : 'Pick the CI to write the guide for',
    },
  )

  return picked?.provider
}

async function findPipelineFiles (folder: string): Promise<string[]> {
  const root = vscode.Uri.file(folder)
  const globs = CI_PROVIDERS.flatMap(provider => provider.detectGlobs)
  const uris = await vscode.workspace.findFiles(new vscode.RelativePattern(root, `{${globs.join(',')}}`))

  return uris.map(uri => vscode.workspace.asRelativePath(uri, false).replaceAll('\\', '/'))
}

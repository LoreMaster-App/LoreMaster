import * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import type { Generator, GeneratorsRunResult } from '../engine-protocol'
import { pickWorkspaceFolder } from '../workspace-files'
import { summariseRun } from './generator-run-report.mapper'
import { approvedPrepareCommands } from './prepare-approvals.store'
import { decidePrepare, parsePrepareSetting } from './prepare-steps.policy'
import { listGenerators, runGenerators } from './run-generators.use-case'

/** The setting that says whether generators' prepare commands run: ask, always or never. */
export const RUN_PREPARE_STEPS_SETTING = 'runPrepareSteps'

/** The command id contributed in package.json. */
export const RUN_GENERATORS_COMMAND = 'loreMaster.runGenerators'

/** What the command is given at registration. */
export interface RunGeneratorsDeps {
  engine: EngineClient
  output: vscode.OutputChannel
}

/**
 * Runs the workspace's generators: test results, Go package documentation, OpenAPI
 * descriptions. They write ordinary Markdown into the folders `.lore-master.yaml` names, which
 * the next sync publishes like any other page. With several generators it asks which; the
 * answer is a one-line summary, with the details and any problems in the LoreMaster output.
 */
export async function runGeneratorsCommand (deps: RunGeneratorsDeps): Promise<GeneratorsRunResult | undefined> {
  const folder = await pickWorkspaceFolder()
  if (!folder) {
    await vscode.window.showInformationMessage('LoreMaster: open a folder to run its generators.')

    return undefined
  }

  try {
    const configured = await listGenerators(deps.engine, folder)
    if (configured.length === 0) {
      await vscode.window.showInformationMessage('LoreMaster: no generators are configured. Add a loreMaster.generators list to your settings (see docs/generators.md), or add one from the Generators view.')

      return undefined
    }
    const chosen = await chooseGenerators(configured)
    if (chosen === undefined) {
      return undefined
    }

    return await runAndReport(deps, folder, chosen, configured)
  } catch (error) {
    await vscode.window.showErrorMessage(`LoreMaster: ${error instanceof Error ? error.message : String(error)}`)

    return undefined
  }
}

/** Which generators to run: all of them (`[]` is never sent; `undefined` means cancelled). */
async function chooseGenerators (configured: Generator[]): Promise<number[] | undefined> {
  if (configured.length === 1) {
    return [0]
  }
  const picked = await vscode.window.showQuickPick(
    [
      { label: 'All generators', description: `${configured.length} configured`, indexes: configured.map((_, index) => index) },
      ...configured.map((generator, index) => ({ label: generator.type, description: `→ ${generator.output}`, indexes: [index] })),
    ],
    { title: 'LoreMaster: run which generators?' },
  ) as { indexes: number[] } | undefined

  return picked?.indexes
}

/**
 * Whether to run the prepare commands of the generators about to run (`configured` is the
 * workspace's generator list). A command is code the settings carry, so it runs only in a trusted
 * workspace, and under `ask` the user sees the exact commands first.
 */
async function shouldPrepare (deps: RunGeneratorsDeps, configured: Generator[], indexes?: number[]): Promise<boolean> {
  const selected = indexes === undefined ? configured : indexes.map(index => configured[index]).filter(Boolean)
  const commands = [...new Set(selected.map(generator => generator.prepare?.trim()).filter((command): command is string => Boolean(command)))]
  const config = vscode.workspace.getConfiguration('loreMaster')
  const decision = decidePrepare({
    commands,
    setting:  parsePrepareSetting(config.get(RUN_PREPARE_STEPS_SETTING)),
    trusted:  vscode.workspace.isTrusted,
    approved: approvedPrepareCommands,
  })

  switch (decision.kind) {
    case 'run': {
      return true
    }
    case 'skip': {
      if (decision.reason === 'untrusted') {
        deps.output.appendLine('The prepare commands of the generators were not run: trust this workspace to allow them.')
      }

      return false
    }
    case 'ask': {
      const answer = await vscode.window.showInformationMessage(
        `LoreMaster: run these commands before generating?\n\n${decision.commands.join('\n')}`,
        { modal: true },
        'Run',
        'Always run',
      )
      if (answer === undefined) {
        return false
      }
      for (const command of decision.commands) {
        approvedPrepareCommands.add(command)
      }
      if (answer === 'Always run') {
        await config.update(RUN_PREPARE_STEPS_SETTING, 'always', vscode.ConfigurationTarget.Global)
      }

      return true
    }
  }
}

/**
 * Runs the generators at `indexes` (all when absent) under a progress notification, writes
 * the details to the LoreMaster output, and answers with the one-line summary: a warning that
 * opens the output when something failed or was left alone. Returns what the engine said so a
 * caller can show it. Errors are the caller's to report. Pass `configured` (the generator list)
 * for the generators' prepare commands to be considered; without it none runs.
 */
export async function runAndReport (deps: RunGeneratorsDeps, folder: string, indexes?: number[], configured?: Generator[]): Promise<GeneratorsRunResult> {
  const prepare = configured === undefined ? false : await shouldPrepare(deps, configured, indexes)
  const result = await vscode.window.withProgress(
    { location: vscode.ProgressLocation.Notification, title: 'LoreMaster: running generators' },
    () => runGenerators(deps.engine, folder, indexes, prepare),
  )
  const report = summariseRun(result)
  for (const line of report.lines) {
    deps.output.appendLine(line)
  }
  if (report.hasProblems) {
    deps.output.show(true)
    await vscode.window.showWarningMessage(`${report.headline} Some need a look; see the LoreMaster output.`)
  } else {
    await vscode.window.showInformationMessage(report.headline)
  }

  return result
}

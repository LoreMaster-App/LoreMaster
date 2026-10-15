import { SETTINGS_READ_METHOD, type SettingsReadResult } from '../engine-protocol'
import type { FileStore } from '../agent-creation'
import type { CiProvider } from './ci-provider.config'
import type { CiNeeds, CiPipelineFile } from './ci-pipeline.contract'
import { planCiNeeds } from './plan-ci-needs.policy'
import { renderAzurePipelines } from './render-azure-pipelines.mapper'
import { renderCiGuide } from './render-ci-guide.mapper'
import { renderGitHubActions } from './render-github-actions.mapper'

/** The minimal engine surface the session needs. */
export interface CiSetupEngine {
  request<R> (method: string, params?: unknown): Promise<R>
}

export interface CiSetupDeps {
  engine:        CiSetupEngine
  files:         FileStore
  workspaceRoot: string
  /** The extension's version; the pipeline downloads the engine of the same version. */
  version:       string
}

/** What the session found out about the workspace, before a CI is chosen. */
export type CiSetupPlan = { kind: 'no-outputs' } | { kind: 'ready', needs: CiNeeds }

/** The settings file CI would read; an output set there is available to a pipeline. */
const FOLDER_SETTINGS = '.vscode/settings.json'

/**
 * Reads the settings the way the editor and the CLI layer them, and works out what a pipeline
 * has to do for them. A workspace without outputs has nothing to put in CI yet.
 */
export async function planCiSetup (deps: CiSetupDeps): Promise<CiSetupPlan> {
  const read = await deps.engine.request<SettingsReadResult>(SETTINGS_READ_METHOD, { workspaceRoot: deps.workspaceRoot })
  if (read.settings.outputs.length === 0) {
    return { kind: 'no-outputs' }
  }

  const folderSettings = await deps.files.read(FOLDER_SETTINGS)
  const settingsInRepo = folderSettings?.includes('"loreMaster.outputs"') ?? false

  return { kind: 'ready', needs: planCiNeeds({ settings: read.settings, version: deps.version, settingsInRepo }) }
}

export interface CiSetupGuide {
  guide:    string
  pipeline: CiPipelineFile
}

/** The pipeline file and the guide for one CI. */
export function buildCiSetupGuide (provider: CiProvider, needs: CiNeeds): CiSetupGuide {
  const content = provider.id === 'github-actions' ? renderGitHubActions(needs) : renderAzurePipelines(needs)
  const pipeline: CiPipelineFile = { path: provider.path, content }

  return { guide: renderCiGuide(provider, needs, pipeline), pipeline }
}

export type WritePipelineOutcome = 'created' | 'updated' | 'unchanged' | 'skipped'

/**
 * Writes the pipeline file the command owns. A file that is already there with other content is
 * only replaced when the user says so; an existing pipeline of theirs is never touched, because
 * the command writes its own file.
 */
export async function writePipeline (
  files: FileStore,
  pipeline: CiPipelineFile,
  confirmReplace: (path: string) => Promise<boolean>,
): Promise<WritePipelineOutcome> {
  const existing = await files.read(pipeline.path)
  if (existing === pipeline.content) {
    return 'unchanged'
  }
  if (existing !== undefined && !await confirmReplace(pipeline.path)) {
    return 'skipped'
  }
  await files.write(pipeline.path, pipeline.content)

  return existing === undefined ? 'created' : 'updated'
}

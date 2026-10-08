import { AGENT_INSTRUCTIONS_METHOD, type AgentInstructionsResult } from '../engine-protocol'
import type { AgentTarget } from './agent-target.config'
import { planAgentWrite } from './plan-agent-write.policy'

/** The minimal engine surface the command needs. */
export interface AgentEngine {
  request<R> (method: string, params?: unknown): Promise<R>
}

/** Files of the workspace, by workspace-relative path. */
export interface FileStore {
  /** The file's text, or undefined when there is no such file. */
  read (relativePath: string): Promise<string | undefined>
  write (relativePath: string, content: string): Promise<void>
}

/** What happened to one target. */
export interface TargetOutcome {
  target:  AgentTarget
  outcome: 'created' | 'updated' | 'unchanged' | 'appended' | 'skipped'
}

export interface CreateAgentResult {
  outcomes:  TargetOutcome[]
  /** False when the workspace has no .lore-master.yaml yet, so the instructions are generic. */
  hasConfig: boolean
}

export interface CreateAgentDeps {
  engine:        AgentEngine
  files:         FileStore
  workspaceRoot: string
  /** Asked before a file the command did not write is replaced; false leaves it alone. */
  confirmOverwrite (relativePath: string): Promise<boolean>
}

/**
 * Writes the chosen targets. The instructions come from the engine, composed from the live
 * nesting rules and this workspace's settings, so the agent teaches what the sync enforces.
 * A file the command wrote is updated in place, a block in a shared file is replaced or
 * appended, and a file it did not write is only replaced when the user says so.
 */
export async function createAgent (deps: CreateAgentDeps, targets: readonly AgentTarget[]): Promise<CreateAgentResult> {
  const { engine, workspaceRoot } = deps
  const composed = await engine.request<AgentInstructionsResult>(AGENT_INSTRUCTIONS_METHOD, { workspaceRoot })

  const outcomes: TargetOutcome[] = []
  for (const target of targets) {
    outcomes.push({ target, outcome: await writeTarget(deps, target, target.render(composed.instructions)) })
  }

  return { outcomes, hasConfig: composed.hasConfig }
}

/** Writes one target as its plan says, asking first when it would replace a file that is not ours. */
async function writeTarget (deps: CreateAgentDeps, target: AgentTarget, rendered: string): Promise<TargetOutcome['outcome']> {
  const plan = planAgentWrite(target, await deps.files.read(target.path), rendered)

  if (plan.action === 'unchanged') {
    return 'unchanged'
  }
  if (plan.action === 'conflict') {
    if (!await deps.confirmOverwrite(target.path)) {
      return 'skipped'
    }
    await deps.files.write(target.path, rendered)

    return 'updated'
  }

  await deps.files.write(target.path, plan.content ?? rendered)
  if (plan.action === 'create') {
    return 'created'
  }

  return plan.action === 'append' ? 'appended' : 'updated'
}

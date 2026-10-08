import * as vscode from 'vscode'
import type { Generator, GeneratorsRunResult } from '../engine-protocol'
import { generatorTypeOf } from './generator-type.config'
import { type GeneratorsViewEngine, readGeneratorSettings } from './generators.use-case'

/** The view id contributed in package.json (inside the LoreMaster view container). */
export const GENERATORS_VIEW_ID = 'loreMaster.generators'

/** One generator of .lore-master.yaml, with its position in the list. */
export interface GeneratorNode {
  index:     number
  generator: Generator
}

/**
 * The LoreMaster "Generators" sidebar view: one row per configured generator, with what it
 * reads and where its pages go, and after a run what it did. Read live from .lore-master.yaml,
 * so it shows the same list the engine will run.
 */
export class GeneratorsViewProvider implements vscode.TreeDataProvider<GeneratorNode> {
  private readonly changed = new vscode.EventEmitter<GeneratorNode | undefined>()
  private lastRun = new Map<number, string>()

  readonly onDidChangeTreeData = this.changed.event

  constructor (private readonly engine: GeneratorsViewEngine) {}

  /** The configuration changed: re-read it and forget the last run, which was of the old one. */
  refresh (): void {
    this.lastRun = new Map()
    this.changed.fire(undefined)
  }

  /** Shows what a run did on the rows it ran, until the configuration next changes. */
  recordRun (result: GeneratorsRunResult): void {
    for (const run of result.runs) {
      this.lastRun.set(run.index, run.error ? `failed: ${run.error}` : `${run.written?.length ?? 0} written, ${run.unchanged?.length ?? 0} unchanged, ${run.removed?.length ?? 0} removed`)
    }
    this.changed.fire(undefined)
  }

  async getChildren (node?: GeneratorNode): Promise<GeneratorNode[]> {
    if (node) {
      return []
    }
    const folder = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath
    if (!folder) {
      return []
    }
    try {
      const { generators } = await readGeneratorSettings(this.engine, folder)

      return generators.map((generator, index) => ({ index, generator }))
    } catch {
      return []
    }
  }

  getTreeItem (node: GeneratorNode): vscode.TreeItem {
    const type = generatorTypeOf(node.generator.type)
    const item = new vscode.TreeItem(type.label, vscode.TreeItemCollapsibleState.None)
    const ran = this.lastRun.get(node.index)
    item.id = `generator:${node.index}`
    item.description = ran === undefined ? `→ ${node.generator.output}` : `→ ${node.generator.output} · ${ran}`
    item.iconPath = new vscode.ThemeIcon(type.icon)
    item.contextValue = 'loreMasterGenerator'
    item.tooltip = [
      type.description,
      `Writes to: ${node.generator.output}`,
      `Reads: ${node.generator.input && node.generator.input.length > 0 ? node.generator.input.join(', ') : 'the default'}`,
      ...(node.generator.title ? [`Index page title: ${node.generator.title}`] : []),
      ...(ran === undefined ? [] : [`Last run: ${ran}`]),
    ].join('\n')

    return item
  }
}

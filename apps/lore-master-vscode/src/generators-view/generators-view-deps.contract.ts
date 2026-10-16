import type * as vscode from 'vscode'
import type { EngineClient } from '../engine-process'
import type { GeneratorsViewProvider } from './generators-view.client'

/** What the Generators view's commands (and the generator form's) are given at registration. */
export interface GeneratorsViewDeps {
  engine:   EngineClient
  output:   vscode.OutputChannel
  provider: GeneratorsViewProvider
}

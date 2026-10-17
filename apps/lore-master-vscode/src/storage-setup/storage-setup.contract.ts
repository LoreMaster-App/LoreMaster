import type { EngineClient } from '../engine-process'
import type { ConfigDescription, FieldDescriptor, PlatformDescriptor } from '../engine-protocol'
import type { ConnectionMeta, ConnectionStore } from '../secret-storage'
import type { SyncTargetUI } from '../sync-target'

/** What configuring a storage that needs a sign-in asks of the editor: the target pickers, plus
 *  choosing a connection or being told there is none. */
export interface ConnectedSetupUI extends SyncTargetUI {
  pickConnection (connections: ConnectionMeta[]): Promise<ConnectionMeta | undefined>
  noConnections (): Promise<void>
  error (message: string): Promise<void>
}

/** Asks for one setting, as its description says: a choice is picked, anything else typed.
 *  `initial` is a value worth offering (the chosen parent page's title for the title prefix).
 *  Undefined means the user cancelled. */
export interface FieldPromptUI {
  promptField (field: FieldDescriptor, initial?: string): Promise<string | undefined>
}

/** Everything the first-run setup asks of the editor. */
export interface StorageSetupUI extends ConnectedSetupUI, FieldPromptUI {
  /** Which of the engine's platforms to set up; undefined when the user cancels. */
  pickStorageTypes (platforms: PlatformDescriptor[]): Promise<string[] | undefined>
}

/** What the first-run setup needs. `addConnection`, when given, lets a platform that needs a
 *  sign-in start the Add Connection wizard inline when there is no connection yet. */
export interface StorageSetupDeps {
  engine:         EngineClient
  connections:    ConnectionStore
  ui:             StorageSetupUI
  description:    ConfigDescription
  addConnection?: () => Promise<ConnectionMeta | undefined>
}

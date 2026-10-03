/** Where a workspace folder syncs: the space, the parent page, and the title prefix. */
export interface SyncTarget {
  space:        string
  parentPageId: string
  parentTitle:  string
  titlePrefix:  string
}

/** The chosen target for each folder, for this editor session. The durable copy lives in
 *  the folder's `.lore-master.yaml` (the engine writes it); this spares re-asking within
 *  a session and answers "has this folder been set up yet?". */
export interface TargetStore {
  get (folder: string): SyncTarget | undefined
  set (folder: string, target: SyncTarget): void
}

export function createTargetStore (): TargetStore {
  const byFolder = new Map<string, SyncTarget>()

  return {
    get: folder => byFolder.get(folder),
    set: (folder, target) => { byFolder.set(folder, target) },
  }
}

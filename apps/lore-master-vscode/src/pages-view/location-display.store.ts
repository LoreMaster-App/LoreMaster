import type { DisplayOptions } from './page-tree.contract'

/** The part of VS Code's Memento the store needs. */
export interface DisplayMemento {
  get<T> (key: string): T | undefined
  update (key: string, value: unknown): PromiseLike<void>
}

const KEY = 'loreMaster.locationDisplay'

/**
 * How each location of the Sync view (Local, and every storage) is shown: whether its pages are named by title or file name,
 * arranged as a tree, a flat list or the repository, and whether the files it leaves out are faded or hidden. A location remembers
 * what was chosen on its own row; one never chosen follows the settings.
 */
export class LocationDisplayStore {
  constructor (private readonly memento: DisplayMemento, private readonly defaults: () => DisplayOptions) {}

  private chosen (): Record<string, Partial<DisplayOptions>> {
    return { ...this.memento.get<Record<string, Partial<DisplayOptions>>>(KEY) }
  }

  /** What a location shows now: its own choices over the defaults. */
  optionsOf (index: number): DisplayOptions {
    return { ...this.defaults(), ...this.chosen()[String(index)] }
  }

  /** Remembers a choice for one location. */
  async set (index: number, change: Partial<DisplayOptions>): Promise<void> {
    const all = this.chosen()
    all[String(index)] = { ...all[String(index)], ...change }
    await this.memento.update(KEY, all)
  }

  /** Forgets every location's own choices, so all follow the settings again. */
  async reset (): Promise<void> {
    await this.memento.update(KEY, undefined)
  }
}

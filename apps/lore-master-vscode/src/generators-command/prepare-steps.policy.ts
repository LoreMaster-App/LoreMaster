/** The `loreMaster.runPrepareSteps` setting: whether generators' prepare commands run. */
export type PrepareSetting = 'ask' | 'always' | 'never'

/** What to do with the prepare commands of the generators about to run. */
export type PrepareDecision =
  | { kind: 'skip'; reason: 'none' | 'never' | 'untrusted' } |
  { kind: 'run' } |
  { kind: 'ask'; commands: string[] }

/** The setting's value, `ask` when it is missing or not one of the three. */
export function parsePrepareSetting (value: unknown): PrepareSetting {
  return value === 'always' || value === 'never' ? value : 'ask'
}

/**
 * Decides, without doing anything, whether the prepare commands run. A command is code the
 * settings file carries, so nothing runs in a workspace the user has not trusted, `never` always
 * wins, and with `ask` a command the user already approved this session does not ask again.
 */
export function decidePrepare (options: {
  commands: readonly string[]
  setting:  PrepareSetting
  trusted:  boolean
  approved: ReadonlySet<string>
}): PrepareDecision {
  const { commands, setting, trusted, approved } = options
  if (commands.length === 0) {
    return { kind: 'skip', reason: 'none' }
  }
  if (setting === 'never') {
    return { kind: 'skip', reason: 'never' }
  }
  if (!trusted) {
    return { kind: 'skip', reason: 'untrusted' }
  }
  if (setting === 'always' || commands.every(command => approved.has(command))) {
    return { kind: 'run' }
  }

  return { kind: 'ask', commands: [...commands] }
}

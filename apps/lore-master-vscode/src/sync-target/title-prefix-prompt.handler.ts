/** The title-prefix prompt. Reused as the QuickPick/input validator too. */
export interface TitlePrefixUI {
  promptTitlePrefix (defaultValue: string, validate: (value: string) => string | undefined): Promise<string | undefined>
}

/** Rejects a prefix that is empty or carries a colon (titles are `<prefix>: <H1>`, so a
 *  colon in the prefix would be ambiguous). Returns the reason, or undefined when valid. */
export function validateTitlePrefix (value: string): string | undefined {
  const trimmed = value.trim()
  if (trimmed === '') {
    return 'The title prefix cannot be empty.'
  }
  if (trimmed.includes(':')) {
    return 'The title prefix cannot contain a colon.'
  }

  return undefined
}

/**
 * The title prefix for a target, asked **only when there isn't one already** (a stored
 * value or one in `.lore-master.yaml`). The default offered is the chosen parent page's
 * title. Returns the existing prefix untouched, the entered one, or undefined if the
 * prompt was cancelled.
 */
export async function resolveTitlePrefix (deps: { existing: string; parentTitle: string; ui: TitlePrefixUI }): Promise<string | undefined> {
  if (deps.existing.trim() !== '') {
    return deps.existing
  }

  return deps.ui.promptTitlePrefix(deps.parentTitle, validateTitlePrefix)
}

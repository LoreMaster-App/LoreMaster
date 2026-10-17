import type { ConfigDescription, FieldDescriptor, Output, PlatformDescriptor } from '../engine-protocol'

/** The platform an output uses, or undefined for one this extension's engine does not describe. */
export function platformOf (description: ConfigDescription, output: Pick<Output, 'platform'>): PlatformDescriptor | undefined {
  return description.platforms.find(platform => platform.id === output.platform)
}

/** True for an output published as a whole (a site or wiki) rather than synced page by page. */
export function isSiteOutput (description: ConfigDescription, output: Pick<Output, 'platform'>): boolean {
  return platformOf(description, output)?.delivery === 'site'
}

/** True for an output synced page by page (a page tree, rollback, watch). */
export function isPagesOutput (description: ConfigDescription, output: Pick<Output, 'platform'>): boolean {
  return platformOf(description, output)?.delivery === 'pages'
}

/** The fields of a storage in the order to show them: the platform's own, then the ones every storage has. */
export function fieldsOf (description: ConfigDescription, platform: PlatformDescriptor): FieldDescriptor[] {
  return [...platform.fields, ...description.outputFields]
}

/** Every setting key any storage can hold, so what does not belong to the chosen platform can be cleared. */
export function allOutputKeys (description: ConfigDescription): string[] {
  const keys = new Set<string>(description.outputFields.map(field => field.key))
  for (const platform of description.platforms) {
    for (const field of platform.fields) {
      keys.add(field.key)
    }
  }

  return [...keys]
}

/** The codicon a platform's icon hint maps to: the hint is neutral, the icon set is the editor's. */
export function codiconFor (iconHint: string): string {
  switch (iconHint) {
    case 'site': {
      return 'globe'
    }
    case 'wiki': {
      return 'notebook'
    }
    default: {
      return 'book'
    }
  }
}

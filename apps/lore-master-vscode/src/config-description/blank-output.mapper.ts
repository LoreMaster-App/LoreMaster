import type { ConfigDescription, Output, PlatformDescriptor } from '../engine-protocol'
import { allOutputKeys } from './platform-lookup.policy'

/**
 * A new storage of the platform: every setting of every platform present and empty, a choice on its
 * default, and the whole workspace's Markdown as content. The settings the platform does not use are
 * left empty for the engine to ignore.
 */
export function blankOutput (description: ConfigDescription, platform: PlatformDescriptor): Output {
  const output: Record<string, unknown> = { platform: platform.id, content: [{ type: 'markdown', roots: ['.'], template: 'default' }] }
  const fields = [...description.outputFields, ...description.platforms.flatMap(each => each.fields)]
  for (const key of allOutputKeys(description)) {
    const field = fields.find(each => each.key === key)
    if (field?.type === 'list' || field?.type === 'number') {
      continue
    }
    output[key] = field?.type === 'choice' ? field.default ?? '' : ''
  }
  if (output.name === '') {
    delete output.name
  }

  return output as unknown as Output
}

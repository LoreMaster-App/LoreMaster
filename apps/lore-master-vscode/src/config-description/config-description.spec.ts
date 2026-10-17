import { CONFIG_DESCRIBE_METHOD } from '../engine-protocol'
import { blankOutput } from './blank-output.mapper'
import { describeConfig } from './describe-config.use-case'
import { allOutputKeys, codiconFor, fieldsOf, isPagesOutput, isSiteOutput, platformOf } from './platform-lookup.policy'
import { describedConfig } from '../../test/config-description/config-description.fixture'

const description = describedConfig()

describe('describeConfig', () => {
  it('asks the engine once per client and remembers the answer', async () => {
    const asked: string[] = []
    const engine = {
      request: <R>(method: string): Promise<R> => {
        asked.push(method)

        return Promise.resolve(description as R)
      },
    }

    const first = await describeConfig(engine)
    const second = await describeConfig(engine)

    expect(first).toBe(second)
    expect(asked).toEqual([CONFIG_DESCRIBE_METHOD])
  })

  it('does not remember a failure: the next ask tries again', async () => {
    let calls = 0
    const engine = {
      request: <R>(): Promise<R> => {
        calls += 1

        return calls === 1 ? Promise.reject(new Error('engine is starting')) : Promise.resolve(description as R)
      },
    }

    await expect(describeConfig(engine)).rejects.toThrow('engine is starting')
    await expect(describeConfig(engine)).resolves.toBe(description)
  })
})

describe('the platform lookups', () => {
  it('tell a site from a page-by-page storage by what the description says', () => {
    expect(isSiteOutput(description, { platform: 'github-pages' })).toBe(true)
    expect(isSiteOutput(description, { platform: 'confluence' })).toBe(false)
    expect(isPagesOutput(description, { platform: 'confluence' })).toBe(true)
    expect(isPagesOutput(description, { platform: 'unheard-of' })).toBe(false)
    expect(platformOf(description, { platform: 'unheard-of' })).toBeUndefined()
  })

  it('list a storage\'s fields with the shared ones after its own', () => {
    const confluence = description.platforms[0]
    const keys = fieldsOf(description, confluence).map(field => field.key)

    expect(keys.slice(0, 2)).toEqual(['baseUrl', 'space'])
    expect(keys).toContain('name')
  })

  it('know every setting any storage can hold', () => {
    expect(allOutputKeys(description)).toEqual(expect.arrayContaining(['baseUrl', 'repo', 'branch', 'path', 'name', 'include', 'exclude']))
  })

  it('map an icon hint to a codicon and fall back to a book', () => {
    expect([codiconFor('site'), codiconFor('wiki'), codiconFor('confluence'), codiconFor('')]).toEqual(['globe', 'notebook', 'book', 'book'])
  })
})

describe('blankOutput', () => {
  it('starts a storage empty with every choice on its default and the whole workspace as content', () => {
    const output = blankOutput(description, description.platforms[0])

    expect(output).toMatchObject({ platform: 'confluence', baseUrl: '', space: '', direction: 'to-platform', mermaidMode: 'image', linkMode: 'title', titleCollision: 'fail' })
    expect(output.content).toEqual([{ type: 'markdown', roots: ['.'], template: 'default' }])
    expect(output.name).toBeUndefined()
  })
})

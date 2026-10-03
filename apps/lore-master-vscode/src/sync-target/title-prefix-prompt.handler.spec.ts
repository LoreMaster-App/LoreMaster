import { resolveTitlePrefix, type TitlePrefixUI, validateTitlePrefix } from './title-prefix-prompt.handler'

describe('validateTitlePrefix', () => {
  it('rejects empty and colon, accepts otherwise', () => {
    expect(validateTitlePrefix('')).toBeDefined()
    expect(validateTitlePrefix(' '.repeat(3))).toBeDefined()
    expect(validateTitlePrefix('ENG: x')).toBeDefined()
    expect(validateTitlePrefix('ENG')).toBeUndefined()
  })
})

describe('resolveTitlePrefix', () => {
  it('does not prompt when a prefix already exists', async () => {
    let prompted = false
    const ui: TitlePrefixUI = {
      promptTitlePrefix: () => {
        prompted = true

        return Promise.resolve('x')
      },
    }

    const prefix = await resolveTitlePrefix({ existing: 'ENG', parentTitle: 'Engineering', ui })

    expect(prefix).toBe('ENG')
    expect(prompted).toBe(false)
  })

  it('prompts with the parent title as the default when there is no prefix yet', async () => {
    let seenDefault = ''
    const ui: TitlePrefixUI = {
      promptTitlePrefix: (value) => {
        seenDefault = value

        return Promise.resolve('Engineering')
      },
    }

    const prefix = await resolveTitlePrefix({ existing: '', parentTitle: 'Engineering', ui })

    expect(seenDefault).toBe('Engineering')
    expect(prefix).toBe('Engineering')
  })

  it('returns undefined when the prompt is cancelled', async () => {
    const ui: TitlePrefixUI = { promptTitlePrefix: () => Promise.resolve(undefined) }

    expect(await resolveTitlePrefix({ existing: '  ', parentTitle: 'Eng', ui })).toBeUndefined()
  })
})

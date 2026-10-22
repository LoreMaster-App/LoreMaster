import { rowState, rowStateLines } from './row-state.policy'

describe('rowState', () => {
  it('says where each of the three choices stands, in words', () => {
    expect(rowState({ label: 'title', view: 'storage', excluded: 'faded' })).toBe('tree · page titles · ignored faded')
    expect(rowState({ label: 'fileName', view: 'flat', excluded: 'hidden' })).toBe('flat list · file names · ignored hidden')
    expect(rowState({ label: 'title', view: 'repo', excluded: 'faded' })).toBe('repository · page titles · ignored faded')
  })

  it('names the ignored files for what the row lists, one choice to a line', () => {
    expect(rowStateLines({ label: 'title', view: 'repo', excluded: 'hidden' }, 'Files git ignores')).toEqual([
      'Arranged as: repository', 'Pages named by: page titles', 'Files git ignores: hidden',
    ])
  })
})

import { parseList } from './parse-list.algorithm'

describe('parseList', () => {
  it('splits on commas and new lines, trims and drops empty entries', () => {
    expect(parseList(' docs/, guides ,\n\n internal/ ,')).toEqual(['docs/', 'guides', 'internal/'])
  })

  it('gives an empty list for an empty string', () => {
    expect(parseList('  ')).toEqual([])
  })
})

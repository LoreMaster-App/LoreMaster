import { toggleTaskInLine } from './toggle-task-line.algorithm'

describe('toggleTaskInLine', () => {
  it('flips an open task to done and a done one back, keeping the rest of the line', () => {
    expect(toggleTaskInLine('- [ ] write the docs')).toBe('- [x] write the docs')
    expect(toggleTaskInLine('- [x] write the docs')).toBe('- [ ] write the docs')
    expect(toggleTaskInLine('- [X] shout')).toBe('- [ ] shout')
  })

  it('reads every kind of bullet, a number, nesting and a quote', () => {
    expect(toggleTaskInLine('  * [ ] nested')).toBe('  * [x] nested')
    expect(toggleTaskInLine('+ [ ] plus')).toBe('+ [x] plus')
    expect(toggleTaskInLine('12. [ ] numbered')).toBe('12. [x] numbered')
    expect(toggleTaskInLine('3) [x] paren')).toBe('3) [ ] paren')
    expect(toggleTaskInLine('> - [ ] quoted')).toBe('> - [x] quoted')
  })

  it('changes nothing that is not a task list item', () => {
    for (const line of ['plain text', '- item', '-[ ] no space', '- [] empty', '- [y] other', '[ ] no bullet', '', '1.[ ] no space']) {
      expect(toggleTaskInLine(line)).toBeUndefined()
    }
  })
})

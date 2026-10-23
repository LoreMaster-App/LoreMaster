import { addTaskLists } from './add-task-lists.algorithm'
import type { TaskMarkdownIt, TaskRenderRule, TaskState, TaskToken } from './task-list.contract'

class FakeToken implements TaskToken {
  content = ''
  map:      [number, number] | null = null
  children: TaskToken[] | null = null
  meta:     unknown = null
  classes:  string[] = []
  constructor (readonly type: string) {}

  attrJoin (name: string, value: string): void {
    if (name === 'class') { this.classes.push(value) }
  }
}

const text = (content: string): FakeToken => Object.assign(new FakeToken('text'), { content })

/** The tokens of a task list as markdown-it makes them: list, then per item the item (with its line), paragraph and inline. */
function listOf (...items: { line: number, content: string }[]): FakeToken[] {
  return [
    new FakeToken('bullet_list_open'),
    ...items.flatMap(({ line, content }) => {
      const item = Object.assign(new FakeToken('list_item_open'), { map: [line, line + 1] as [number, number] })
      const inline = Object.assign(new FakeToken('inline'), { content, children: [text(content)] })

      return [item, new FakeToken('paragraph_open'), inline, new FakeToken('paragraph_close'), new FakeToken('list_item_close')]
    }),
    new FakeToken('bullet_list_close'),
  ]
}

const noRule = (): void => {}

function plugin (tokens: FakeToken[]): { rules: Record<string, TaskRenderRule | undefined>, tokens: FakeToken[] } {
  let rule: (state: TaskState) => void = noRule
  const rules: Record<string, TaskRenderRule | undefined> = {}
  const md: TaskMarkdownIt = { core: { ruler: { push: (_name, made) => { rule = made } } }, renderer: { rules } }
  addTaskLists(md)
  rule({ tokens, Token: FakeToken })

  return { rules, tokens }
}

const inlines = (tokens: FakeToken[]): FakeToken[] => tokens.filter(each => each.type === 'inline')
const checkboxOf = (tokens: FakeToken[], at: number): TaskToken => (inlines(tokens)[at].children ?? [])[0]

describe('addTaskLists', () => {
  it('turns the marker that opens an item into a checkbox that remembers its state and line, and marks the item and the list', () => {
    const { tokens } = plugin(listOf({ line: 4, content: '[ ] open one' }, { line: 5, content: '[x] done one' }))

    expect(checkboxOf(tokens, 0)).toMatchObject({ type: 'loremaster_task', meta: { checked: false, line: 4 } })
    expect(checkboxOf(tokens, 1)).toMatchObject({ type: 'loremaster_task', meta: { checked: true, line: 5 } })
    expect(tokens[0].classes).toEqual(['contains-task-list'])
    expect(tokens[1].classes).toEqual(['task-list-item'])
    expect(tokens[6].classes).toEqual(['task-list-item', 'task-list-item-done'])
    expect(inlines(tokens)[0].children?.[1].content).toBe('open one')
  })

  it('leaves an ordinary item and a marker with no space after it alone', () => {
    const { tokens } = plugin(listOf({ line: 0, content: 'plain' }, { line: 1, content: '[x]nospace' }))

    expect(tokens.some(each => each.classes.length > 0)).toBe(false)
    expect(inlines(tokens).every(each => each.children?.length === 1)).toBe(true)
  })

  it('draws a link of the action scheme when the page is rendered for a file', () => {
    const { rules, tokens } = plugin(listOf({ line: 7, content: '[ ] go' }))
    const html = rules.loremaster_task?.([checkboxOf(tokens, 0)], 0, {}, { currentDocument: { toString: () => 'file:///repo/a.md' } }) ?? ''

    expect(html).toContain('href="loremaster-action:/toggle-task?document=file%3A%2F%2F%2Frepo%2Fa.md&amp;line=7"')
    expect(html).toContain('role="checkbox"')
    expect(html).toContain('aria-checked="false"')
  })

  it('draws a plain disabled checkbox when no file is known, and says a done item is done', () => {
    const { rules, tokens } = plugin(listOf({ line: 0, content: '[x] go' }))
    const html = rules.loremaster_task?.([checkboxOf(tokens, 0)], 0, {}, {}) ?? ''

    expect(html).toBe('<input class="loremaster-task-static" type="checkbox" disabled checked>')
  })
})

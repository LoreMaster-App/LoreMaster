import { previewActionLink } from '../preview-actions'
import type { TaskMarkdownIt, TaskMeta, TaskRenderRule, TaskState, TaskToken } from './task-list.contract'

/** The token type of a task's checkbox. */
const TASK_TOKEN = 'loremaster_task'

/** The action the checkbox asks for; the extension answers it by editing the file. */
export const TOGGLE_TASK_ACTION = 'toggle-task'

const MARKER = /^\[([ x])\](?=[\t ]|$)[\t ]?/i

const escapeHtml = (text: string): string => text.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;')

/** The list token an item at `index` belongs to: the nearest list opened before it. */
function listOpeningBefore (tokens: TaskToken[], index: number): TaskToken | undefined {
  for (let back = index - 1; back >= 0; back--) {
    if (tokens[back].type === 'bullet_list_open' || tokens[back].type === 'ordered_list_open') { return tokens[back] }
  }

  return undefined
}

/** Turns the `[ ]` or `[x]` that opens a list item into a checkbox token, and marks the item and its list (once). */
function markTasks (state: TaskState): void {
  const { tokens } = state
  const marked = new Set<TaskToken>()
  for (let index = 0; index + 2 < tokens.length; index++) {
    const [item, paragraph, inline] = [tokens[index], tokens[index + 1], tokens[index + 2]]
    if (item.type !== 'list_item_open' || paragraph.type !== 'paragraph_open' || inline.type !== 'inline' || item.map === null) { continue }
    const first = inline.children?.[0]
    const match = first?.type === 'text' ? MARKER.exec(first.content) : null
    if (first === undefined || match === null || inline.children === null) { continue }

    first.content = first.content.slice(match[0].length)
    inline.content = inline.content.slice(match[0].length)
    const checked = match[1] !== ' '
    const checkbox = new state.Token(TASK_TOKEN, '', 0)
    checkbox.meta = { checked, line: item.map[0] } satisfies TaskMeta
    inline.children.unshift(checkbox)
    item.attrJoin('class', 'task-list-item')
    if (checked) { item.attrJoin('class', 'task-list-item-done') }
    const list = listOpeningBefore(tokens, index)
    if (list !== undefined && !marked.has(list)) {
      marked.add(list)
      list.attrJoin('class', 'contains-task-list')
    }
  }
}

/** A checkbox that only shows the state: a render that does not know the file cannot offer to change it. */
function staticCheckbox (meta: TaskMeta): string {
  return `<input class="loremaster-task-static" type="checkbox" disabled${meta.checked ? ' checked' : ''}>`
}

/** Writes a task's checkbox as a link of LoreMaster's action scheme, or as a plain checkbox when no file is open. */
const drawTask: TaskRenderRule = (tokens, index, _options, env) => {
  const meta = tokens[index].meta as TaskMeta
  const document = env.currentDocument
  if (document === undefined) { return staticCheckbox(meta) }
  const link = previewActionLink(TOGGLE_TASK_ACTION, { document: document.toString(), line: String(meta.line) })
  const title = meta.checked ? 'Mark as not done' : 'Mark as done'

  return `<a class="loremaster-task${meta.checked ? ' loremaster-task-done' : ''}" href="${escapeHtml(link)}" role="checkbox" aria-checked="${meta.checked}" title="${title}"></a>`
}

/**
 * Teaches the Markdown preview task lists, which VS Code does not draw: `- [ ] item` and `- [x] item` get a checkbox, and clicking
 * it asks LoreMaster to flip the `[ ]` in the file. The checkbox is a link of LoreMaster's action scheme (the preview runs no
 * commands), written when the page is rendered because only then is the open file known.
 */
export function addTaskLists<Md extends TaskMarkdownIt> (md: Md): Md {
  md.core.ruler.push(TASK_TOKEN, markTasks)
  md.renderer.rules[TASK_TOKEN] = drawTask

  return md
}

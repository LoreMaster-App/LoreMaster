/** How a tag is drawn in the Notes view: a codicon and the theme colour it is drawn in. */
export interface TagStyle {
  icon:  string
  color: string
}

const STYLES: Record<string, TagStyle> = {
  TODO:  { icon: 'tasklist', color: 'charts.blue' },
  FIXME: { icon: 'wrench', color: 'charts.orange' },
  BUG:   { icon: 'bug', color: 'charts.red' },
  HACK:  { icon: 'flame', color: 'charts.yellow' },
  XXX:   { icon: 'alert', color: 'charts.red' },
  NOTE:  { icon: 'note', color: 'charts.green' },
  WARN:  { icon: 'warning', color: 'charts.yellow' },
  WHY:   { icon: 'question', color: 'charts.purple' },
}

/** What a tag with no style of its own looks like. */
const PLAIN: TagStyle = { icon: 'comment', color: 'foreground' }

/** The style of a tag. */
export function styleOfTag (tag: string): TagStyle {
  return STYLES[tag] ?? PLAIN
}

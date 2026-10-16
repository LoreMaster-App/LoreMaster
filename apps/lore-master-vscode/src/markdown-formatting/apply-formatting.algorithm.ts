import type { EditResult, TextSelection } from '../text-edits'
import { decreaseHeading, increaseHeading, toggleQuote, toggleTask } from './change-line-format.algorithm'
import { toggleCodeBlock } from './toggle-code-block.algorithm'
import { toggleInline } from './toggle-inline.algorithm'

/** Every formatting command there is. */
export type FormattingAction = 'bold' | 'italic' | 'strikethrough' | 'inlineCode' | 'codeBlock' | 'quote' | 'task' | 'increaseHeading' | 'decreaseHeading'

/** What one action does for one cursor, or nothing when it has nothing to change. */
export function applyFormatting (action: FormattingAction, lines: readonly string[], selection: TextSelection): EditResult | undefined {
  switch (action) {
    case 'bold': {
      return toggleInline(lines, selection, '**')
    }
    case 'italic': {
      return toggleInline(lines, selection, '*')
    }
    case 'strikethrough': {
      return toggleInline(lines, selection, '~~')
    }
    case 'inlineCode': {
      return toggleInline(lines, selection, '`')
    }
    case 'codeBlock': {
      return toggleCodeBlock(lines, selection)
    }
    case 'quote': {
      return toggleQuote(lines, selection)
    }
    case 'task': {
      return toggleTask(lines, selection)
    }
    case 'increaseHeading': {
      return increaseHeading(lines, selection)
    }
    case 'decreaseHeading': {
      return decreaseHeading(lines, selection)
    }
  }
}

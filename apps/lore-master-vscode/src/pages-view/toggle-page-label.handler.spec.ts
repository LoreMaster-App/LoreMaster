import * as vscode from 'vscode'
import { PAGE_LABEL_SETTING, togglePageLabel } from './toggle-page-label.handler'

const stored = (): unknown => vscode.workspace.getConfiguration('loreMaster').get(PAGE_LABEL_SETTING)

describe('togglePageLabel', () => {
  it('switches between the content title and the file name, and remembers the choice', async () => {
    expect(stored()).toBeUndefined()

    await togglePageLabel()
    expect(stored()).toBe('fileName')

    await togglePageLabel()
    expect(stored()).toBe('title')
  })
})

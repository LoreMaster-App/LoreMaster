import * as vscode from 'vscode'
import { describedConfig } from '../../test/config-description/config-description.fixture'
import { readResultOf } from '../../test/config-description/engine-answers.fixture'
import { CONFIG_DESCRIBE_METHOD, type Output, SETTINGS_READ_METHOD } from '../engine-protocol'
import { type StoragesEngine, StoragesViewProvider } from './storages-view.client'

const confluence: Output = {
  platform:       'confluence',
  baseUrl:        'https://x.atlassian.net/wiki',
  space:          'ENG',
  parentPageId:   '1',
  titlePrefix:    'ENG',
  direction:      'to-platform',
  content:        [{ type: 'markdown', roots: ['.'], template: 'default' }],
  mermaidMode:    'image',
  titleCollision: 'fail',
  linkMode:       'title',
}

const pages: Output = {
  platform:       'github-pages',
  baseUrl:        '',
  space:          '',
  parentPageId:   '',
  titlePrefix:    '',
  direction:      'to-platform',
  content:        [{ type: 'markdown', roots: ['docs'], template: 'default' }],
  mermaidMode:    '',
  titleCollision: '',
  linkMode:       '',
  repo:           'owner/name',
  branch:         'gh-pages',
}

const scaffold: Output = { ...confluence, baseUrl: '', space: '', parentPageId: '', titlePrefix: '' }

function engine (outputs: Output[]): StoragesEngine {
  return {
    request (method: string) {
      if (method === CONFIG_DESCRIBE_METHOD) {
        return Promise.resolve(describedConfig() as never)
      }

      return Promise.resolve((method === SETTINGS_READ_METHOD ? readResultOf({ version: 1, outputs }) : null) as never)
    },
  }
}

function setFolders (folders: { uri: { fsPath: string }; name: string }[] | undefined): void {
  (vscode.workspace as { workspaceFolders: unknown }).workspaceFolders = folders
}

describe('StoragesViewProvider', () => {
  beforeEach(() => { setFolders([{ uri: { fsPath: '/w' }, name: 'w' }]) })
  afterEach(() => { setFolders(undefined) })

  it('lists configured outputs by their settings index and hides the blank scaffold', async () => {
    const nodes = await new StoragesViewProvider(engine([confluence, pages, scaffold])).getChildren()

    expect(nodes.map(node => node.output.platform)).toEqual(['confluence', 'github-pages'])
    expect(nodes.map(node => node.index)).toEqual([0, 1])
  })

  it('renders a labelled, removable tree item per output', async () => {
    const [node] = await new StoragesViewProvider(engine([pages])).getChildren()
    const item = new StoragesViewProvider(engine([pages])).getTreeItem(node)

    expect(item.label).toContain('github-pages')
    expect(item.description).toBe('owner/name')
    expect((item.iconPath as { id: string }).id).toBe('globe')
    expect(item.contextValue).toBe('loreMasterStorage')
  })

  it('returns nothing when no folder is open', async () => {
    setFolders(undefined)

    expect(await new StoragesViewProvider(engine([confluence])).getChildren()).toEqual([])
  })
})

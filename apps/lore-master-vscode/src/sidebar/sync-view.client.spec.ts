import { SYNC_VIEW_ID, SyncViewProvider } from './sync-view.client'

describe('SyncViewProvider', () => {
  const provider = new SyncViewProvider()

  it('is the view id contributed in package.json', () => {
    expect(SYNC_VIEW_ID).toBe('loreMaster.sync')
  })

  it('lists the three sync actions at the root, each wired to a command', () => {
    const actions = provider.getChildren()

    expect(actions.map(action => action.label)).toEqual(['Add sync storage', 'Sync', 'Update config…'])
    expect(actions.map(action => action.command)).toEqual(['loreMaster.addStorage', 'loreMaster.sync', 'loreMaster.openConfig'])
  })

  it('has no children under an action (the tree is flat)', () => {
    expect(provider.getChildren(provider.getChildren()[0])).toEqual([])
  })

  it('turns an action into a clickable tree item', () => {
    const item = provider.getTreeItem(provider.getChildren()[1])

    expect(item.label).toBe('Sync')
    expect(item.command).toEqual({ command: 'loreMaster.sync', title: 'Sync' })
  })
})

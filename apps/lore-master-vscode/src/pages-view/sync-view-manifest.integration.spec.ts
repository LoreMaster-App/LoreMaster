import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { DISPLAY_COMMANDS, GENERATE_AND_SYNC_STORAGE_COMMAND, REFRESH_STORAGE_COMMAND, SYNC_STORAGE_COMMAND } from './location-commands.config'
import { PAGES_VIEW_ID } from './pages-view.client'
import { rowContext } from './row-context.policy'

interface Manifest {
  contributes: {
    commands: Array<{ command: string, title: string, icon?: string }>
    views:    Record<string, Array<{ id: string }>>
    menus:    Record<string, Array<{ command: string, when?: string, group?: string }>>
  }
}

const manifest = JSON.parse(readFileSync(join(__dirname, '../../package.json'), 'utf8')) as Manifest
const inline = manifest.contributes.menus['view/item/context'].filter(entry => entry.group?.startsWith('inline'))

/** Whether a menu entry's `when` shows for a row with this context, for the clause shapes the manifest uses. */
function shownFor (when: string | undefined, context: string): boolean {
  return (when ?? '').split(' && ').every(clause => {
    const negated = /^!\(viewItem =~ \/(.*)\/\)$/.exec(clause.trim())
    if (negated) {
      return !new RegExp(negated[1]).test(context)
    }
    const positive = /^viewItem =~ \/(.*)\/$/.exec(clause.trim())
    if (positive) {
      return new RegExp(positive[1]).test(context)
    }
    const exact = /^viewItem == (\w+)$/.exec(clause.trim())
    if (exact) {
      return context.split(' ', 1)[0] === exact[1]
    }

    // The view clause: only the Sync view's own rows are of interest.
    return clause.trim() === `view == ${PAGES_VIEW_ID}`
  })
}

/** The inline actions a row with this context shows, in order. */
function inlineFor (context: string): string[] {
  return inline.filter(entry => shownFor(entry.when, context)).map(entry => entry.command)
}

const nextView = (view: 'storage' | 'flat' | 'repo'): string[] => inlineFor(rowContext('loreMasterStorage', { label: 'title', view, excluded: 'faded' })).filter(command => /^loreMaster\.view/.test(command))

describe('the Sync view in the manifest', () => {
  it('is the one tree of locations beside the Generators view, with no Pages or Storages view', () => {
    const ids = manifest.contributes.views.loreMaster.map(view => view.id)

    expect(ids).toEqual([PAGES_VIEW_ID, 'loreMaster.generators'])
    expect(PAGES_VIEW_ID).toBe('loreMaster.sync')
  })

  it('has every command a location row runs', () => {
    const contributed = manifest.contributes.commands.map(command => command.command)

    for (const id of [...Object.keys(DISPLAY_COMMANDS), REFRESH_STORAGE_COMMAND, SYNC_STORAGE_COMMAND, GENERATE_AND_SYNC_STORAGE_COMMAND]) {
      expect(contributed).toContain(id)
    }
  })

  it('shows the actions of the whole view in its title, always', () => {
    const title = manifest.contributes.menus['view/title'].filter(entry => entry.when === `view == ${PAGES_VIEW_ID}`)

    expect(title.map(entry => entry.command)).toEqual([
      'loreMaster.addStorage', 'loreMaster.openConfig', 'loreMaster.refreshPages', 'loreMaster.sync', 'loreMaster.generateAndSync',
    ])
    expect(title.every(entry => manifest.contributes.commands.find(command => command.command === entry.command)?.icon)).toBe(true)
  })

  it('gives a storage row its own actions, and Local only the ones that are not about a platform', () => {
    const storage = rowContext('loreMasterStorage', { label: 'title', view: 'storage', excluded: 'faded' })
    const local = rowContext('loreMasterLocal', { label: 'title', view: 'storage', excluded: 'faded' })
    expect(inlineFor(storage)).toEqual([
      'loreMaster.checkRemote', 'loreMaster.showFileNames', 'loreMaster.viewFlat', 'loreMaster.hideIgnored',
      'loreMaster.refreshStorage', 'loreMaster.syncStorage', 'loreMaster.generateAndSyncStorage', 'loreMaster.editStorage', 'loreMaster.removeStorage',
    ])
    expect(inlineFor(local)).toEqual(['loreMaster.showFileNames', 'loreMaster.viewFlat', 'loreMaster.refreshStorage'])
  })

  it('offers the other choice for what a row shows now', () => {
    const flatTitles = rowContext('loreMasterStorage', { label: 'fileName', view: 'flat', excluded: 'hidden' })
    const shown = inlineFor(flatTitles)

    expect(shown).toEqual(expect.arrayContaining(['loreMaster.showPageTitles', 'loreMaster.viewRepository', 'loreMaster.showIgnored']))
    expect(shown).not.toEqual(expect.arrayContaining(['loreMaster.showFileNames']))
  })

  it('steps the view button through the three arrangements, so each is reached by clicking', () => {
    expect(nextView('storage')).toEqual(['loreMaster.viewFlat'])
    expect(nextView('flat')).toEqual(['loreMaster.viewRepository'])
    expect(nextView('repo')).toEqual(['loreMaster.viewTree'])
  })
})

import { migrateSettings } from './migrate-settings.use-case'

describe('migrateSettings', () => {
  it('reports each folder that was imported and asks the engine once per folder', async () => {
    const request = jest.fn()
      .mockResolvedValueOnce({ migrated: true, path: '/a/.vscode/settings.json' })
      .mockResolvedValueOnce({ migrated: false })

    const outcome = await migrateSettings({ engine: { request }, workspaceRoots: ['/a', '/b'] })

    expect(request).toHaveBeenCalledTimes(2)
    expect(request).toHaveBeenCalledWith('settings/migrate', { workspaceRoot: '/a' })
    expect(outcome).toEqual({ migrated: ['/a/.vscode/settings.json'], failures: [] })
  })

  it('keeps going after a folder fails and says what went wrong', async () => {
    const request = jest.fn()
      .mockRejectedValueOnce(new Error('outputs[0].platform is not one of confluence'))
      .mockResolvedValueOnce({ migrated: true, path: '/b/.vscode/settings.json' })

    const outcome = await migrateSettings({ engine: { request }, workspaceRoots: ['/a', '/b'] })

    expect(outcome.migrated).toEqual(['/b/.vscode/settings.json'])
    expect(outcome.failures).toEqual(['outputs[0].platform is not one of confluence'])
  })
})

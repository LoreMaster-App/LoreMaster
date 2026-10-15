import type { FileStore } from '../agent-creation'
import type { Settings } from '../engine-protocol'
import { CI_PROVIDERS } from './ci-provider.config'
import { buildCiSetupGuide, planCiSetup, writePipeline } from './set-up-ci.use-case'

function memoryFiles (initial: Record<string, string> = {}): FileStore & { files: Map<string, string> } {
  const files = new Map(Object.entries(initial))

  return {
    files,
    read:  path => Promise.resolve(files.get(path)),
    write: (path, content) => {
      files.set(path, content)

      return Promise.resolve()
    },
  }
}

function engineWith (settings: Settings) {
  return { request: () => Promise.resolve({ exists: true, firstSync: false, settings } as never) }
}

const cloud: Settings = {
  version: 1,
  outputs: [{
    platform:       'confluence',
    baseUrl:        'https://acme.atlassian.net/wiki',
    space:          'ENG',
    parentPageId:   '1',
    titlePrefix:    'ENG',
    direction:      'to-platform',
    content:        [],
    mermaidMode:    '',
    titleCollision: '',
    linkMode:       '',
  }],
  generators: [{ type: 'test-results', output: 'docs/tests', prepare: 'npm test' }],
}

const github = CI_PROVIDERS[0]
const azure = CI_PROVIDERS[1]

describe('planCiSetup', () => {
  it('says there is nothing to put in CI for a workspace without outputs', async () => {
    const plan = await planCiSetup({ engine: engineWith({ version: 1, outputs: [] }), files: memoryFiles(), workspaceRoot: '/w', version: '1.0.0' })

    expect(plan).toEqual({ kind: 'no-outputs' })
  })

  it('knows the outputs are in the repository when the folder settings hold them', async () => {
    const files = memoryFiles({ '.vscode/settings.json': '{ "loreMaster.outputs": [] }' })
    const plan = await planCiSetup({ engine: engineWith(cloud), files, workspaceRoot: '/w', version: '1.0.0' })

    expect(plan).toMatchObject({ kind: 'ready', needs: { settingsInRepo: true, version: '1.0.0' } })
  })

  it('flags settings that CI would not have', async () => {
    const plan = await planCiSetup({ engine: engineWith(cloud), files: memoryFiles(), workspaceRoot: '/w', version: '1.0.0' })

    expect(plan).toMatchObject({ kind: 'ready', needs: { settingsInRepo: false } })
  })
})

async function needs () {
  const plan = await planCiSetup({ engine: engineWith(cloud), files: memoryFiles(), workspaceRoot: '/w', version: '0.15.7' })
  if (plan.kind !== 'ready') {
    throw new Error('expected a plan')
  }

  return plan.needs
}

describe('buildCiSetupGuide', () => {
  it('writes a GitHub Actions workflow with the pinned engine, the secrets and the annotation commit', async () => {
    const { pipeline, guide } = buildCiSetupGuide(github, await needs())

    expect(pipeline.path).toBe('.github/workflows/lore-master.yml')
    expect(pipeline.content).toContain('VERSION: 0.15.7')
    expect(pipeline.content).toContain('LORE_MASTER_TOKEN: ${{ secrets.LORE_MASTER_TOKEN }}')
    expect(pipeline.content).toContain('./lore-master-engine sync --generate --prepare --yes')
    expect(pipeline.content).toContain('contents: write')
    expect(pipeline.content).toContain('[skip ci]')
    expect(guide).toContain('# Run LoreMaster in GitHub Actions')
    expect(guide).toContain('`LORE_MASTER_EMAIL`')
    expect(guide).toContain('"loreMaster.outputs"')
    expect(guide).toContain('npm test')
  })

  it('writes an Azure Pipelines file that maps each secret variable into the step', async () => {
    const { pipeline, guide } = buildCiSetupGuide(azure, await needs())

    expect(pipeline.path).toBe('.azure-pipelines/lore-master.yml')
    expect(pipeline.content).toContain("loreMasterVersion: '0.15.7'")
    expect(pipeline.content).toContain('LORE_MASTER_TOKEN: $(LORE_MASTER_TOKEN)')
    expect(guide).toContain('Keep this value secret')
  })
})

describe('writePipeline', () => {
  const pipeline = { path: '.github/workflows/lore-master.yml', content: 'name: LoreMaster\n' }

  it('creates the file, and leaves it alone when it is already current', async () => {
    const files = memoryFiles()

    expect(await writePipeline(files, pipeline, () => Promise.resolve(false))).toBe('created')
    expect(await writePipeline(files, pipeline, () => Promise.resolve(false))).toBe('unchanged')
  })

  it('replaces other content only when the user agrees', async () => {
    const files = memoryFiles({ [pipeline.path]: 'name: mine\n' })

    expect(await writePipeline(files, pipeline, () => Promise.resolve(false))).toBe('skipped')
    expect(files.files.get(pipeline.path)).toBe('name: mine\n')
    expect(await writePipeline(files, pipeline, () => Promise.resolve(true))).toBe('updated')
    expect(files.files.get(pipeline.path)).toBe(pipeline.content)
  })
})

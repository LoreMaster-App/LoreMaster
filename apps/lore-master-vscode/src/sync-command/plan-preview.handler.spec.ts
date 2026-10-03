import type { SyncPlanResult } from '../engine-protocol'
import { type PlanPreviewUI, planDetails, previewPlan, summarizePlan } from './plan-preview.handler'

const plan = (over: Partial<SyncPlanResult> = {}): SyncPlanResult => ({
  planId:  'p',
  actions: [{ kind: 'create', title: 'A', path: 'a.md' }],
  counts:  { create: 1 },
  ...over,
})

function ui (choices: ('run' | 'details' | 'cancel')[]): PlanPreviewUI & { details: number; errors: string[][] } {
  let next = 0
  const self = {
    details:      0,
    errors:       [] as string[][],
    choose:       () => Promise.resolve(choices[next++]),
    writeDetails: () => { self.details += 1 },
    showErrors:   (list: string[]) => {
      self.errors.push(list)

      return Promise.resolve()
    },
  }

  return self
}

describe('summarizePlan', () => {
  it('lists the non-zero counts, or says nothing to do', () => {
    expect(summarizePlan(plan({ counts: { create: 2, update: 1, unchanged: 0 } }))).toBe('2 create, 1 update')
    expect(summarizePlan(plan({ counts: {} }))).toBe('nothing to do')
  })
})

describe('planDetails', () => {
  it('renders an action, a warning and an error', () => {
    const lines = planDetails(plan({ warnings: ['w'], errors: ['e'], actions: [{ kind: 'update', title: 'A', path: 'a.md', reason: 'changed' }] }))

    expect(lines[0]).toContain('update')
    expect(lines[0]).toContain('A')
    expect(lines).toContain('warning: w')
    expect(lines).toContain('error: e')
  })
})

describe('previewPlan', () => {
  it('refuses a plan with errors and shows them', async () => {
    const preview = ui([])

    const run = await previewPlan(plan({ errors: ['broken'] }), preview)

    expect(run).toBe(false)
    expect(preview.errors).toEqual([['broken']])
    expect(preview.details).toBe(1)
  })

  it('returns true on Run', async () => {
    expect(await previewPlan(plan(), ui(['run']))).toBe(true)
  })

  it('shows details then runs', async () => {
    const preview = ui(['details', 'run'])

    const run = await previewPlan(plan(), preview)

    expect(run).toBe(true)
    expect(preview.details).toBe(1)
  })

  it('returns false on Cancel', async () => {
    expect(await previewPlan(plan(), ui(['cancel']))).toBe(false)
  })
})

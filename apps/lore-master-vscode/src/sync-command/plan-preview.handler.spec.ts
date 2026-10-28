import type { SyncPlanResult } from '../engine-protocol'
import { planDetails, summarizePlan } from './plan-preview.handler'

const plan = (over: Partial<SyncPlanResult> = {}): SyncPlanResult => ({
  planId:  'p',
  actions: [{ kind: 'create', title: 'A', path: 'a.md' }],
  counts:  { create: 1 },
  ...over,
})

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

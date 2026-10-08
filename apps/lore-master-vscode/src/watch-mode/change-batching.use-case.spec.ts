import { ChangeBatcher, type Timers } from './change-batching.use-case'

/** Timers that only move when the test says so. */
class FakeTimers implements Timers {
  private nextId = 1
  private readonly scheduled = new Map<number, { at: number; callback: () => void }>()
  time = 0

  setTimeout (callback: () => void, milliseconds: number): unknown {
    const id = this.nextId++
    this.scheduled.set(id, { at: this.time + milliseconds, callback })

    return id
  }

  clearTimeout (handle: unknown): void {
    this.scheduled.delete(handle as number)
  }

  now (): number {
    return this.time
  }

  /** Moves time forward, firing what falls due, and lets promises settle. */
  async advance (milliseconds: number): Promise<void> {
    const end = this.time + milliseconds
    for (;;) {
      const due = [...this.scheduled].filter(([, timer]) => timer.at <= end).sort((a, b) => a[1].at - b[1].at)[0]
      if (!due) {
        break
      }
      this.scheduled.delete(due[0])
      this.time = due[1].at
      due[1].callback()
      await settle()
    }
    this.time = end
  }
}

const settle = async (): Promise<void> => {
  for (let turn = 0; turn < 5; turn++) {
    await Promise.resolve()
  }
}

function batcher (run: (changed: string[]) => Promise<void>, timers: FakeTimers, onFailure?: (error: unknown, retryInMs: number) => void): ChangeBatcher {
  return new ChangeBatcher({ quietMs: 2000, maxBackoffMs: 16_000, run, timers, onFailure })
}

describe('ChangeBatcher', () => {
  it('runs one batch once nothing has changed for the quiet time', async () => {
    const timers = new FakeTimers()
    const batches: string[][] = []
    const subject = batcher(async changed => { batches.push(changed) }, timers)

    subject.add(['b.md'])
    await timers.advance(1999)
    expect(batches).toEqual([])
    await timers.advance(1)

    expect(batches).toEqual([['b.md']])
  })

  it('makes two quick edits one batch, restarting the wait with each', async () => {
    const timers = new FakeTimers()
    const batches: string[][] = []
    const subject = batcher(async changed => { batches.push(changed) }, timers)

    subject.add(['b.md'])
    await timers.advance(1500)
    subject.add(['a.md', 'b.md'])
    await timers.advance(1500)
    expect(batches).toEqual([])
    await timers.advance(500)

    expect(batches).toEqual([['a.md', 'b.md']])
  })

  it('holds changes made while a batch runs for the next batch, never overlapping', async () => {
    const timers = new FakeTimers()
    const batches: string[][] = []
    const gate = { open: () => {} }
    let active = 0
    let overlapped = false
    const subject = batcher(async changed => {
      active++
      overlapped ||= active > 1
      batches.push(changed)
      if (batches.length === 1) {
        // The compiler's library predates Promise.withResolvers.

        await new Promise<void>(resolve => { gate.open = resolve })
      }
      active--
    }, timers)

    subject.add(['a.md'])
    await timers.advance(2000)
    subject.add(['b.md'])
    await timers.advance(5000)
    expect(batches).toEqual([['a.md']])
    gate.open()
    await settle()
    await timers.advance(2000)

    expect(batches).toEqual([['a.md'], ['b.md']])
    expect(overlapped).toBe(false)
  })

  it('tries a failed batch again with its files, after a doubling wait, and resets on success', async () => {
    const timers = new FakeTimers()
    const batches: string[][] = []
    const failures: number[] = []
    let remaining = 2
    const subject = batcher(async changed => {
      batches.push(changed)
      if (remaining-- > 0) {
        throw new Error('down')
      }
    }, timers, (_error, retryInMs) => { failures.push(retryInMs) })

    subject.add(['a.md'])
    await timers.advance(2000)
    await timers.advance(2000)
    await timers.advance(4000)

    expect(failures).toEqual([2000, 4000])
    expect(batches).toEqual([['a.md'], ['a.md'], ['a.md']])
    expect(subject.busy).toBe(false)
  })

  it('does not shorten a retry wait when more changes arrive', async () => {
    const timers = new FakeTimers()
    const batches: string[][] = []
    let remaining = 2
    const subject = batcher(async changed => {
      batches.push(changed)
      if (remaining-- > 0) {
        throw new Error('down')
      }
    }, timers)

    subject.add(['a.md'])
    await timers.advance(2000)
    await timers.advance(2000)
    // The second failure waits 4 seconds; a change 1 second later must not run it early.
    subject.add(['b.md'])
    await timers.advance(1000)
    expect(batches).toHaveLength(2)
    await timers.advance(3000)

    expect(batches[2]).toEqual(['a.md', 'b.md'])
  })

  it('stops when disposed, and ignores changes after that', async () => {
    const timers = new FakeTimers()
    const batches: string[][] = []
    const subject = batcher(async changed => { batches.push(changed) }, timers)

    subject.add(['a.md'])
    subject.dispose()
    subject.add(['b.md'])
    await timers.advance(10_000)

    expect(batches).toEqual([])
    expect(subject.busy).toBe(false)
  })

  it('ignores an empty list of changes', async () => {
    const timers = new FakeTimers()
    const subject = batcher(async () => {}, timers)

    subject.add([])

    expect(subject.busy).toBe(false)
  })
})

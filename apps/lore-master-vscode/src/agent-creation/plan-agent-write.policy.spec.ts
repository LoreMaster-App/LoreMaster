import { FILE_MARKER, targetById } from './agent-target.config'
import { planAgentWrite } from './plan-agent-write.policy'

const file = targetById('claude-code')
const block = targetById('agents-md')

const rendered = `---\nname: x\n---\n\n${FILE_MARKER}\n\nbody\n`
const mineBlock = block.render('rules v2\n')
const olderBlock = block.render('rules v1\n')

describe('planAgentWrite for a whole file', () => {
  it('creates a file that is not there', () => {
    expect(planAgentWrite(file, undefined, rendered)).toEqual({ action: 'create', content: rendered })
  })

  it('leaves a file that already says what it should', () => {
    expect(planAgentWrite(file, rendered, rendered)).toEqual({ action: 'unchanged' })
  })

  it('updates a file it wrote earlier, which carries its marker', () => {
    expect(planAgentWrite(file, rendered.replace('body', 'older body'), rendered)).toEqual({ action: 'update', content: rendered })
  })

  it('reports a conflict for a file it did not write, and writes nothing', () => {
    expect(planAgentWrite(file, '# My own agent\n', rendered)).toEqual({ action: 'conflict' })
  })
})

describe('planAgentWrite for a block in a shared file', () => {
  it('creates the file with the block when there is none', () => {
    expect(planAgentWrite(block, undefined, mineBlock)).toEqual({ action: 'create', content: mineBlock })
  })

  it('appends after the user\'s content, leaving it untouched, with one blank line between', () => {
    expect(planAgentWrite(block, '# Agents\n\nBe kind.\n', mineBlock)).toEqual({ action: 'append', content: `# Agents\n\nBe kind.\n\n${mineBlock}` })
    expect(planAgentWrite(block, '# Agents', mineBlock)).toEqual({ action: 'append', content: `# Agents\n\n${mineBlock}` })
    expect(planAgentWrite(block, '# Agents\n\n', mineBlock)).toEqual({ action: 'append', content: `# Agents\n\n${mineBlock}` })
    expect(planAgentWrite(block, '', mineBlock)).toEqual({ action: 'append', content: mineBlock })
  })

  it('replaces its own earlier block in place, keeping what is around it', () => {
    const existing = `# Agents\n\nBefore.\n\n${olderBlock}\nAfter.\n`

    expect(planAgentWrite(block, existing, mineBlock)).toEqual({ action: 'update', content: `# Agents\n\nBefore.\n\n${mineBlock}\nAfter.\n` })
  })

  it('leaves the file alone when its block is already current', () => {
    expect(planAgentWrite(block, `# Agents\n\n${mineBlock}`, mineBlock)).toEqual({ action: 'unchanged' })
  })

  it('does not take a dollar sign in the new block for a replacement pattern', () => {
    const priced = block.render('costs $& and $1\n')

    expect(planAgentWrite(block, olderBlock, priced)).toEqual({ action: 'update', content: priced })
  })
})

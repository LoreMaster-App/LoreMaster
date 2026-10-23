import { locateDiagramFences } from './locate-diagram-fences.algorithm'

describe('locateDiagramFences', () => {
  it('finds Mermaid and D2 fences with the line their source starts on', () => {
    const text = ['# Doc', '', '```mermaid', 'graph TD', '  A-->B', '```', '', '```d2', 'a -> b', '```'].join('\n')

    expect(locateDiagramFences(text)).toEqual([
      { language: 'mermaid', source: 'graph TD\n  A-->B', firstLine: 3 },
      { language: 'd2', source: 'a -> b', firstLine: 8 },
    ])
  })

  it('skips other languages, and a diagram fence inside a longer fence', () => {
    const text = ['```ts', 'let a = 1', '```', '````markdown', '```mermaid', 'graph TD', '```', '````'].join('\n')

    expect(locateDiagramFences(text)).toEqual([])
  })

  it('reads tilde fences, any case, an info string after the language and windows line ends', () => {
    const text = '~~~ Mermaid title="x"\r\ngraph TD\r\n~~~\r\n'

    expect(locateDiagramFences(text)).toEqual([{ language: 'mermaid', source: 'graph TD', firstLine: 1 }])
  })

  it('runs a fence that is never closed to the end of the document', () => {
    expect(locateDiagramFences('```d2\na -> b\nc')).toEqual([{ language: 'd2', source: 'a -> b\nc', firstLine: 1 }])
  })

  it('does not close a fence with a shorter or different marker', () => {
    const text = ['````d2', 'a -> b', '```', 'still the diagram', '````'].join('\n')

    expect(locateDiagramFences(text)).toEqual([{ language: 'd2', source: 'a -> b\n```\nstill the diagram', firstLine: 1 }])
  })
})

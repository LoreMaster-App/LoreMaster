import type * as vscode from 'vscode'
import { createMermaidRenderer, type DiagramPanel } from './mermaid-webview.client'

interface FakePanel extends DiagramPanel {
  posted:   { id: number, source: string }[]
  disposed: boolean
  reply (message: { id: number, svg?: string, error?: string }): void
}

function fakePanel (): FakePanel {
  const receivers: ((message: { id: number, svg?: string, error?: string }) => unknown)[] = []
  const disposers: (() => unknown)[] = []
  const panel: FakePanel = {
    posted:   [],
    disposed: false,
    webview:  {
      html: '',
      postMessage (message: unknown) {
        panel.posted.push(message as { id: number, source: string })

        return Promise.resolve(true)
      },
      onDidReceiveMessage (listener) {
        receivers.push(listener)

        return { dispose () {} }
      },
    },
    onDidDispose (listener) {
      disposers.push(listener)

      return { dispose () {} }
    },
    dispose () {
      panel.disposed = true
      for (const listener of disposers) {
        listener()
      }
    },
    reply (message) {
      for (const listener of receivers) {
        listener(message)
      }
    },
  }

  return panel
}

const uri = {} as vscode.Uri

function rendererWith (panels: FakePanel[], budgetMs: number) {
  return createMermaidRenderer(uri, {
    budgetMs,
    openPanel: () => {
      const panel = fakePanel()
      panels.push(panel)

      return { panel, mermaidUri: 'vscode-resource://mermaid.min.js', cspSource: 'vscode-resource:' }
    },
  })
}

describe('createMermaidRenderer', () => {
  afterEach(() => {
    jest.useRealTimers()
  })

  it('returns the svg the webview answers with', async () => {
    const panels: FakePanel[] = []
    const renderer = rendererWith(panels, 1000)

    const drawing = renderer.render('graph LR; A-->B')
    panels[0].reply({ id: panels[0].posted[0].id, svg: '<svg>ok</svg>' })

    await expect(drawing).resolves.toBe('<svg>ok</svg>')
    expect(panels[0].webview.html).toContain('mermaid.initialize')
  })

  it('reuses one panel for every diagram', async () => {
    const panels: FakePanel[] = []
    const renderer = rendererWith(panels, 1000)

    const first = renderer.render('a')
    panels[0].reply({ id: 1, svg: '<svg>1</svg>' })
    await first
    const second = renderer.render('b')
    panels[0].reply({ id: 2, svg: '<svg>2</svg>' })
    await second

    expect(panels).toHaveLength(1)
  })

  it('rejects with the webview\'s own reason', async () => {
    const panels: FakePanel[] = []
    const renderer = rendererWith(panels, 1000)

    const drawing = renderer.render('graph LR; A-->')
    panels[0].reply({ id: 1, error: 'Parse error on line 1' })

    await expect(drawing).rejects.toThrow('Parse error on line 1')
  })

  it('gives up on a diagram that never finishes and starts the next one on a fresh panel', async () => {
    jest.useFakeTimers()
    const panels: FakePanel[] = []
    const renderer = rendererWith(panels, 5000)

    const stuck = await expect(renderer.render('huge')).rejects.toThrow('did not finish drawing within 5s')
    jest.advanceTimersByTime(5000)
    await stuck
    expect(panels[0].disposed).toBe(true)

    const next = renderer.render('small')
    expect(panels).toHaveLength(2)
    panels[1].reply({ id: 2, svg: '<svg>next</svg>' })
    await expect(next).resolves.toBe('<svg>next</svg>')
  })

  it('does not time a diagram out once it has been answered', async () => {
    jest.useFakeTimers()
    const panels: FakePanel[] = []
    const renderer = rendererWith(panels, 5000)

    const drawing = renderer.render('a')
    panels[0].reply({ id: 1, svg: '<svg/>' })
    await drawing
    jest.advanceTimersByTime(10_000)

    expect(panels[0].disposed).toBe(false)
  })

  it('fails the waiting diagrams when the user closes the panel', async () => {
    const panels: FakePanel[] = []
    const renderer = rendererWith(panels, 1000)

    const drawing = renderer.render('a')
    panels[0].dispose()

    await expect(drawing).rejects.toThrow('the diagram renderer was closed')
  })
})

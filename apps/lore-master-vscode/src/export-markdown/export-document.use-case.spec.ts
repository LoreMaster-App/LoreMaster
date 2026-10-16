import { EXPORT_FORMATS_METHOD, EXPORT_RUN_METHOD, type ExportRunParams } from '../engine-protocol'
import { exportDocument, exportDocumentAsText, listExportFormats } from './export-document.use-case'
import type { ExportEngine } from './export-markdown.contract'

function engine (failFormat?: string): { engine: ExportEngine, runs: ExportRunParams[] } {
  const runs: ExportRunParams[] = []
  const fake: ExportEngine = {
    request<R> (method: string, params?: unknown): Promise<R> {
      if (method === EXPORT_FORMATS_METHOD) {
        return Promise.resolve({ formats: [{ id: 'html', label: 'HTML', extension: '.html' }, { id: 'docx', label: 'Word', extension: '.docx', binary: true }] } as never)
      }
      expect(method).toBe(EXPORT_RUN_METHOD)
      const run = params as ExportRunParams
      runs.push(run)
      if (run.format === failFormat) {
        return Promise.reject(new Error('cannot write'))
      }

      return Promise.resolve(run.inline ? { name: 'a.txt', text: `text of ${run.content}` } as never : { name: `a.${run.format}`, path: `/w/a.${run.format}` } as never)
    },
  }

  return { engine: fake, runs }
}

describe('exportDocument', () => {
  it('writes one file per format and passes the unsaved text and the folder', async () => {
    const { engine: fake, runs } = engine()

    const outcome = await exportDocument(fake, { path: '/w/a.md', content: '# A', formats: ['html', 'docx'], outDir: '/w/dist' })

    expect(outcome).toEqual({ written: [{ format: 'html', name: 'a.html', path: '/w/a.html' }, { format: 'docx', name: 'a.docx', path: '/w/a.docx' }], failed: [] })
    expect(runs).toEqual([
      { path: '/w/a.md', content: '# A', format: 'html', outDir: '/w/dist' },
      { path: '/w/a.md', content: '# A', format: 'docx', outDir: '/w/dist' },
    ])
  })

  it('hands the print settings to the engine for pdf and png', async () => {
    const { engine: fake, runs } = engine()

    await exportDocument(fake, { path: '/w/a.md', formats: ['pdf'], print: { browserPath: '/b/chrome', pageSize: 'Letter', imageWidth: 800 } })

    expect(runs[0]).toMatchObject({ format: 'pdf', browserPath: '/b/chrome', pageSize: 'Letter', imageWidth: 800 })
  })

  it('keeps going when one format fails and says which', async () => {
    const { engine: fake } = engine('html')

    const outcome = await exportDocument(fake, { path: '/w/a.md', formats: ['html', 'docx'] })

    expect(outcome.failed).toEqual([{ format: 'html', message: 'cannot write' }])
    expect(outcome.written.map(file => file.format)).toEqual(['docx'])
  })
})

describe('exportDocumentAsText and listExportFormats', () => {
  it('asks for the result inline', async () => {
    const { engine: fake, runs } = engine()

    expect(await exportDocumentAsText(fake, { path: '/w/a.md', content: 'x', format: 'jira-wiki' })).toBe('text of x')
    expect(runs[0]).toMatchObject({ inline: true, format: 'jira-wiki' })
  })

  it('lists what the engine offers', async () => {
    const formats = await listExportFormats(engine().engine)

    expect(formats.map(format => format.id)).toEqual(['html', 'docx'])
  })
})

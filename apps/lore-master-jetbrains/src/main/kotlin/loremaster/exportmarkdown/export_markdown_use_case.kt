package loremaster.exportmarkdown

import loremaster.engineprocess.EngineRpcClient
import java.util.concurrent.TimeUnit

private const val EXPORT_TIMEOUT_SECONDS = 120L

/** Asks the engine which formats it can export to, in the order a picker shows them. */
fun listExportFormats(engine: EngineRpcClient): List<ExportFormat> =
    engine.request(METHOD_EXPORT_FORMATS, null, ExportFormatsResult::class.java)
        .get(EXPORT_TIMEOUT_SECONDS, TimeUnit.SECONDS)
        .formats

/** Exports one Markdown file (or the editor's unsaved text of it) and returns what the engine wrote. */
fun exportMarkdown(engine: EngineRpcClient, params: ExportRunParams): ExportRunResult =
    engine.request(METHOD_EXPORT_RUN, params, ExportRunResult::class.java)
        .get(EXPORT_TIMEOUT_SECONDS, TimeUnit.SECONDS)

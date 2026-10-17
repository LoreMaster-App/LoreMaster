package loremaster.exportmarkdown

/** The engine's `export/formats`: no params, answers [ExportFormatsResult]. */
const val METHOD_EXPORT_FORMATS = "export/formats"

/** The engine's `export/run`: converts one Markdown document and writes the result. */
const val METHOD_EXPORT_RUN = "export/run"

/** One format a document can be exported to. [extension] has its dot. */
data class ExportFormat(
    val id: String = "",
    val label: String = "",
    val extension: String = "",
    val binary: Boolean = false,
)

data class ExportFormatsResult(val formats: List<ExportFormat> = emptyList())

/** What `export/run` takes. [content] is the editor's unsaved text; the file is read when it is null. */
data class ExportRunParams(
    val path: String,
    val format: String,
    val content: String? = null,
    val outDir: String? = null,
    val browserPath: String? = null,
    val referencePath: String? = null,
    val pageSize: String? = null,
    val imageWidth: Int? = null,
)

/** What `export/run` answers: the file's name and where it was written. */
data class ExportRunResult(val name: String = "", val path: String = "")

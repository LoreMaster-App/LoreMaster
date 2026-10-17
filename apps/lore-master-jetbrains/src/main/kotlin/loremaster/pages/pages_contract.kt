package loremaster.pages

const val METHOD_PAGES_CHECK = "pages/check"
const val METHOD_PAGES_BUILD = "pages/build"
const val METHOD_PAGES_PUBLISH = "pages/publish"

/** [output] is the index of a github-pages or github-wiki output in the settings' outputs list. */
data class PagesCheckParams(val workspaceRoot: String, val output: Int)

data class PagesChange(val path: String = "", val kind: String = "")

data class PagesCheckResult(
    val branch: String? = null,
    val remote: String? = null,
    val upToDate: Boolean = false,
    val changes: List<PagesChange>? = null,
    val changesTotal: Int = 0,
    val newPages: Int = 0,
    val newPagePaths: List<String>? = null,
    val files: Int = 0,
    val warnings: List<String>? = null,
    val errors: List<String>? = null,
)

data class PagesBuildParams(val workspaceRoot: String, val output: Int, val outDir: String)

data class PagesBuildResult(
    val outDir: String? = null,
    val files: Int = 0,
    val warnings: List<String>? = null,
    val errors: List<String>? = null,
)

data class PagesPublishParams(val workspaceRoot: String, val output: Int)

data class PagesPublishResult(
    val branch: String? = null,
    val remote: String? = null,
    val commit: String? = null,
    val changed: Boolean = false,
    val files: Int = 0,
    val url: String? = null,
    val warnings: List<String>? = null,
    val errors: List<String>? = null,
)

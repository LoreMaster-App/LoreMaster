package loremaster.generators

const val METHOD_GENERATORS_RUN = "generators/run"

/** [prepare] runs each generator's prepare command first; without it no command is ever executed. */
data class GeneratorsRunParams(val workspaceRoot: String, val prepare: Boolean = false, val generators: List<Int>? = null)

data class GeneratorRun(
    val index: Int = 0,
    val type: String = "",
    val output: String = "",
    val prepared: Boolean = false,
    val written: List<String>? = null,
    val unchanged: List<String>? = null,
    val removed: List<String>? = null,
    val warnings: List<String>? = null,
    val error: String? = null,
)

data class GeneratorsRunResult(val runs: List<GeneratorRun> = emptyList())

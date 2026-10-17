package loremaster.generators

import loremaster.engineprocess.EngineRpcClient
import java.util.concurrent.TimeUnit

/** Runs the project's generators, which write ordinary Markdown pages that then sync like any other. */
fun runGenerators(engine: EngineRpcClient, workspaceRoot: String, prepare: Boolean, only: List<Int>? = null): GeneratorsRunResult =
    engine.request(METHOD_GENERATORS_RUN, GeneratorsRunParams(workspaceRoot, prepare, only), GeneratorsRunResult::class.java)
        .get(30, TimeUnit.MINUTES)

/** One line for the notification: what was written, what failed. */
fun describeRuns(result: GeneratorsRunResult): String {
    if (result.runs.isEmpty()) return "No generator is configured (loreMaster.generators)."

    return result.runs.joinToString("\n") { run ->
        run.error?.let { "${run.type}: failed: $it" }
            ?: "${run.type}: ${run.written.orEmpty().size} written, ${run.unchanged.orEmpty().size} unchanged, ${run.removed.orEmpty().size} removed"
    }
}

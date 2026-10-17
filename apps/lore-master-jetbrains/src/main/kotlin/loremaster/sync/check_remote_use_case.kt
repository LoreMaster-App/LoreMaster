package loremaster.sync

import loremaster.connection.Credential
import loremaster.connection.closeSession
import loremaster.connection.openSession
import loremaster.engineprocess.EngineRpcClient
import loremaster.projectsettings.OutputSettings
import java.util.concurrent.TimeUnit

private const val PLAN_TIMEOUT_MINUTES = 10L

/** The kinds of action a plan lists, in the order a person wants to read them. */
private val PLAN_KINDS = listOf(
    "create" to "to create",
    "update" to "to update",
    "move" to "to move",
    "rename_title" to "to rename",
    "adopt" to "to adopt",
    "conflict" to "in conflict",
    "orphan" to "whose file is gone",
    "unchanged" to "unchanged",
)

/**
 * Asks the platform where an output stands without writing anything: sign in, plan, close. Null when the user declined to sign in.
 * The plan is not executed and is thrown away by the engine.
 */
fun checkRemote(engine: EngineRpcClient, credentialFor: (baseUrl: String) -> Credential?, workspaceRoot: String, output: OutputSettings, outputIndex: Int): SyncPlanResult? {
    val credential = credentialFor(output.baseUrl) ?: return null
    val session = openSession(engine, output.baseUrl, credential)
    try {
        return engine.request(METHOD_SYNC_PLAN, SyncPlanParams(session.sessionId, workspaceRoot, outputIndex, null), SyncPlanResult::class.java)
            .get(PLAN_TIMEOUT_MINUTES, TimeUnit.MINUTES)
    } finally {
        runCatching { closeSession(engine, session.sessionId) }
    }
}

/** One sentence on what a sync would do: the plan's counts, or its errors. */
fun describePlan(label: String, plan: SyncPlanResult): String {
    if (!plan.errors.isNullOrEmpty()) return "$label cannot sync yet:\n${plan.errors.joinToString("\n")}"
    val parts = PLAN_KINDS.mapNotNull { (kind, words) -> plan.counts[kind]?.takeIf { it > 0 }?.let { "$it $words" } }

    return if (parts.isEmpty()) "$label: nothing to sync." else "$label: ${parts.joinToString(", ")}."
}

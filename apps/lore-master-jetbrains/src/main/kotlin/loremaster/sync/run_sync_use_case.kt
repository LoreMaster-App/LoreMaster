package loremaster.sync

import loremaster.connection.Credential
import loremaster.connection.closeSession
import loremaster.connection.openSession
import loremaster.engineprocess.EngineRpcClient
import loremaster.projectsettings.OutputSettings
import java.util.concurrent.TimeUnit

private const val PLAN_TIMEOUT_MINUTES = 10L
private const val EXECUTE_TIMEOUT_MINUTES = 60L

/** How a sync ended. */
sealed interface SyncOutcome {
    data class Done(val plan: SyncPlanResult, val report: SyncExecuteResult) : SyncOutcome

    /** The plan has errors the author must fix first. */
    data class Refused(val errors: List<String>) : SyncOutcome

    /** The user declined to sign in or to go ahead with the plan. */
    data object Cancelled : SyncOutcome
}

/**
 * One sync of one output: sign in, plan, let the user review the plan, execute. The session lives only for the run. The
 * user's decisions are asked through [credentialFor] (null: declined) and [review] (null: declined), which keeps this
 * free of any window.
 */
class RunSync(
    private val engine: EngineRpcClient,
    private val credentialFor: (baseUrl: String) -> Credential?,
    private val review: (SyncPlanResult) -> SyncChoice?,
) {
    fun run(workspaceRoot: String, output: OutputSettings, outputIndex: Int, scope: List<String>? = null): SyncOutcome {
        val credential = credentialFor(output.baseUrl) ?: return SyncOutcome.Cancelled
        val session = openSession(engine, output.baseUrl, credential)
        try {
            val plan = engine.request(METHOD_SYNC_PLAN, SyncPlanParams(session.sessionId, workspaceRoot, outputIndex, scope), SyncPlanResult::class.java)
                .get(PLAN_TIMEOUT_MINUTES, TimeUnit.MINUTES)
            if (!plan.errors.isNullOrEmpty()) {
                return SyncOutcome.Refused(plan.errors)
            }
            val choice = review(plan) ?: return SyncOutcome.Cancelled
            val report = engine.request(METHOD_SYNC_EXECUTE, SyncExecuteParams(plan.planId, choice.force, choice.prune), SyncExecuteResult::class.java)
                .get(EXECUTE_TIMEOUT_MINUTES, TimeUnit.MINUTES)

            return SyncOutcome.Done(plan, report)
        } finally {
            runCatching { closeSession(engine, session.sessionId) }
        }
    }
}

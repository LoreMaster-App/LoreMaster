package loremaster.watch

import loremaster.configdescription.describeConfig
import loremaster.configdescription.isPages
import loremaster.connection.CredentialVaultClient
import loremaster.engineprocess.EngineRpcClient
import loremaster.projectsettings.readProjectSettings
import loremaster.sync.RunSync
import loremaster.sync.SyncChoice
import loremaster.sync.SyncOutcome

/**
 * Syncs [scope] (or everything) to every Confluence output without asking anything: it signs in with the saved credential and
 * goes ahead with the plan as it is, never overwriting a page edited on the platform and never trashing. Throws when something
 * stopped it, so the watcher keeps the batch and tries again.
 */
fun syncConfluenceOutputs(engine: EngineRpcClient, workspaceRoot: String, scope: List<String>?) {
    val description = describeConfig(engine)
    val outputs = readProjectSettings(engine, workspaceRoot).settings.outputs.withIndex().filter { description.isPages(it.value.platform) }
    for ((index, output) in outputs) {
        val flow = RunSync(
            engine,
            { CredentialVaultClient.load(it) ?: error("No saved sign-in for $it. Run LoreMaster: Connect to Confluence.") },
            { SyncChoice() },
        )
        when (val outcome = flow.run(workspaceRoot, output, index, scope)) {
            is SyncOutcome.Refused -> error("${output.label()} cannot be synced: ${outcome.errors.first()}")
            is SyncOutcome.Done -> outcome.report.pages.firstOrNull { it.outcome == "failed" }
                ?.let { error("${output.label()}: ${it.title} failed: ${it.error}") }
            SyncOutcome.Cancelled -> Unit
        }
    }
}

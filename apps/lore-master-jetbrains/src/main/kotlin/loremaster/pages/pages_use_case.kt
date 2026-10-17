package loremaster.pages

import loremaster.engineprocess.EngineRpcClient
import java.util.concurrent.TimeUnit

private const val PAGES_TIMEOUT_MINUTES = 30L

/** Compares the site an output would publish with what is published, without publishing. */
fun checkPages(engine: EngineRpcClient, workspaceRoot: String, output: Int): PagesCheckResult =
    engine.request(METHOD_PAGES_CHECK, PagesCheckParams(workspaceRoot, output), PagesCheckResult::class.java)
        .get(PAGES_TIMEOUT_MINUTES, TimeUnit.MINUTES)

/** Writes the static site of an output into a local folder; no git, no network. */
fun buildPages(engine: EngineRpcClient, workspaceRoot: String, output: Int, outDir: String): PagesBuildResult =
    engine.request(METHOD_PAGES_BUILD, PagesBuildParams(workspaceRoot, output, outDir), PagesBuildResult::class.java)
        .get(PAGES_TIMEOUT_MINUTES, TimeUnit.MINUTES)

/** Publishes the site to the output's branch with the user's own git. */
fun publishPages(engine: EngineRpcClient, workspaceRoot: String, output: Int): PagesPublishResult =
    engine.request(METHOD_PAGES_PUBLISH, PagesPublishParams(workspaceRoot, output), PagesPublishResult::class.java)
        .get(PAGES_TIMEOUT_MINUTES, TimeUnit.MINUTES)

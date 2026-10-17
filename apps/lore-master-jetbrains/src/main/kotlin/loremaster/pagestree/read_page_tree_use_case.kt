package loremaster.pagestree

import loremaster.engineprocess.EngineRpcClient
import java.util.concurrent.TimeUnit

/** The pages an output would sync, as the files alone describe them. */
fun readPageTree(engine: EngineRpcClient, workspaceRoot: String, output: Int): WorkspaceTreeResult =
    engine.request(METHOD_WORKSPACE_TREE, WorkspaceTreeParams(workspaceRoot, output), WorkspaceTreeResult::class.java)
        .get(2, TimeUnit.MINUTES)

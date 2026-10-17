package loremaster.syncview

import loremaster.engineprocess.EngineRpcClient
import java.util.concurrent.TimeUnit

/** The pages an output would sync, as the files alone describe them. */
fun readPageTree(engine: EngineRpcClient, workspaceRoot: String, output: Int): WorkspaceTreeResult =
    engine.request(METHOD_WORKSPACE_TREE, WorkspaceTreeParams(workspaceRoot, output), WorkspaceTreeResult::class.java)
        .get(2, TimeUnit.MINUTES)

/** Every Markdown file of the project, and which storages sync it, whatever a storage leaves out. */
fun readLocalTree(engine: EngineRpcClient, workspaceRoot: String): WorkspaceTreeResult =
    engine.request(METHOD_WORKSPACE_TREE, WorkspaceTreeParams(workspaceRoot, 0, scope = "local"), WorkspaceTreeResult::class.java)
        .get(2, TimeUnit.MINUTES)

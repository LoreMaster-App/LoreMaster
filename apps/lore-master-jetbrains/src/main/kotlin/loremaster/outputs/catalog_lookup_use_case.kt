package loremaster.outputs

import loremaster.engineprocess.EngineRpcClient
import java.util.concurrent.TimeUnit

private const val LOOKUP_TIMEOUT_SECONDS = 60L

/** Every space the signed-in user can see, sorted by name. */
fun listSpaces(engine: EngineRpcClient, sessionId: String): List<Space> =
    engine.request(METHOD_SPACE_LIST, SpaceListParams(sessionId), SpaceListResult::class.java)
        .get(LOOKUP_TIMEOUT_SECONDS, TimeUnit.SECONDS)
        .spaces

/** Pages of a space whose title contains [query], ignoring case; an empty query lists the space's pages. */
fun searchPages(engine: EngineRpcClient, sessionId: String, spaceKey: String, query: String): List<RemotePage> =
    engine.request(METHOD_PAGE_SEARCH, PageSearchParams(sessionId, spaceKey, query), PagesResult::class.java)
        .get(LOOKUP_TIMEOUT_SECONDS, TimeUnit.SECONDS)
        .pages

/** The direct children of a page. */
fun pageChildren(engine: EngineRpcClient, sessionId: String, pageId: String): List<RemotePage> =
    engine.request(METHOD_PAGE_CHILDREN, PageChildrenParams(sessionId, pageId), PagesResult::class.java)
        .get(LOOKUP_TIMEOUT_SECONDS, TimeUnit.SECONDS)
        .pages

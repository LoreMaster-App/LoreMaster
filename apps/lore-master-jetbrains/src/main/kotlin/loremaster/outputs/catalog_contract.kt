package loremaster.outputs

/** The engine's `space/list`: the spaces the session's user can see, sorted by name. */
const val METHOD_SPACE_LIST = "space/list"

/** The engine's `page/search`: pages of a space whose title contains the query; an empty query lists them. */
const val METHOD_PAGE_SEARCH = "page/search"

/** The engine's `page/children`: a page's direct children, for browsing to a parent page. */
const val METHOD_PAGE_CHILDREN = "page/children"

data class SpaceListParams(val sessionId: String)

/** One space. [homepageId] is its home page, the default parent page; empty when none. */
data class Space(val id: String = "", val key: String = "", val name: String = "", val homepageId: String? = null) {
    override fun toString() = "$name ($key)"
}

data class SpaceListResult(val spaces: List<Space> = emptyList())

data class PageSearchParams(val sessionId: String, val spaceKey: String, val query: String, val limit: Int = 50)

data class PageChildrenParams(val sessionId: String, val pageId: String)

data class RemotePage(val id: String = "", val title: String = "", val parentId: String? = null, val url: String? = null) {
    override fun toString() = title
}

data class PagesResult(val pages: List<RemotePage> = emptyList())

package loremaster.configdescription

/** The platform an output uses, or null for one this plugin's engine does not describe. */
fun ConfigDescription.platformOf(platformId: String): PlatformDescriptor? = platforms.firstOrNull { it.id == platformId }

/** True for an output published as a whole (a site or wiki) rather than synced page by page. */
fun ConfigDescription.isSite(platformId: String): Boolean = platformOf(platformId)?.delivery == DELIVERY_SITE

/** True for an output synced page by page (a page tree, rollback, watch). */
fun ConfigDescription.isPages(platformId: String): Boolean = platformOf(platformId)?.delivery == DELIVERY_PAGES

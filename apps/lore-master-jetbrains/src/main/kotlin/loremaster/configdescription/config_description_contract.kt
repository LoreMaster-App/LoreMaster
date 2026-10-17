package loremaster.configdescription

/** The engine's `config/describe`: everything that can be configured, so this plugin keeps no list of platforms or settings. */
const val METHOD_CONFIG_DESCRIBE = "config/describe"

/** Delivery of a platform: page by page (plan, execute, rollback, page tree) or published as one unit. */
const val DELIVERY_PAGES = "pages"
const val DELIVERY_SITE = "site"

/** Field types a form can show. */
const val FIELD_TEXT = "text"
const val FIELD_CHOICE = "choice"
const val FIELD_LIST = "list"
const val FIELD_BOOL = "bool"
const val FIELD_NUMBER = "number"
const val FIELD_PATH = "path"

/** Lookup kinds this plugin offers a picker for; any other kind is shown as plain text. */
const val LOOKUP_SPACE = "space"
const val LOOKUP_PAGE = "page"

data class ConfigDescription(
    val platforms: List<PlatformDescriptor> = emptyList(),
    val outputFields: List<FieldDescriptor> = emptyList(),
    val contentFields: List<FieldDescriptor> = emptyList(),
    val generators: List<GeneratorTypeDescriptor> = emptyList(),
    val generatorFields: List<FieldDescriptor> = emptyList(),
    val sections: List<SectionDescriptor> = emptyList(),
)

data class PlatformDescriptor(
    val id: String = "",
    val label: String = "",
    val description: String = "",
    val icon: String = "",
    val delivery: String = DELIVERY_PAGES,
    val nameTemplate: String = "",
    val summaryTemplate: String = "",
    val connection: ConnectionDescriptor? = null,
    val fields: List<FieldDescriptor> = emptyList(),
)

data class ConnectionDescriptor(
    /** The key of the output field holding the site address. */
    val field: String = "",
    val detectMethod: String = "",
    val kinds: List<CredentialKindDescriptor> = emptyList(),
)

data class CredentialKindDescriptor(
    val id: String = "",
    val label: String = "",
    val help: String = "",
    val editions: List<String> = emptyList(),
    val fields: List<CredentialFieldDescriptor> = emptyList(),
)

data class CredentialFieldDescriptor(val key: String = "", val label: String = "", val secret: Boolean = false)

data class FieldDescriptor(
    val key: String = "",
    val label: String = "",
    val help: String = "",
    val type: String = FIELD_TEXT,
    val placeholder: String = "",
    val required: Boolean = false,
    val default: String = "",
    val choices: List<ChoiceDescriptor> = emptyList(),
    val suggestions: List<String> = emptyList(),
    val min: Int? = null,
    val max: Int? = null,
    val pathKind: String = "",
    val pattern: String = "",
    val patternHelp: String = "",
    val advanced: Boolean = false,
    /** The editors that honour the setting; empty means every editor. */
    val shells: List<String> = emptyList(),
    val lookup: LookupDescriptor? = null,
)

data class ChoiceDescriptor(val value: String = "", val label: String = "", val help: String = "")

data class LookupDescriptor(
    val kind: String = "",
    val needs: List<String> = emptyList(),
    val fills: List<FillDescriptor> = emptyList(),
)

data class FillDescriptor(val field: String = "", val from: String = "")

data class GeneratorTypeDescriptor(
    val id: String = "",
    val label: String = "",
    val description: String = "",
    val fields: List<FieldDescriptor> = emptyList(),
)

data class SectionDescriptor(
    val id: String = "",
    val label: String = "",
    /** "project" settings travel with the repository; "user" settings are the person's own. */
    val scope: String = "",
    val settings: List<FieldDescriptor> = emptyList(),
)

package loremaster.connection

import com.google.gson.Gson
import com.intellij.credentialStore.CredentialAttributes
import com.intellij.credentialStore.Credentials
import com.intellij.credentialStore.generateServiceName
import com.intellij.ide.passwordSafe.PasswordSafe

/**
 * The credential for a site, kept in the IDE's password safe (the operating system keychain, or the IDE's own encrypted
 * store). One entry per site address, so it serves every project. The credential travels to the engine per session and is
 * never written to a file.
 */
object CredentialVaultClient {
    private val gson = Gson()

    fun load(baseUrl: String): Credential? =
        PasswordSafe.instance.getPassword(attributes(baseUrl))
            ?.let { runCatching { gson.fromJson(it, Credential::class.java) }.getOrNull() }

    fun save(baseUrl: String, credential: Credential) =
        PasswordSafe.instance.set(attributes(baseUrl), Credentials(baseUrl, gson.toJson(credential)))

    fun forget(baseUrl: String) = PasswordSafe.instance.set(attributes(baseUrl), null)

    private fun attributes(baseUrl: String) = CredentialAttributes(generateServiceName("LoreMaster", normaliseAddress(baseUrl)))
}

/** The same site written two ways (a trailing slash, other case) shares one entry. */
fun normaliseAddress(baseUrl: String): String = baseUrl.trim().trimEnd('/').lowercase()

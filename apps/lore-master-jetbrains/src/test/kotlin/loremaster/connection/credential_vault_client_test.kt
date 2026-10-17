package loremaster.connection

import com.intellij.testFramework.fixtures.BasePlatformTestCase

/** The credential round trip through the IDE's password safe. */
class CredentialVaultClientTest : BasePlatformTestCase() {
    fun `test a credential is kept per site and forgotten`() {
        val site = "https://vault-test.example.net/wiki"
        val credential = Credential(CredentialKind.API_TOKEN, email = "ada@example.net", token = "t0ken")
        try {
            CredentialVaultClient.save(site, credential)

            assertEquals(credential, CredentialVaultClient.load("https://Vault-Test.example.net/wiki/"))
            assertNull(CredentialVaultClient.load("https://other.example.net"))
        } finally {
            CredentialVaultClient.forget(site)
        }
        assertNull(CredentialVaultClient.load(site))
    }

    fun `test the connect actions are registered`() {
        val manager = com.intellij.openapi.actionSystem.ActionManager.getInstance()

        assertTrue(manager.getAction("loremaster.connect") is ConnectHandler)
        assertTrue(manager.getAction("loremaster.forgetConnection") is ForgetConnectionHandler)
    }
}

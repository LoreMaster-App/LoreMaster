package loremaster.connection

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

class ConnectionFormModelTest {
    @Test
    fun `each way of signing in sends only its own fields`() {
        assertEquals(Credential("apitoken", email = "a@b.c", token = "t"), ConnectionForm("x", CredentialKind.API_TOKEN, email = " a@b.c ", token = " t ", user = "u", password = "p").credential())
        assertEquals(Credential("pat", token = "t"), ConnectionForm("x", CredentialKind.PAT, email = "a@b.c", token = "t").credential())
        assertEquals(Credential("basic", user = "u", password = " p "), ConnectionForm("x", CredentialKind.BASIC, token = "t", user = " u ", password = " p ").credential())
    }

    @Test
    fun `the form says what is missing`() {
        assertEquals("Enter the Confluence address.", ConnectionForm().problem())
        assertEquals("Enter the email and the API token.", ConnectionForm("x", CredentialKind.API_TOKEN, email = "a").problem())
        assertEquals("Enter the personal access token.", ConnectionForm("x", CredentialKind.PAT).problem())
        assertEquals("Enter the user name and the password.", ConnectionForm("x", CredentialKind.BASIC, user = "u").problem())
        assertNull(ConnectionForm("x", CredentialKind.PAT, token = "t").problem())
    }

    @Test
    fun `a site written two ways shares an entry`() {
        assertEquals(normaliseAddress("https://Acme.atlassian.net/wiki/"), normaliseAddress(" https://acme.atlassian.net/wiki "))
    }
}

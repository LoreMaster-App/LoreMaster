package loremaster.outputs

import loremaster.configdescription.ChoiceDescriptor
import loremaster.configdescription.ConfigDescription
import loremaster.configdescription.FIELD_CHOICE
import loremaster.configdescription.FIELD_LIST
import loremaster.configdescription.FIELD_NUMBER
import loremaster.configdescription.FieldDescriptor
import loremaster.configdescription.PlatformDescriptor
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

/** Turning what a form holds into an output, and checking it, from the engine's description alone (no platform is named in the code). */
class OutputJsonMapperTest {
    private val direction = FieldDescriptor(
        key = "direction", label = "Direction", type = FIELD_CHOICE, default = "to-platform",
        choices = listOf(ChoiceDescriptor("to-platform", "One way"), ChoiceDescriptor("two-way", "Both ways")),
    )
    private val workspace = FieldDescriptor(key = "workspace", label = "Workspace", required = true, pattern = "[a-z]+", patternHelp = "Lower case letters.")
    private val notion = PlatformDescriptor(id = "notion", label = "Notion", fields = listOf(workspace, direction))
    private val description = ConfigDescription(
        platforms = listOf(notion, PlatformDescriptor(id = "other", label = "Other", fields = listOf(FieldDescriptor(key = "repo", label = "Repo")))),
        outputFields = listOf(FieldDescriptor(key = "name", label = "Name"), FieldDescriptor(key = "include", label = "Include", type = FIELD_LIST)),
    )

    @Test
    fun `a blank output of a platform it was never taught has every setting empty, choices on their default and the workspace as content`() {
        val output = blankOutput(description, notion)

        assertEquals("notion", output.get("platform").asString)
        assertEquals("", output.get("workspace").asString)
        assertEquals("", output.get("repo").asString)
        assertEquals("to-platform", output.get("direction").asString)
        assertFalse(output.has("name") || output.has("include"))
        assertEquals(".", output.getAsJsonArray("content")[0].asJsonObject.getAsJsonArray("roots")[0].asString)
    }

    @Test
    fun `a number field is written as a number, or left out when empty or not a number`() {
        val width = FieldDescriptor(key = "maxWidth", label = "Widest", type = FIELD_NUMBER)
        val sized = description.copy(outputFields = description.outputFields + width)

        assertFalse(blankOutput(sized, notion).has("maxWidth"))
        assertEquals(640, outputFrom(sized, notion, mapOf("workspace" to "acme", "maxWidth" to " 640 ")).get("maxWidth").asInt)
        assertFalse(outputFrom(sized, notion, mapOf("workspace" to "acme", "maxWidth" to "")).has("maxWidth"))
        assertFalse(outputFrom(sized, notion, mapOf("workspace" to "acme", "maxWidth" to "wide")).has("maxWidth"))
    }

    @Test
    fun `the values of a form become the output, trimmed, with lists as lists and a blank name left out`() {
        val output = outputFrom(description, notion, mapOf("workspace" to " acme ", "direction" to "two-way", "name" to "  ", "include" to "a.md, b/ ,", "roots" to "docs, guides", "excludes" to "tmp/"))

        assertEquals("acme", output.get("workspace").asString)
        assertEquals("two-way", output.get("direction").asString)
        assertFalse(output.has("name"))
        assertEquals(listOf("a.md", "b/"), output.getAsJsonArray("include").map { it.asString })
        val content = output.getAsJsonArray("content")[0].asJsonObject
        assertEquals(listOf("docs", "guides"), content.getAsJsonArray("roots").map { it.asString })
        assertEquals(listOf("tmp/"), content.getAsJsonArray("excludes").map { it.asString })
    }

    @Test
    fun `a choice the form did not give keeps its default, and one given blank takes it too`() {
        assertEquals("to-platform", outputFrom(description, notion, mapOf("workspace" to "acme")).get("direction").asString)
        assertEquals("to-platform", outputFrom(description, notion, mapOf("workspace" to "acme", "direction" to " ")).get("direction").asString)
    }

    @Test
    fun `settings of other platforms stay empty`() {
        assertEquals("", outputFrom(description, notion, mapOf("repo" to "ignored", "workspace" to "acme")).get("repo").asString)
    }

    @Test
    fun `a required field and a pattern are checked from the description`() {
        assertEquals(mapOf("workspace" to "Workspace is required."), validateOutputValues(description, notion, mapOf("workspace" to " ")))
        assertEquals(mapOf("workspace" to "Lower case letters."), validateOutputValues(description, notion, mapOf("workspace" to "Acme1")))
        assertTrue(validateOutputValues(description, notion, mapOf("workspace" to "acme")).isEmpty())
    }
}

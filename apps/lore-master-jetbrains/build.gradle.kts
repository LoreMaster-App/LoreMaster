import org.jetbrains.intellij.platform.gradle.TestFrameworkType

plugins {
    id("org.jetbrains.kotlin.jvm") version "2.1.21"
    id("org.jetbrains.intellij.platform") version "2.7.0"
}

group = "app.loremaster"
version = providers.gradleProperty("pluginVersion").getOrElse("0.0.0")

repositories {
    mavenCentral()
    intellijPlatform { defaultRepositories() }
}

dependencies {
    intellijPlatform {
        intellijIdeaCommunity("2025.1")
        testFramework(TestFrameworkType.Platform)
    }
    testImplementation("junit:junit:4.13.2")
    testImplementation(kotlin("test"))
}

kotlin { jvmToolchain(21) }

intellijPlatform {
    pluginConfiguration {
        name = "LoreMaster"
        ideaVersion { sinceBuild = "251" }
        changeNotes = "First preview: export a Markdown file to HTML, Word, OpenDocument, PDF, PNG, plain text, Jira and MediaWiki, from the editor, the project view or the Tools menu."
    }
    publishing {
        token = providers.environmentVariable("JETBRAINS_MARKETPLACE_TOKEN")
    }
}

// One plugin zip serves every platform, so it carries the engine built for each (dist/platforms/lore-master-engine/<os>-<arch>/).
// Build them with `VERSION=<v> npx nx run lore-master-engine:build-all`, or point -PengineDir somewhere else.
val engineDir = providers.gradleProperty("engineDir")
    .orElse(layout.projectDirectory.dir("../../dist/platforms/lore-master-engine").asFile.path)

tasks.prepareSandbox {
    from(engineDir) {
        into("${pluginName.get()}/bin")
    }
}

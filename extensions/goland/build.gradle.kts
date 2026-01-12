plugins {
    id("org.jetbrains.kotlin.jvm") version "1.9.21"
    id("org.jetbrains.intellij") version "1.16.1"
}

group = "dev.unqueryvet"
version = "1.0.0"

repositories {
    mavenCentral()
}

intellij {
    version.set("2023.3")
    type.set("GO") // GoLand
    plugins.set(listOf("org.jetbrains.plugins.go"))
}

tasks {
    patchPluginXml {
        sinceBuild.set("233")
        untilBuild.set("253.*")
    }

    buildSearchableOptions {
        enabled = false
    }

    signPlugin {
        certificateChain.set(System.getenv("CERTIFICATE_CHAIN"))
        privateKey.set(System.getenv("PRIVATE_KEY"))
        password.set(System.getenv("PRIVATE_KEY_PASSWORD"))
    }

    publishPlugin {
        token.set(System.getenv("PUBLISH_TOKEN"))
        val channel = project.findProperty("pluginChannel")?.toString() ?: "default"
        // "default" channel means stable release (no channel specified)
        if (channel != "default") {
            channels.set(listOf(channel))
        }
    }
}

kotlin {
    jvmToolchain(17)
}

dependencies {
    implementation("com.google.code.gson:gson:2.10.1")
}

# lore-master-jetbrains

The JetBrains shell (IntelliJ IDEA, WebStorm, PyCharm, GoLand, Rider, ...) over the same `lore-master-engine` sidecar the
VS Code extension uses. Epic: #433. Design: `docs/architecture/engine-and-shells.md`.

Kotlin, Gradle (IntelliJ Platform Gradle Plugin 2.x), JDK 21. Files are `<snake>_<role>.kt`, one package per slice under
`src/main/kotlin/loremaster/`.

| Slice | What it does |
|---|---|
| `engineprocess` | Finds the engine binary, frames JSON-RPC over its stdio, matches responses to requests |

```bash
cd apps/lore-master-jetbrains
./gradlew test                                       # unit tests
LORE_MASTER_ENGINE=<path to a built engine> ./gradlew test   # also pings the real engine
./gradlew runIde                                     # a sandbox IDE with the plugin
```

Not built yet: secret storage, settings, sync, the Pages and Generators tool windows, export actions, Mermaid through JCEF,
Marketplace publishing, and a CI job (the runner needs a JDK).

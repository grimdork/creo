# Java / Kotlin / Gradle

Initialise a JVM project:

```sh
$ creo -i java         # defaults to Kotlin + Gradle
$ creo -i kotlin       # same
$ creo -i gradle       # same
```

This creates `settings.gradle.kts`, `build.gradle.kts` (Kotlin DSL),
`src/main/kotlin/<pkg>/App.kt`, and a `fiat` file.

## Build tool detection

The build tool is detected in this order:

| Detection | Tool used |
|---|---|
| `gradlew` exists | `./gradlew` |
| `mvnw` exists | `./mvnw` |
| `build.gradle.kts` or `build.gradle` exists | `gradle` |
| `pom.xml` exists | `mvn` |
| (none found) | `gradle` |

## Defaults

### Gradle

| Property | Value |
|---|---|
| `bin=` | `build/libs` — Gradle's jar output directory |
| `cmd=` | `$GRADLE build` |
| `sources=` | `*.java *.kt build.gradle.kts build.gradle settings.gradle.kts settings.gradle` |

### Maven

| Property | Value |
|---|---|
| `bin=` | `target` — Maven's output directory |
| `cmd=` | `$MVN package` |
| `sources=` | `*.java *.kt pom.xml` |

## Variables

| Variable | Default |
|---|---|
| `$GRADLE` | `./gradlew`, `gradle`, or as detected |
| `$MVN` | `./mvnw`, `mvn`, or as detected |
| `$JAVA` | `java` |

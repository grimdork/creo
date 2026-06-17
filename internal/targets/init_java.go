package targets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grimdork/creo/internal/fiat"
)

// InitJava scaffolds a Java/Kotlin project with Gradle build files and a basic source.
func InitJava(dir string, force, verbose bool) ([]string, error) {
	file, err := ensureFiat(dir)
	if err != nil {
		return nil, err
	}

	if fiat.FindTarget(file, "build") == nil {
		bt := &fiat.Target{
			Name:     "build",
			Language: "java",
			Desc:     "Build the Java/Kotlin project",
		}
		file.AddTarget(bt)
	}

	_, proj := absDirName(dir)
	pkg := "com." + strings.ToLower(proj)
	pkgPath := strings.ReplaceAll(pkg, ".", "/")

	settings := `rootProject.name = "` + proj + `"
`
	if err := tryWrite(filepath.Join(dir, "settings.gradle.kts"), settings,
		force, verbose, "settings.gradle.kts"); err != nil {
		return nil, err
	}

	buildGradle := `plugins {
    kotlin("jvm") version "2.0.0"
    application
}

application {
    mainClass = "` + pkg + `.AppKt"
}

repositories {
    mavenCentral()
}

dependencies {
    implementation(kotlin("stdlib"))
}
`
	if err := tryWrite(filepath.Join(dir, "build.gradle.kts"), buildGradle,
		force, verbose, "build.gradle.kts"); err != nil {
		return nil, err
	}

	klassDir := filepath.Join(dir, "src", "main", "kotlin", pkgPath)
	if err := os.MkdirAll(klassDir, 0755); err != nil {
		return nil, fmt.Errorf(errCreating, "src/main/kotlin/"+pkgPath, err)
	}

	appContent := `package ` + pkg + `

fun main() {
    println("hello from ` + proj + `")
}
`
	if err := tryWrite(filepath.Join(klassDir, "App.kt"), appContent,
		force, verbose, "src/main/kotlin/"+pkgPath+"/App.kt"); err != nil {
		return nil, err
	}

	if err := file.Write(); err != nil {
		return nil, err
	}

	return []string{"build/", ".gradle/", "*.jar", "/.creo"}, nil
}

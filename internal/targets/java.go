package targets

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/grimdork/creo/internal/fiat"
)

// JavaProjectName reads the project name from settings.gradle.kts, settings.gradle or pom.xml.
func JavaProjectName(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "settings.gradle.kts"))
	if err != nil {
		data, err = os.ReadFile(filepath.Join(dir, "settings.gradle"))
		if err != nil {
			data, err = os.ReadFile(filepath.Join(dir, "pom.xml"))
			if err != nil {
				return filepath.Base(dir)
			}
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "<artifactId>") {
					start := strings.Index(line, ">")
					end := strings.LastIndex(line, "<")
					if start >= 0 && end > start {
						return line[start+1 : end]
					}
				}
			}
			return filepath.Base(dir)
		}
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "rootProject.name") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[1])
				name = strings.Trim(name, `"`)
				return name
			}
		}
	}
	return filepath.Base(dir)
}

func detectBuildTool(dir string) string {
	if _, err := os.Stat(filepath.Join(dir, "gradlew")); err == nil {
		return "./gradlew"
	}
	if _, err := os.Stat(filepath.Join(dir, "mvnw")); err == nil {
		return "./mvnw"
	}
	if _, err := os.Stat(filepath.Join(dir, "build.gradle.kts")); err == nil {
		return "gradle"
	}
	if _, err := os.Stat(filepath.Join(dir, "build.gradle")); err == nil {
		return "gradle"
	}
	if _, err := os.Stat(filepath.Join(dir, "pom.xml")); err == nil {
		return "mvn"
	}
	return "gradle"
}

func applyJava(f *fiat.File, t *fiat.Target) {
	absDir := absDir(f)

	proj := JavaProjectName(absDir)
	setDefaultVar(f.Vars, "PROJECT", proj)
	setDefaultVar(f.Vars, "JAVA", "java")

	bt := detectBuildTool(absDir)
	if strings.Contains(bt, "gradle") || bt == "./gradlew" {
		setDefaultVar(f.Vars, "GRADLE", bt)
		if t.Sources == "" {
			t.Sources = "*.java *.kt build.gradle.kts build.gradle settings.gradle.kts settings.gradle"
		}
		t.Bin = expandBin(f, t, "build/libs")
		if len(t.Cmds) == 0 {
			t.Cmds = append(t.Cmds, "$GRADLE build")
		}
	} else {
		setDefaultVar(f.Vars, "MVN", bt)
		if t.Sources == "" {
			t.Sources = "*.java *.kt pom.xml"
		}
		t.Bin = expandBin(f, t, "target")
		if len(t.Cmds) == 0 {
			t.Cmds = append(t.Cmds, "$MVN package")
		}
	}
}

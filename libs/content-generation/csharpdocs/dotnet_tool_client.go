package csharpdocs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"lore-master/libs/content-generation/externaltool"
)

// Runner runs an external tool. The generator takes one so that tests need no .NET.
type Runner func(ctx context.Context, command externaltool.Command) (externaltool.Result, error)

const (
	buildTimeout = 15 * time.Minute
	docTimeout   = 10 * time.Minute

	dotnetHint = "install the .NET SDK (https://dot.net) and run the generator again"
	toolHint   = "install it with: dotnet tool install -g DefaultDocumentation.Console"
)

// builtProject is where a build left the assembly and its XML documentation.
type builtProject struct {
	Assembly string
	XML      string
}

// buildProject builds the project in Release with the documentation file switched on, so the
// XML is as fresh as the code (an up-to-date project builds in moments), and asks MSBuild where
// the assembly and the XML went (-getProperty alone only evaluates, so the Build target is named). A project that targets several frameworks is built for the
// last one. A failed build is an error carrying the end of the compiler's output; a build that
// left no XML file says so.
func buildProject(ctx context.Context, runner Runner, projectDir string, projectFile string) (builtProject, error) {
	properties, err := runBuild(ctx, runner, projectDir, projectFile, "")
	if err != nil {
		return builtProject{}, err
	}
	if properties.TargetPath == "" && properties.TargetFrameworks != "" {
		if properties, err = runBuild(ctx, runner, projectDir, projectFile, highestFramework(properties.TargetFrameworks)); err != nil {
			return builtProject{}, err
		}
	}
	if properties.TargetPath == "" {
		return builtProject{}, errors.New("the build did not report an assembly")
	}

	built := builtProject{Assembly: properties.TargetPath, XML: resolveFrom(projectDir, properties.DocumentationFile)}
	if _, err := os.Stat(built.Assembly); err != nil {
		return builtProject{}, fmt.Errorf("the build reported %s but it is not there", built.Assembly)
	}
	if built.XML == "" || !fileExists(built.XML) {
		return builtProject{}, fmt.Errorf("the build produced no XML documentation file (expected %s); set <GenerateDocumentationFile>true</GenerateDocumentationFile> in %s", built.XML, projectFile)
	}

	return built, nil
}

type buildProperties struct {
	TargetPath        string
	DocumentationFile string
	TargetFrameworks  string
}

func runBuild(ctx context.Context, runner Runner, projectDir string, projectFile string, framework string) (buildProperties, error) {
	args := []string{"build", projectFile, "-c", "Release", "-nologo", "-v", "q", "-t:Build", "-p:GenerateDocumentationFile=true",
		"-getProperty:TargetPath", "-getProperty:DocumentationFile", "-getProperty:TargetFrameworks"}
	if framework != "" {
		args = append(args, "-p:TargetFramework="+framework)
	}
	result, err := runner(ctx, externaltool.Command{Name: "dotnet", Args: args, Dir: projectDir, Timeout: buildTimeout})
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return buildProperties{}, &externaltool.MissingToolError{Tool: "dotnet", Hint: dotnetHint}
		}

		return buildProperties{}, err
	}
	if failure := result.Failure("dotnet build"); failure != "" {
		return buildProperties{}, errors.New(failure)
	}

	var reply struct {
		Properties buildProperties `json:"Properties"`
	}
	start := strings.Index(result.Stdout, "{")
	if start < 0 || json.Unmarshal([]byte(result.Stdout[start:]), &reply) != nil {
		return buildProperties{}, errors.New("dotnet build did not report the project's properties; the .NET SDK 8 or newer is needed")
	}

	return reply.Properties, nil
}

// resolveFrom makes a path from MSBuild absolute: it reports the documentation file relative to
// the project.
func resolveFrom(projectDir string, p string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}

	return filepath.Join(projectDir, filepath.FromSlash(strings.ReplaceAll(p, "\\", "/")))
}

// runDefaultDocumentation writes one page per namespace and per type (members stay on their
// type's page) into outDir, and returns the pages by file name.
func runDefaultDocumentation(ctx context.Context, runner Runner, projectDir string, built builtProject, outDir string) (map[string]string, error) {
	result, err := runner(ctx, externaltool.Command{
		Name: "defaultdocumentation",
		Args: []string{"-a", built.Assembly, "-d", built.XML, "-o", outDir, "-g", "Namespaces,Types"},
		Dir:  projectDir, Timeout: docTimeout,
	})
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, &externaltool.MissingToolError{Tool: "defaultdocumentation", Hint: toolHint}
		}

		return nil, err
	}
	if failure := result.Failure("defaultdocumentation"); failure != "" {
		return nil, errors.New(failure)
	}

	pages := map[string]string{}
	err = filepath.WalkDir(outDir, func(full string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || !strings.EqualFold(filepath.Ext(full), ".md") {
			return walkErr
		}
		content, err := os.ReadFile(full)
		if err != nil {
			return err
		}
		pages[entry.Name()] = string(content)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading what DefaultDocumentation wrote: %w", err)
	}

	return pages, nil
}

func fileExists(full string) bool {
	info, err := os.Stat(full)

	return err == nil && !info.IsDir()
}

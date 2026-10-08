package csharpdocs

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"lore-master/libs/content-generation/externaltool"
	"lore-master/libs/content-generation/generatedfile"
)

func put(t *testing.T, root string, relative string, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func installed(string) bool { return true }

// fakeDotnet stands in for dotnet and DefaultDocumentation: a build leaves an assembly and its
// XML in the project folder and reports them the way MSBuild does; the documentation tool
// writes what the real one wrote for the sample project (testdata).
type fakeDotnet struct {
	commands []externaltool.Command
	// frameworks makes the first build report several target frameworks and no assembly.
	frameworks string
	// noXML makes a build leave no documentation file.
	noXML bool
	// failBuild makes dotnet exit non-zero.
	failBuild bool
	// failTool makes DefaultDocumentation exit non-zero.
	failTool bool
}

func (f *fakeDotnet) run(_ context.Context, command externaltool.Command) (externaltool.Result, error) {
	f.commands = append(f.commands, command)
	if command.Name == "dotnet" {
		return f.build(command)
	}
	if f.failTool {
		return externaltool.Result{ExitCode: 1, Stderr: "Unhandled exception"}, nil
	}
	out := command.Args[slices.Index(command.Args, "-o")+1]
	entries, err := os.ReadDir("testdata/default-documentation")
	if err != nil {
		return externaltool.Result{}, err
	}
	for _, entry := range entries {
		content, err := os.ReadFile(filepath.Join("testdata/default-documentation", entry.Name()))
		if err != nil {
			return externaltool.Result{}, err
		}
		if err := os.WriteFile(filepath.Join(out, entry.Name()), content, 0o644); err != nil {
			return externaltool.Result{}, err
		}
	}

	return externaltool.Result{Stdout: "- DefaultDocumentation.Internal.DocItemGenerators.OwnPageSetter\n"}, nil
}

func (f *fakeDotnet) build(command externaltool.Command) (externaltool.Result, error) {
	if f.failBuild {
		return externaltool.Result{ExitCode: 1, Stdout: "Cart.cs(3,1): error CS1002: ; expected"}, nil
	}
	reply := func(properties map[string]string) (externaltool.Result, error) {
		encoded, _ := json.MarshalIndent(map[string]any{"Properties": properties}, "", "  ")

		return externaltool.Result{Stdout: string(encoded)}, nil
	}
	if f.frameworks != "" && !slices.Contains(command.Args, "-p:TargetFramework=net8.0") {
		return reply(map[string]string{"TargetPath": "", "DocumentationFile": "", "TargetFrameworks": f.frameworks})
	}
	assembly := filepath.Join(command.Dir, "bin", "Release", "net8.0", "Shop.dll")
	if err := os.MkdirAll(filepath.Dir(assembly), 0o755); err != nil {
		return externaltool.Result{}, err
	}
	if err := os.WriteFile(assembly, []byte("x"), 0o644); err != nil {
		return externaltool.Result{}, err
	}
	xml := `obj\Release\net8.0\Shop.xml`
	if !f.noXML {
		if err := os.MkdirAll(filepath.Join(command.Dir, "obj", "Release", "net8.0"), 0o755); err != nil {
			return externaltool.Result{}, err
		}
		if err := os.WriteFile(filepath.Join(command.Dir, "obj", "Release", "net8.0", "Shop.xml"), []byte("x"), 0o644); err != nil {
			return externaltool.Result{}, err
		}
	}

	return reply(map[string]string{"TargetPath": assembly, "DocumentationFile": xml, "TargetFrameworks": ""})
}

func shopWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	put(t, root, "Shop/Shop.csproj", "x")

	return root
}

func paths(files []generatedfile.File) []string {
	var out []string
	for _, file := range files {
		out = append(out, file.Path)
	}
	slices.Sort(out)

	return out
}

func bodyOf(t *testing.T, files []generatedfile.File, path string) string {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			return string(file.Body)
		}
	}
	t.Fatalf("no %s in %v", path, paths(files))

	return ""
}

func TestNamespacesNestAsFoldersAndTypesAreFilesInThem(t *testing.T) {
	output, err := generateWith(context.Background(), (&fakeDotnet{}).run, installed, shopWorkspace(t), generatedfile.Spec{})

	if err != nil {
		t.Fatal(err)
	}
	want := []string{"README.md", "Shop/Shop/README.md", "Shop/Shop/Billing/Invoice.md", "Shop/Shop/Billing/README.md", "Shop/Shop/Cart.md"}
	slices.Sort(want)
	if got := paths(output.Files); !slices.Equal(got, want) {
		t.Fatalf("paths %v, want %v", got, want)
	}
	cart := bodyOf(t, output.Files, "Shop/Shop/Cart.md")
	for _, wanted := range []string{"# Shop.Cart Class", "public int Add(string sku, int quantity=1);", "#### Returns"} {
		if !strings.Contains(cart, wanted) {
			t.Errorf("Cart.md lacks %q:\n%s", wanted, cart)
		}
	}
	if strings.Contains(cart, "<a name=") || strings.Contains(cart, "index.md") || strings.HasPrefix(cart, "####") {
		t.Fatalf("anchors or breadcrumbs left in:\n%s", cart)
	}
}

func TestLinksBetweenPagesFollowThemToTheirNewPlace(t *testing.T) {
	output, err := generateWith(context.Background(), (&fakeDotnet{}).run, installed, shopWorkspace(t), generatedfile.Spec{})
	if err != nil {
		t.Fatal(err)
	}

	namespace := bodyOf(t, output.Files, "Shop/Shop/README.md")
	billing := bodyOf(t, output.Files, "Shop/Shop/Billing/README.md")
	invoice := bodyOf(t, output.Files, "Shop/Shop/Billing/Invoice.md")
	if !strings.Contains(namespace, "[Cart](Cart.md)") {
		t.Errorf("namespace page:\n%s", namespace)
	}
	if !strings.Contains(billing, "[Invoice](Invoice.md)") {
		t.Errorf("billing page:\n%s", billing)
	}
	if !strings.Contains(invoice, "[Invoice](Invoice.md)") || !strings.Contains(invoice, "learn.microsoft.com/en-us/dotnet/api/system.object") {
		t.Errorf("invoice page:\n%s", invoice)
	}
	for _, file := range output.Files {
		if strings.Contains(string(file.Body), "Shop.Cart.md") || strings.Contains(string(file.Body), "Shop.Billing.md") {
			t.Errorf("%s still links to a flat file name", file.Path)
		}
	}
}

func TestEveryPageHasAUniqueTitleAndTheIndexListsTheNamespaces(t *testing.T) {
	output, err := generateWith(context.Background(), (&fakeDotnet{}).run, installed, shopWorkspace(t), generatedfile.Spec{Title: "Shop API"})
	if err != nil {
		t.Fatal(err)
	}

	seen := map[string]string{}
	for _, file := range output.Files {
		title, _, _ := strings.Cut(strings.TrimPrefix(string(file.Body), "# "), "\n")
		if other, clash := seen[title]; clash {
			t.Errorf("%s and %s are both titled %q", other, file.Path, title)
		}
		seen[title] = file.Path
	}
	index := bodyOf(t, output.Files, "README.md")
	if !strings.HasPrefix(index, "# Shop API\n") || !strings.Contains(index, "(Shop/Shop/README.md)") || !strings.Contains(index, "(Shop/Shop/Billing/README.md)") {
		t.Fatalf("index:\n%s", index)
	}
}

func TestTheProjectIsBuiltInReleaseThenDocumentedFromItsAssemblyAndXML(t *testing.T) {
	double := &fakeDotnet{}
	root := shopWorkspace(t)

	if _, err := generateWith(context.Background(), double.run, installed, root, generatedfile.Spec{}); err != nil {
		t.Fatal(err)
	}

	if len(double.commands) != 2 || double.commands[0].Name != "dotnet" || double.commands[1].Name != "defaultdocumentation" {
		t.Fatalf("commands %+v", double.commands)
	}
	build := strings.Join(double.commands[0].Args, " ")
	for _, wanted := range []string{"build Shop.csproj", "-c Release", "-t:Build", "-p:GenerateDocumentationFile=true", "-getProperty:TargetPath"} {
		if !strings.Contains(build, wanted) {
			t.Errorf("build lacks %q: %s", wanted, build)
		}
	}
	tool := double.commands[1].Args
	if tool[slices.Index(tool, "-a")+1] != filepath.Join(root, "Shop", "bin", "Release", "net8.0", "Shop.dll") ||
		tool[slices.Index(tool, "-d")+1] != filepath.Join(root, "Shop", "obj", "Release", "net8.0", "Shop.xml") ||
		tool[slices.Index(tool, "-g")+1] != "Namespaces,Types" {
		t.Fatalf("tool arguments %v", tool)
	}
}

func TestAMultiTargetingProjectIsBuiltForItsLastFramework(t *testing.T) {
	double := &fakeDotnet{frameworks: "net6.0;net8.0"}

	if _, err := generateWith(context.Background(), double.run, installed, shopWorkspace(t), generatedfile.Spec{}); err != nil {
		t.Fatal(err)
	}

	if len(double.commands) != 3 || !slices.Contains(double.commands[1].Args, "-p:TargetFramework=net8.0") {
		t.Fatalf("commands %+v", double.commands)
	}
}

func TestAMissingToolStopsTheRunBeforeAnythingIsBuilt(t *testing.T) {
	double := &fakeDotnet{}

	_, err := generateWith(context.Background(), double.run, func(string) bool { return false }, shopWorkspace(t), generatedfile.Spec{})

	var missing *externaltool.MissingToolError
	if !errors.As(err, &missing) || missing.Tool != "defaultdocumentation" || !strings.Contains(err.Error(), "dotnet tool install -g DefaultDocumentation.Console") {
		t.Fatalf("error %v", err)
	}
	if len(double.commands) != 0 {
		t.Fatalf("something ran: %+v", double.commands)
	}
}

func TestAMissingDotnetSaysToInstallTheSDK(t *testing.T) {
	missingDotnet := func(context.Context, externaltool.Command) (externaltool.Result, error) {
		return externaltool.Result{}, exec.ErrNotFound
	}

	_, err := generateWith(context.Background(), missingDotnet, installed, shopWorkspace(t), generatedfile.Spec{})

	var missing *externaltool.MissingToolError
	if !errors.As(err, &missing) || missing.Tool != "dotnet" || !strings.Contains(err.Error(), "install the .NET SDK") {
		t.Fatalf("error %v", err)
	}
}

func TestABuildThatLeavesNoXMLSaysHowToSwitchItOn(t *testing.T) {
	_, err := generateWith(context.Background(), (&fakeDotnet{noXML: true}).run, installed, shopWorkspace(t), generatedfile.Spec{})

	if err == nil || !strings.Contains(err.Error(), "no XML documentation file") || !strings.Contains(err.Error(), "GenerateDocumentationFile") || !strings.Contains(err.Error(), "Shop.csproj") {
		t.Fatalf("error %v", err)
	}
}

func TestAFailedBuildOrToolIsAnErrorNamingTheProject(t *testing.T) {
	_, err := generateWith(context.Background(), (&fakeDotnet{failBuild: true}).run, installed, shopWorkspace(t), generatedfile.Spec{})
	if err == nil || !strings.Contains(err.Error(), "building Shop/Shop.csproj") || !strings.Contains(err.Error(), "error CS1002") {
		t.Fatalf("build error %v", err)
	}

	_, err = generateWith(context.Background(), (&fakeDotnet{failTool: true}).run, installed, shopWorkspace(t), generatedfile.Spec{})
	if err == nil || !strings.Contains(err.Error(), "documenting Shop") || !strings.Contains(err.Error(), "Unhandled exception") {
		t.Fatalf("tool error %v", err)
	}
}

func TestTestProjectsAreSkippedAndInputNarrowsTheRest(t *testing.T) {
	root := t.TempDir()
	for _, project := range []string{"src/Shop/Shop.csproj", "src/Legacy/Legacy.csproj", "tests/Shop.Tests/Shop.Tests.csproj"} {
		put(t, root, project, "x")
	}
	double := &fakeDotnet{}

	_, err := generateWith(context.Background(), double.run, installed, root, generatedfile.Spec{Input: []string{"src/", "!src/Legacy/"}})

	if err != nil {
		t.Fatal(err)
	}
	if len(double.commands) != 2 || filepath.Base(double.commands[0].Dir) != "Shop" {
		t.Fatalf("commands %+v", double.commands)
	}
}

func TestAWorkspaceWithNoProjectGetsAnIndexSayingSoAndNeedsNoTool(t *testing.T) {
	output, err := generateWith(context.Background(), (&fakeDotnet{}).run, func(string) bool { return false }, t.TempDir(), generatedfile.Spec{})

	if err != nil || len(output.Files) != 1 || !strings.Contains(bodyOf(t, output.Files, "README.md"), "No C# projects") {
		t.Fatalf("err %v files %v", err, paths(output.Files))
	}
}

func TestTestProjectNames(t *testing.T) {
	for name, want := range map[string]bool{
		"Shop.csproj": false, "Shop.Tests.csproj": true, "Shop.Test.csproj": true, "Shop.UnitTests.csproj": true,
		"Tests.csproj": true, "Shop.Tests.Integration.csproj": true, "Contest.csproj": false, "Attestation.csproj": false,
	} {
		if got := isTestProject(name); got != want {
			t.Errorf("isTestProject(%q) = %v, want %v", name, got, want)
		}
	}
}

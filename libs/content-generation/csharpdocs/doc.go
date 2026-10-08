// Package csharpdocs is the csharp-docs generator: it builds each C# project with dotnet (so the
// XML documentation file is fresh), runs DefaultDocumentation over the assembly and that XML,
// and writes a page per namespace and per type under the output folder, namespaces nested as
// folders. DefaultDocumentation is expected as a dotnet tool; when it is not there the generator
// says what to install.
package csharpdocs

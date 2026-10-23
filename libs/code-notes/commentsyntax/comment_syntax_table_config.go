package commentsyntax

var (
	cStyle    = Syntax{Line: []string{"//"}, Block: []BlockPair{{"/*", "*/"}}, Quotes: "\"'`"}
	hashStyle = Syntax{Line: []string{"#"}, Quotes: "\"'"}
	htmlStyle = Syntax{Block: []BlockPair{{"<!--", "-->"}}}
	cssStyle  = Syntax{Block: []BlockPair{{"/*", "*/"}}, Quotes: "\"'"}
	sqlStyle  = Syntax{Line: []string{"--"}, Block: []BlockPair{{"/*", "*/"}}, Quotes: "'\""}
	luaStyle  = Syntax{Block: []BlockPair{{"--[[", "]]"}}, Line: []string{"--"}, Quotes: "\"'"}
	semicolon = Syntax{Line: []string{";"}, Quotes: "\""}
	percent   = Syntax{Line: []string{"%"}, Quotes: "\""}
	python    = Syntax{Line: []string{"#"}, Block: []BlockPair{{`"""`, `"""`}, {"'''", "'''"}}, Quotes: "\"'"}
	powershel = Syntax{Line: []string{"#"}, Block: []BlockPair{{"<#", "#>"}}, Quotes: "\"'"}
	vueStyle  = Syntax{Line: []string{"//"}, Block: []BlockPair{{"/*", "*/"}, {"<!--", "-->"}}, Quotes: "\"'`"}
	php       = Syntax{Line: []string{"//", "#"}, Block: []BlockPair{{"/*", "*/"}}, Quotes: "\"'"}
	scssStyle = Syntax{Line: []string{"//"}, Block: []BlockPair{{"/*", "*/"}}, Quotes: "\"'"}
)

// byExtension is the syntax of each file extension (lower case, with the dot).
var byExtension = map[string]Syntax{
	".c": cStyle, ".h": cStyle, ".cc": cStyle, ".cpp": cStyle, ".cxx": cStyle, ".hpp": cStyle, ".cs": cStyle, ".java": cStyle,
	".js": cStyle, ".jsx": cStyle, ".mjs": cStyle, ".cjs": cStyle, ".ts": cStyle, ".tsx": cStyle, ".mts": cStyle, ".cts": cStyle,
	".go": cStyle, ".rs": cStyle, ".swift": cStyle, ".kt": cStyle, ".kts": cStyle, ".dart": cStyle, ".scala": cStyle,
	".groovy": cStyle, ".gradle": cStyle, ".proto": cStyle, ".m": cStyle, ".mm": cStyle, ".jsonc": cStyle, ".json5": cStyle,
	".zig": cStyle, ".v": cStyle, ".sol": cStyle,
	".php": php,
	".vue": vueStyle, ".svelte": vueStyle, ".astro": vueStyle,
	".css": cssStyle, ".scss": scssStyle, ".less": scssStyle,
	".py": python, ".pyi": python, ".pyw": python,
	".sh": hashStyle, ".bash": hashStyle, ".zsh": hashStyle, ".fish": hashStyle, ".yml": hashStyle, ".yaml": hashStyle,
	".toml": hashStyle, ".rb": hashStyle, ".pl": hashStyle, ".pm": hashStyle, ".r": hashStyle, ".mk": hashStyle,
	".tf": hashStyle, ".nix": hashStyle, ".cmake": hashStyle, ".dockerfile": hashStyle, ".gitignore": hashStyle, ".env": hashStyle,
	".ex": hashStyle, ".exs": hashStyle, ".jl": hashStyle, ".cr": hashStyle, ".coffee": hashStyle,
	".ps1": powershel, ".psm1": powershel,
	".sql": sqlStyle, ".hs": sqlStyle, ".lua": luaStyle,
	".html": htmlStyle, ".htm": htmlStyle, ".xml": htmlStyle, ".svg": htmlStyle, ".md": htmlStyle, ".markdown": htmlStyle,
	".xhtml": htmlStyle, ".csproj": htmlStyle, ".props": htmlStyle, ".xaml": htmlStyle, ".mdx": htmlStyle,
	".ini": semicolon, ".cfg": semicolon, ".clj": semicolon, ".cljs": semicolon, ".lisp": semicolon, ".el": semicolon, ".asm": semicolon,
	".tex": percent, ".erl": percent, ".mat": percent,
}

// byName is the syntax of files known by their whole name.
var byName = map[string]Syntax{
	"Dockerfile": hashStyle, "Makefile": hashStyle, "makefile": hashStyle, "Gemfile": hashStyle, "Rakefile": hashStyle,
	"Jenkinsfile": cStyle, "CMakeLists.txt": hashStyle, ".bashrc": hashStyle, ".zshrc": hashStyle, ".editorconfig": hashStyle,
}

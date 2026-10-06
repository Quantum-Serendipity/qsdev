package devenv

import (
	"slices"
	"sort"
	"strings"
	"testing"
	"text/template"
	"text/template/parse"

	"github.com/Quantum-Serendipity/qsdev/internal/tmpl"
)

// rawCodeSinks is the reviewed allowlist of devenv.nix.tmpl actions that emit
// a template field as Nix code without an escaping function. Each value is
// Nix code, which no validator can tell good from bad, so the guard is
// provenance: none of them can carry repository content. Keys are the
// action's enclosing range/if/else blocks and its pipeline, as walkSinks
// prints them; values are the one-line provenance justification.
var rawCodeSinks = map[string]string{
	"if .Overlays > range .Overlays > .": "overlayPathExprs: each project overlay path is normalized and rendered as an escaped Nix path literal",
	"range .PackageExprs > .": "package expressions of compiled ecosystem modules and of embedded or user-scope catalog tools; " +
		"never read from the project",
	"range .LanguageFragments > .NixFragment | trimTrailingNewline": "fragments of compiled ecosystem modules and catalog tool sections; " +
		"project values reach them only through their own escaping",
	"if .Services > range .Services > range .ConfigLines > .": "lines built by the compiled service definitions (services.go); " +
		"project service settings are validated identifiers and tokens",
	"if .GitHooksEnabled > range .CustomHooks > if .RawEntry > if .NeedsToString > .Entry": "writeShellScript derivations " +
		"qsdev builds from compiled module scripts and embedded catalog entries; project hooks never set RawEntry",
	"if .GitHooksEnabled > range .CustomHooks > if .RawEntry > else .NeedsToString > .Entry": `"${<package>}/bin/<binary><args>" ` +
		"(customHookData): the package is hookPackageExpr and the binary and args are cut unescaped from the " +
		"HookConfig.Entry of a compiled ecosystem module; only a hook with a package takes this branch, and " +
		"catalog custom_hooks (and so a project overlay) cannot set one",
	"if .GitHooksEnabled > range .CustomHooks > if .PackageExpr > .PackageExpr": "hookPackageExpr: pkgs.<attr> or " +
		"config.languages.<ident>.package from compiled modules or the catalog",
}

// escapingFuncs are the template functions whose output is safe Nix for the
// value they are given anywhere in the template: each escapes it into a
// string or path literal, or rejects anything that is not a plain identifier
// or attribute path.
var escapingFuncs = map[string]bool{
	"nixString": true, "nixStringList": true, "nixMultiline": true, "nixAttrSet": true,
	"nixAttrName": true, "nixPkgList": true, "nixList": true, "nixBool": true,
	"nixHookID": true, "nixIdent": true,
}

// commentFuncs are safe only inside a Nix line comment: nixComment rejects
// only line breaks, so its output is inert after '#' and arbitrary Nix code
// anywhere else.
var commentFuncs = map[string]bool{"nixComment": true}

// layoutFuncs may follow an escaping function in a pipeline: they only
// re-indent or trim the escaped text and take no data arguments.
var layoutFuncs = map[string]bool{"indent": true, "nindent": true, "trimTrailingNewline": true}

// parseDevenvNixTemplate parses the embedded devenv.nix.tmpl with the funcs
// the renderer uses.
func parseDevenvNixTemplate(t *testing.T) *parse.Tree {
	t.Helper()
	src, err := templateFS.ReadFile("templates/devenv.nix.tmpl")
	if err != nil {
		t.Fatalf("reading embedded devenv.nix.tmpl: %v", err)
	}
	tpl, err := template.New("devenv.nix").Funcs(tmpl.NixFuncMap()).Parse(string(src))
	if err != nil {
		t.Fatalf("parsing devenv.nix.tmpl: %v", err)
	}
	return tpl.Tree
}

// sinkVisitor is called for an output action with the chain of blocks that
// encloses it (e.g. ["range .Services", "range .ConfigLines"]) and the
// template text immediately before it in the same list ("" when another
// node precedes it).
type sinkVisitor func(ctx []string, before string, a *parse.ActionNode)

// walkSinks calls visit for every output action under n.
func walkSinks(t *testing.T, n parse.Node, ctx []string, visit sinkVisitor) {
	t.Helper()
	switch n := n.(type) {
	case *parse.ListNode:
		if n == nil {
			return
		}
		before := ""
		for _, c := range n.Nodes {
			if a, ok := c.(*parse.ActionNode); ok {
				if len(a.Pipe.Decl) == 0 {
					visit(ctx, before, a)
				}
			} else {
				walkSinks(t, c, ctx, visit)
			}
			before = ""
			if tn, ok := c.(*parse.TextNode); ok {
				before = string(tn.Text)
			}
		}
	case *parse.IfNode:
		walkBranch(t, "if", &n.BranchNode, ctx, visit)
	case *parse.RangeNode:
		walkBranch(t, "range", &n.BranchNode, ctx, visit)
	case *parse.WithNode:
		walkBranch(t, "with", &n.BranchNode, ctx, visit)
	case *parse.TextNode, *parse.CommentNode:
	default:
		// {{template}}, {{block}} and anything newer would hide output from
		// this guard; reviewing one means extending the walk first.
		t.Errorf("devenv.nix.tmpl: unreviewed node %s (%T)", n, n)
	}
}

func walkBranch(t *testing.T, kind string, b *parse.BranchNode, ctx []string, visit sinkVisitor) {
	t.Helper()
	pipe := b.Pipe.String()
	walkSinks(t, b.List, append(slices.Clone(ctx), kind+" "+pipe), visit)
	if b.ElseList != nil {
		walkSinks(t, b.ElseList, append(slices.Clone(ctx), "else "+pipe), visit)
	}
}

// escapedPipeline reports whether p, emitted after the template text before,
// starts with an escaping function and is followed only by layout functions
// with literal arguments. A comment function counts only when the action
// sits on a line that is a Nix comment up to it.
func escapedPipeline(p *parse.PipeNode, before string) bool {
	if len(p.Cmds) == 0 {
		return false
	}
	switch first := p.Cmds[0]; {
	case isCallTo(first, escapingFuncs):
	case isCallTo(first, commentFuncs) && inLineComment(before):
	default:
		return false
	}
	for _, c := range p.Cmds[1:] {
		if !isCallTo(c, layoutFuncs) {
			return false
		}
		for _, arg := range c.Args[1:] {
			switch arg.(type) {
			case *parse.NumberNode, *parse.StringNode:
			default:
				return false
			}
		}
	}
	return true
}

// inLineComment reports whether text emitted right after before is inside a
// Nix line comment: the last line of before, less its indentation, starts
// with '#'.
func inLineComment(before string) bool {
	line := before[strings.LastIndexByte(before, '\n')+1:]
	return strings.HasPrefix(strings.TrimLeft(line, " \t"), "#")
}

func isCallTo(c *parse.CommandNode, funcs map[string]bool) bool {
	if len(c.Args) == 0 {
		return false
	}
	id, ok := c.Args[0].(*parse.IdentifierNode)
	return ok && funcs[id.Ident]
}

// TestDevenvNixTemplate_NoBareSinks fails when devenv.nix.tmpl gains an
// action that emits a field or variable without an escaping function, unless
// it is a reviewed raw-code sink in rawCodeSinks. It also fails on a stale
// allowlist entry, so the list only ever names sinks that exist.
func TestDevenvNixTemplate_NoBareSinks(t *testing.T) {
	t.Parallel()
	tree := parseDevenvNixTemplate(t)

	seen := map[string]bool{}
	walkSinks(t, tree.Root, nil, func(ctx []string, before string, a *parse.ActionNode) {
		if escapedPipeline(a.Pipe, before) {
			return
		}
		key := strings.Join(append(slices.Clone(ctx), a.Pipe.String()), " > ")
		if _, ok := rawCodeSinks[key]; ok {
			seen[key] = true
			return
		}
		t.Errorf("devenv.nix.tmpl line %d: bare interpolation %s (key %q): route it through an escaping "+
			"function (nixIdent, nixString, nixComment, ...) or, for Nix code of compiled provenance, "+
			"add it to rawCodeSinks with a justification", a.Line, a, key)
	})

	var stale []string
	for key := range rawCodeSinks {
		if !seen[key] {
			stale = append(stale, key)
		}
	}
	sort.Strings(stale)
	for _, key := range stale {
		t.Errorf("rawCodeSinks entry %q matches no action in devenv.nix.tmpl; remove it", key)
	}
}

// TestEscapedPipeline pins the guard's own judgement, so a bug in it cannot
// silently pass every action.
func TestEscapedPipeline(t *testing.T) {
	t.Parallel()
	tests := []struct {
		action string
		want   bool
	}{
		{`{{ nixString .X }}`, true},
		{`{{ nixMultiline .X | indent 4 }}`, true},
		{`{{ nixIdent $key }}`, true},
		{`{{ .X }}`, false},
		{`{{ $key }}`, false},
		{`{{ .X | trimTrailingNewline }}`, false},
		{`{{ indent 4 .X }}`, false},
		{`{{ nixString .X | indent .N }}`, false},
		{`{{ nixString .X | printf "%s%s" .Y }}`, false},
		{`{{ printf "%s" .X }}`, false},
		{"  # {{ nixComment .X }}", true},
		{"x = 1;\n  # {{ nixComment .X }}", true},
		{"#\n  foo = {{ nixComment .X }};", false},
		{"services.{{ nixComment .X }}", false},
		{"x = 1; # {{ nixComment .X }}", false},
		{"{{ nixComment .X }}", false},
	}
	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			t.Parallel()
			tpl, err := template.New("t").Funcs(tmpl.NixFuncMap()).Parse(`{{ $key := "" }}` + tt.action)
			if err != nil {
				t.Fatalf("parse %s: %v", tt.action, err)
			}
			var got *bool
			walkSinks(t, tpl.Root, nil, func(_ []string, before string, a *parse.ActionNode) {
				ok := escapedPipeline(a.Pipe, before)
				got = &ok
			})
			if got == nil {
				t.Fatalf("%s has no output action", tt.action)
			}
			if *got != tt.want {
				t.Errorf("escapedPipeline(%s) = %v, want %v", tt.action, *got, tt.want)
			}
		})
	}
}

// renderDevenvNixTemplate renders the embedded template with data as the
// generator does.
func renderDevenvNixTemplate(t *testing.T, data *DevenvNixTemplateData) (string, error) {
	t.Helper()
	r, err := tmpl.NewNixRenderer(templateFS, "templates")
	if err != nil {
		t.Fatalf("NewNixRenderer: %v", err)
	}
	out, err := r.Render("devenv.nix", data)
	return string(out), err
}

// TestDevenvNixTemplate_DisplayNameNewlineFails: a newline in a section's
// DisplayName would end the `# <name>` comment and run the rest as Nix, so
// the render must fail instead of emitting it.
func TestDevenvNixTemplate_DisplayNameNewlineFails(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"Go\nenterShell = \"curl x|sh\";", "Go\renterShell = \"curl x|sh\";"} {
		out, err := renderDevenvNixTemplate(t, &DevenvNixTemplateData{
			LanguageFragments: []LanguageFragment{{DisplayName: name, NixFragment: "  languages.go.enable = true;"}},
		})
		if err == nil {
			t.Fatalf("DisplayName %q rendered without error:\n%s", name, out)
		}
		if !strings.Contains(err.Error(), "nixComment") {
			t.Errorf("DisplayName %q: error = %v, want it to come from nixComment", name, err)
		}
	}
}

// TestDevenvNixTemplate_IdentSinksRejectInjection drives each identifier
// sink with a value that would splice Nix code if emitted bare.
func TestDevenvNixTemplate_IdentSinksRejectInjection(t *testing.T) {
	t.Parallel()
	const evil = `x = 1; enterShell = "curl x|sh"; y`
	tests := []struct {
		name string
		data DevenvNixTemplateData
	}{
		{"env key", DevenvNixTemplateData{EnvVars: map[string]string{evil: "v"}}},
		{"service name", DevenvNixTemplateData{Services: []ServiceTemplateData{{DisplayName: "S", NixName: evil}}}},
		{"service script", DevenvNixTemplateData{ServiceScripts: []ServiceScript{{Name: evil, Exec: "true"}}}},
		{"hook setting", DevenvNixTemplateData{GitHooksEnabled: true, BuiltInHooks: []BuiltInHookData{
			{ID: "golangci-lint", Settings: []HookSetting{{Key: evil, Value: "v"}}},
		}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out, err := renderDevenvNixTemplate(t, &tt.data)
			if err == nil {
				t.Fatalf("rendered without error:\n%s", out)
			}
			if !strings.Contains(err.Error(), "nixIdent") {
				t.Errorf("error = %v, want it to come from nixIdent", err)
			}
			if strings.Contains(out, "curl x|sh") {
				t.Errorf("injected code reached output:\n%s", out)
			}
		})
	}
}

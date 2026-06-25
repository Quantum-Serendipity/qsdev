package workspace

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	// Registers the ecosystem modules so per-package sub-detection has a
	// populated default registry.
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules"
)

// Resource URI template anchors for the per-package context resource
// (qsdev://project/{package}/context).
const (
	packageURIPrefix = "qsdev://project/"
	packageURISuffix = "/context"
	mimeJSON         = "application/json"
)

// ExtractPackageID parses the package identifier out of a
// qsdev://project/{package}/context URI. The identifier may be a relative
// directory ("packages/utils"), a qualified name ("npm:@myorg/utils"), or a bare
// name, and may be percent-encoded. It reports ok=false when uri does not match
// the template.
func ExtractPackageID(uri string) (string, bool) {
	if !strings.HasPrefix(uri, packageURIPrefix) || !strings.HasSuffix(uri, packageURISuffix) {
		return "", false
	}
	mid := uri[len(packageURIPrefix) : len(uri)-len(packageURISuffix)]
	if mid == "" {
		return "", false
	}
	if decoded, err := url.PathUnescape(mid); err == nil {
		mid = decoded
	}
	return mid, true
}

// RenderPackageContext resolves the package addressed by a per-package context
// URI and returns its context as a JSON ResourceResult. It reports ok=false when
// the URI does not match the template or no package resolves, so the caller can
// degrade to a not_configured response.
func (g *WorkspaceGraph) RenderPackageContext(uri string) (*spi.ResourceResult, bool) {
	id, ok := ExtractPackageID(uri)
	if !ok {
		return nil, false
	}
	pkg := g.Resolve(id)
	if pkg == nil {
		return nil, false
	}

	payload := g.PackageContextJSON(pkg)
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		// A marshaling failure of a plain map is effectively impossible; surface a
		// minimal valid document rather than failing the read.
		data = []byte(fmt.Sprintf("{\"name\":%q,\"ecosystem\":%q}", pkg.Name, pkg.Ecosystem))
	}
	return &spi.ResourceResult{
		Contents: []spi.ResourceContent{{URI: uri, MIMEType: mimeJSON, Text: string(data)}},
	}, true
}

// PackageContextJSON builds the JSON-serializable per-package context document:
// the package's identity (name, qualified name, ecosystem, relative directory,
// manifest), its declared dependencies, and a sub-detection of the ecosystems
// present in the package directory.
func (g *WorkspaceGraph) PackageContextJSON(pkg *Package) map[string]any {
	deps := pkg.Dependencies
	if deps == nil {
		deps = []string{}
	}
	return map[string]any{
		"name":           pkg.Name,
		"qualified_name": pkg.QualifiedName(),
		"ecosystem":      pkg.Ecosystem,
		"rel_dir":        pkg.RelDir,
		"manifest_path":  pkg.ManifestPath,
		"dependencies":   deps,
		"sub_detection":  g.subDetect(pkg.RelDir),
	}
}

// subDetect runs the ecosystem registry against the package directory and
// returns the ecosystems it detects there (sorted). It is filesystem-only and
// has no process side effects.
func (g *WorkspaceGraph) subDetect(relDir string) map[string]any {
	absDir := filepath.Join(g.Root(), filepath.FromSlash(relDir))
	summary := ecosystem.DefaultRegistry().DetectAll(absDir)

	ecos := make([]string, 0, len(summary.Project.Ecosystems))
	for name, present := range summary.Project.Ecosystems {
		if present {
			ecos = append(ecos, name)
		}
	}
	sort.Strings(ecos)
	return map[string]any{
		"ecosystems":       ecos,
		"has_go_mod":       summary.Project.HasGoMod,
		"has_package_json": summary.Project.HasPackageJSON,
	}
}

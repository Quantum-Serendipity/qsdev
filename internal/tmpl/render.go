package tmpl

import (
	"bytes"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// Renderer loads and renders templates from an fs.FS.
type Renderer struct {
	tmpl *template.Template
}

// NewNixRenderer creates a Renderer that parses all *.tmpl files under root
// within fsys, using the NixFuncMap for template functions.
func NewNixRenderer(fsys fs.FS, root string) (*Renderer, error) {
	return newRenderer(fsys, root, NixFuncMap())
}

// NewMarkdownRenderer creates a Renderer that parses all *.tmpl files under
// root within fsys, using the MarkdownFuncMap for template functions.
func NewMarkdownRenderer(fsys fs.FS, root string) (*Renderer, error) {
	return newRenderer(fsys, root, MarkdownFuncMap())
}

// newRenderer walks the FS under root, collecting all *.tmpl files and parsing
// them into a single template set. Template names are the path relative to root
// with the .tmpl suffix removed.
func newRenderer(fsys fs.FS, root string, funcMap template.FuncMap) (*Renderer, error) {
	t := template.New("").Funcs(funcMap)

	var found bool
	err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("reading template %s: %w", path, err)
		}

		// Template name = path relative to root, minus .tmpl suffix.
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("computing relative path for %s: %w", path, err)
		}
		name := strings.TrimSuffix(relPath, ".tmpl")

		_, err = t.New(name).Parse(string(data))
		if err != nil {
			return fmt.Errorf("parsing template %s: %w", name, err)
		}
		found = true
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking template directory %s: %w", root, err)
	}
	if !found {
		return nil, fmt.Errorf("no *.tmpl files found under %s", root)
	}

	return &Renderer{tmpl: t}, nil
}

// Render executes the named template with data and returns the rendered bytes.
// If the template is not found, the error includes the template name and lists
// all available templates.
func (r *Renderer) Render(templateName string, data any) ([]byte, error) {
	t := r.tmpl.Lookup(templateName)
	if t == nil {
		return nil, fmt.Errorf(
			"template %q not found; available templates: %s",
			templateName,
			strings.Join(r.AvailableTemplates(), ", "),
		)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template %q: %w", templateName, err)
	}
	return buf.Bytes(), nil
}

// RenderString is a convenience wrapper around Render that returns a string.
func (r *Renderer) RenderString(templateName string, data any) (string, error) {
	b, err := r.Render(templateName, data)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// AvailableTemplates returns a sorted list of all template names in the
// renderer's template set, excluding the unnamed root template.
func (r *Renderer) AvailableTemplates() []string {
	var names []string
	for _, t := range r.tmpl.Templates() {
		name := t.Name()
		if name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

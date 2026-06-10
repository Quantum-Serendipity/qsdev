package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/gobwas/glob"
)

type CompiledCondition interface {
	Evaluate(ctx *EvalContext) (bool, error)
}

type toolMatchCondition struct {
	toolName string
}

func (c *toolMatchCondition) Evaluate(ctx *EvalContext) (bool, error) {
	return ctx.ToolName == c.toolName, nil
}

type globCondition struct {
	glob glob.Glob
}

func (c *globCondition) Evaluate(ctx *EvalContext) (bool, error) {
	path := ctx.FilePath
	if path == "" {
		path = extractPathFromInput(ctx.ToolInput)
	}
	return c.glob.Match(path), nil
}

type regexCondition struct {
	re *regexp.Regexp
}

func (c *regexCondition) Evaluate(ctx *EvalContext) (bool, error) {
	target := ctx.Command
	if target == "" {
		target = string(ctx.ToolInput)
	}
	return c.re.MatchString(target), nil
}

type commandCondition struct {
	re *regexp.Regexp
}

func (c *commandCondition) Evaluate(ctx *EvalContext) (bool, error) {
	return c.re.MatchString(ctx.Command), nil
}

type fileExistenceCondition struct {
	path string
}

func (c *fileExistenceCondition) Evaluate(ctx *EvalContext) (bool, error) {
	_, err := os.Stat(filepath.Join(ctx.CWD, c.path))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("checking file existence for %q: %w", c.path, err)
}

type fileTypeCondition struct {
	path     string
	fileType string
}

func (c *fileTypeCondition) Evaluate(ctx *EvalContext) (bool, error) {
	info, err := os.Lstat(filepath.Join(ctx.CWD, c.path))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("checking file type for %q: %w", c.path, err)
	}

	switch c.fileType {
	case "file":
		return info.Mode().IsRegular(), nil
	case "directory":
		return info.IsDir(), nil
	case "symlink":
		return info.Mode()&os.ModeSymlink != 0, nil
	default:
		return false, fmt.Errorf("unknown file type %q", c.fileType)
	}
}

type deniedPathCheckCondition struct {
	glob glob.Glob
}

func (c *deniedPathCheckCondition) Evaluate(ctx *EvalContext) (bool, error) {
	path := ctx.FilePath
	if path == "" {
		path = extractPathFromInput(ctx.ToolInput)
	}
	return c.glob.Match(path), nil
}

type semanticCondition struct {
	prompt string
}

func (c *semanticCondition) Evaluate(_ *EvalContext) (bool, error) {
	return false, nil
}

type allCondition struct {
	children []CompiledCondition
}

func (c *allCondition) Evaluate(ctx *EvalContext) (bool, error) {
	for _, child := range c.children {
		result, err := child.Evaluate(ctx)
		if err != nil {
			return false, err
		}
		if !result {
			return false, nil
		}
	}
	return true, nil
}

type anyCondition struct {
	children []CompiledCondition
}

func (c *anyCondition) Evaluate(ctx *EvalContext) (bool, error) {
	for _, child := range c.children {
		result, err := child.Evaluate(ctx)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil
		}
	}
	return false, nil
}

type notCondition struct {
	child CompiledCondition
}

func (c *notCondition) Evaluate(ctx *EvalContext) (bool, error) {
	result, err := c.child.Evaluate(ctx)
	if err != nil {
		return false, err
	}
	return !result, nil
}

func CompileCondition(cond Condition) (CompiledCondition, error) {
	switch cond.Type {
	case ToolMatch:
		return &toolMatchCondition{toolName: cond.ToolName}, nil

	case PathGlob:
		g, err := glob.Compile(cond.Pattern)
		if err != nil {
			return nil, fmt.Errorf("compiling %s condition: %w", cond.Type, err)
		}
		return &globCondition{glob: g}, nil

	case RegexMatch:
		re, err := regexp.Compile(cond.Pattern)
		if err != nil {
			return nil, fmt.Errorf("compiling %s condition: invalid regex %q: %w", cond.Type, cond.Pattern, err)
		}
		return &regexCondition{re: re}, nil

	case CommandMatch:
		pattern := `(?:^|\s)` + regexp.QuoteMeta(cond.Pattern) + `(?:\s|$)`
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("compiling %s condition: %w", cond.Type, err)
		}
		return &commandCondition{re: re}, nil

	case FileExistence:
		return &fileExistenceCondition{path: cond.Path}, nil

	case FileType:
		return &fileTypeCondition{path: cond.Path, fileType: cond.FileType}, nil

	case DeniedPathCheck:
		g, err := glob.Compile(cond.Pattern)
		if err != nil {
			return nil, fmt.Errorf("compiling %s condition: %w", cond.Type, err)
		}
		return &deniedPathCheckCondition{glob: g}, nil

	case Semantic:
		return &semanticCondition{prompt: cond.Prompt}, nil

	case All:
		children, err := compileChildren(cond.Conditions)
		if err != nil {
			return nil, fmt.Errorf("compiling %s condition: %w", cond.Type, err)
		}
		return &allCondition{children: children}, nil

	case Any:
		children, err := compileChildren(cond.Conditions)
		if err != nil {
			return nil, fmt.Errorf("compiling %s condition: %w", cond.Type, err)
		}
		return &anyCondition{children: children}, nil

	case Not:
		if cond.Condition == nil {
			return nil, fmt.Errorf("compiling %s condition: missing child condition", cond.Type)
		}
		child, err := CompileCondition(*cond.Condition)
		if err != nil {
			return nil, fmt.Errorf("compiling %s condition: %w", cond.Type, err)
		}
		return &notCondition{child: child}, nil

	default:
		return nil, fmt.Errorf("compiling condition: unknown type %q", cond.Type)
	}
}

func compileChildren(conditions []Condition) ([]CompiledCondition, error) {
	children := make([]CompiledCondition, 0, len(conditions))
	for _, child := range conditions {
		compiled, err := CompileCondition(child)
		if err != nil {
			return nil, err
		}
		children = append(children, compiled)
	}
	return children, nil
}

func extractPathFromInput(input json.RawMessage) string {
	if len(input) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(input, &m); err != nil {
		return ""
	}
	for _, key := range []string{"file_path", "path", "file"} {
		if v, ok := m[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

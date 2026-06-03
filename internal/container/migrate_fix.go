package container

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ApplyFixes reads a compose file, applies all auto-fixable issues targeting
// it, and returns the modified YAML bytes. It uses the yaml.v3 Node API to
// preserve comments and formatting.
func ApplyFixes(filePath string, issues []MigrationIssue) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filePath, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", filePath, err)
	}

	// Filter to auto-fixable issues for this file.
	var applicable []MigrationIssue
	for _, issue := range issues {
		if issue.File == filePath && issue.AutoFixable && issue.Fix != nil {
			applicable = append(applicable, issue)
		}
	}

	if len(applicable) == 0 {
		return data, nil
	}

	for _, issue := range applicable {
		if err := applyFix(&doc, issue); err != nil {
			return nil, fmt.Errorf("applying fix for %s/%s (%s): %w",
				issue.Service, issue.Category, filePath, err)
		}
	}

	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, fmt.Errorf("encoding %s: %w", filePath, err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("closing encoder for %s: %w", filePath, err)
	}

	return []byte(buf.String()), nil
}

// applyFix dispatches to the appropriate fix strategy based on issue category.
func applyFix(doc *yaml.Node, issue MigrationIssue) error {
	switch issue.Category {
	case CategoryVolumePerms:
		return applyUsernsMode(doc, issue.Service)
	case CategoryImageName:
		return applyImageQualification(doc, issue.Service)
	case CategoryPrivPorts:
		return applyPortRemap(doc, issue.Service)
	case CategorySocketMount:
		return applySocketReplacement(doc, issue.Service)
	case CategorySELinux:
		return applySELinuxSuffix(doc, issue.Service)
	default:
		return nil
	}
}

// applyUsernsMode adds userns_mode: keep-id to a service.
func applyUsernsMode(doc *yaml.Node, serviceName string) error {
	svcNode := findServiceNode(doc, serviceName)
	if svcNode == nil {
		return fmt.Errorf("service %q not found in document", serviceName)
	}

	// Check if userns_mode already exists.
	if _, valNode := findMappingKey(svcNode, "userns_mode"); valNode != nil {
		valNode.Value = "keep-id"
		return nil
	}

	addMappingKey(svcNode, "userns_mode", "keep-id")
	return nil
}

// applyImageQualification qualifies the image name with docker.io.
func applyImageQualification(doc *yaml.Node, serviceName string) error {
	svcNode := findServiceNode(doc, serviceName)
	if svcNode == nil {
		return fmt.Errorf("service %q not found in document", serviceName)
	}

	_, valNode := findMappingKey(svcNode, "image")
	if valNode == nil {
		return fmt.Errorf("service %q has no image key", serviceName)
	}

	valNode.Value = qualifyImageName(valNode.Value)
	return nil
}

// applyPortRemap remaps host ports below 1024 to port+8000.
func applyPortRemap(doc *yaml.Node, serviceName string) error {
	svcNode := findServiceNode(doc, serviceName)
	if svcNode == nil {
		return fmt.Errorf("service %q not found in document", serviceName)
	}

	_, portsNode := findMappingKey(svcNode, "ports")
	if portsNode == nil || portsNode.Kind != yaml.SequenceNode {
		return nil
	}

	for _, item := range portsNode.Content {
		if item.Kind == yaml.ScalarNode {
			item.Value = remapPort(item.Value)
		}
	}

	return nil
}

// remapPort remaps a port string's host port if it is below 1024.
func remapPort(portStr string) string {
	// Strip protocol suffix.
	proto := ""
	if idx := strings.Index(portStr, "/"); idx >= 0 {
		proto = portStr[idx:]
		portStr = portStr[:idx]
	}

	parts := strings.Split(portStr, ":")
	switch len(parts) {
	case 2:
		// host:container
		hostPort, err := strconv.Atoi(parts[0])
		if err == nil && hostPort > 0 && hostPort < 1024 {
			parts[0] = strconv.Itoa(hostPort + 8000)
		}
	case 3:
		// ip:host:container
		hostPort, err := strconv.Atoi(parts[1])
		if err == nil && hostPort > 0 && hostPort < 1024 {
			parts[1] = strconv.Itoa(hostPort + 8000)
		}
	}

	return strings.Join(parts, ":") + proto
}

// applySocketReplacement replaces Docker socket paths with Podman socket paths.
func applySocketReplacement(doc *yaml.Node, serviceName string) error {
	svcNode := findServiceNode(doc, serviceName)
	if svcNode == nil {
		return fmt.Errorf("service %q not found in document", serviceName)
	}

	_, volsNode := findMappingKey(svcNode, "volumes")
	if volsNode == nil || volsNode.Kind != yaml.SequenceNode {
		return nil
	}

	for _, item := range volsNode.Content {
		if item.Kind == yaml.ScalarNode && strings.Contains(item.Value, "/var/run/docker.sock") {
			item.Value = strings.Replace(item.Value,
				"/var/run/docker.sock",
				"${XDG_RUNTIME_DIR}/podman/podman.sock",
				1)
		}
	}

	return nil
}

// applySELinuxSuffix appends :Z to bind mount volumes that lack SELinux labels.
func applySELinuxSuffix(doc *yaml.Node, serviceName string) error {
	svcNode := findServiceNode(doc, serviceName)
	if svcNode == nil {
		return fmt.Errorf("service %q not found in document", serviceName)
	}

	_, volsNode := findMappingKey(svcNode, "volumes")
	if volsNode == nil || volsNode.Kind != yaml.SequenceNode {
		return nil
	}

	for _, item := range volsNode.Content {
		if item.Kind != yaml.ScalarNode {
			continue
		}
		vol := item.Value
		parts := strings.Split(vol, ":")
		if len(parts) < 2 {
			continue
		}
		host := parts[0]
		if !strings.HasPrefix(host, ".") && !strings.HasPrefix(host, "/") && !strings.HasPrefix(host, "~") {
			continue
		}
		// Check if already has z/Z option.
		if len(parts) >= 3 {
			opts := parts[len(parts)-1]
			if strings.Contains(opts, "z") || strings.Contains(opts, "Z") {
				continue
			}
		}
		item.Value = vol + ":Z"
	}

	return nil
}

// findServiceNode locates a service mapping node by name within the document.
func findServiceNode(doc *yaml.Node, serviceName string) *yaml.Node {
	if doc == nil {
		return nil
	}

	// The document node wraps the actual content.
	root := doc
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}

	_, servicesVal := findMappingKey(root, "services")
	if servicesVal == nil || servicesVal.Kind != yaml.MappingNode {
		return nil
	}

	_, svcVal := findMappingKey(servicesVal, serviceName)
	if svcVal == nil || svcVal.Kind != yaml.MappingNode {
		return nil
	}

	return svcVal
}

// findMappingKey searches a mapping node for a key and returns both the
// key node and value node.
func findMappingKey(node *yaml.Node, key string) (*yaml.Node, *yaml.Node) {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil, nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i], node.Content[i+1]
		}
	}
	return nil, nil
}

// addMappingKey appends a new scalar key-value pair to a mapping node.
func addMappingKey(node *yaml.Node, key, value string) {
	keyNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: key,
	}
	valNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: value,
	}
	node.Content = append(node.Content, keyNode, valNode)
}

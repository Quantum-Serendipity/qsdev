package container

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const portRemapOffset = 8000

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

// applyPortRemap remaps host ports below 1024 to port+8000, except where that
// port is already published in the file (see portRemapTarget).
func applyPortRemap(doc *yaml.Node, serviceName string) error {
	svcNode := findServiceNode(doc, serviceName)
	if svcNode == nil {
		return fmt.Errorf("service %q not found in document", serviceName)
	}

	_, portsNode := findMappingKey(svcNode, "ports")
	if portsNode == nil || portsNode.Kind != yaml.SequenceNode {
		return nil
	}

	published := documentPublishedPorts(doc)
	for _, item := range portsNode.Content {
		switch item.Kind {
		case yaml.ScalarNode:
			item.Value = remapPort(item.Value, published)
		case yaml.MappingNode:
			remapMappingPort(item, published)
		}
	}

	return nil
}

// documentPublishedPorts returns every host port the document's services
// publish, as it stands before any remapping.
func documentPublishedPorts(doc *yaml.Node) map[publishedPort]bool {
	result := make(map[publishedPort]bool)
	for _, svcNode := range serviceNodes(doc) {
		_, portsNode := findMappingKey(svcNode, "ports")
		if portsNode == nil || portsNode.Kind != yaml.SequenceNode {
			continue
		}
		for _, item := range portsNode.Content {
			if p, ok := nodePort(item); ok {
				result[p] = true
			}
		}
	}
	return result
}

// nodePort parses a port entry node (short or long syntax) into the host port
// it publishes.
func nodePort(node *yaml.Node) (publishedPort, bool) {
	switch node.Kind {
	case yaml.ScalarNode:
		return parsePortSpec(node.Value)
	case yaml.MappingNode:
		_, pub := findMappingKey(node, "published")
		if pub == nil || pub.Kind != yaml.ScalarNode {
			return publishedPort{}, false
		}
		protocol := ""
		if _, proto := findMappingKey(node, "protocol"); proto != nil {
			protocol = proto.Value
		}
		return longPortSpec(pub.Value, protocol)
	}
	return publishedPort{}, false
}

// remapMappingPort remaps the "published" key in a map-style port entry.
func remapMappingPort(node *yaml.Node, published map[publishedPort]bool) {
	p, ok := nodePort(node)
	if !ok || p.port >= 1024 {
		return
	}
	target, ok := portRemapTarget(p, published)
	if !ok {
		return
	}
	_, pubNode := findMappingKey(node, "published")
	pubNode.Value = strconv.Itoa(target)
}

// remapPort remaps a port string's host port if it is below 1024 and its
// remap target is free.
func remapPort(portStr string, published map[publishedPort]bool) string {
	p, ok := parsePortSpec(portStr)
	if !ok || p.port >= 1024 {
		return portStr
	}
	target, ok := portRemapTarget(p, published)
	if !ok {
		return portStr
	}

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
		parts[0] = strconv.Itoa(target)
	case 3:
		// ip:host:container
		parts[1] = strconv.Itoa(target)
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

// applySELinuxSuffix adds an SELinux relabel option to the service's
// project-relative bind mounts that lack one: :Z for a path only this service
// mounts, :z for one several services share. Other bind mounts (system paths,
// the home directory, sockets) are left alone; see selinuxRelabelOption.
func applySELinuxSuffix(doc *yaml.Node, serviceName string) error {
	svcNode := findServiceNode(doc, serviceName)
	if svcNode == nil {
		return fmt.Errorf("service %q not found in document", serviceName)
	}

	_, volsNode := findMappingKey(svcNode, "volumes")
	if volsNode == nil || volsNode.Kind != yaml.SequenceNode {
		return nil
	}

	bindHosts := countBindHosts(documentVolumes(doc))
	for _, item := range volsNode.Content {
		if item.Kind != yaml.ScalarNode || !strings.Contains(item.Value, ":") || hasSELinuxOption(item.Value) {
			continue
		}
		option, ok := selinuxRelabelOption(bindHostOf(item.Value), bindHosts)
		if !ok {
			continue
		}
		item.Value = appendSELinuxOption(item.Value, option)
	}

	return nil
}

// documentVolumes returns each service's short-syntax volume entries.
func documentVolumes(doc *yaml.Node) map[string][]string {
	result := make(map[string][]string)
	for name, svcNode := range serviceNodes(doc) {
		_, volsNode := findMappingKey(svcNode, "volumes")
		if volsNode == nil || volsNode.Kind != yaml.SequenceNode {
			continue
		}
		for _, item := range volsNode.Content {
			if item.Kind == yaml.ScalarNode {
				result[name] = append(result[name], item.Value)
			}
		}
	}
	return result
}

// serviceNodes returns the document's service mapping nodes by name.
func serviceNodes(doc *yaml.Node) map[string]*yaml.Node {
	result := make(map[string]*yaml.Node)
	root := doc
	if root != nil && root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	_, servicesVal := findMappingKey(root, "services")
	if servicesVal == nil || servicesVal.Kind != yaml.MappingNode {
		return result
	}
	for i := 0; i+1 < len(servicesVal.Content); i += 2 {
		if svc := servicesVal.Content[i+1]; svc.Kind == yaml.MappingNode {
			result[servicesVal.Content[i].Value] = svc
		}
	}
	return result
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

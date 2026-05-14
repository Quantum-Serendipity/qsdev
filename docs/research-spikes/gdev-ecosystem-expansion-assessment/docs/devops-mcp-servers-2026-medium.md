<!-- Source: https://medium.com/k8slens/18-best-devops-mcp-servers-for-2026-the-definitive-guide-bfde04654a35 -->
<!-- Retrieved: 2026-05-14 -->

# 18 Best DevOps MCP Servers for 2026 (Medium/k8slens)

## Version Control and CI/CD (3 servers)

**1. GitHub MCP Server**
- Function: Code browsing, issue management, PR operations, CI/CD workflow triggering through GitHub Actions
- Maturity: "the most widely deployed DevOps MCP in the ecosystem"
- Type: Official (GitHub)
- Access: Supports read-only flag to prevent mutations

**2. GitLab MCP Server**
- Function: Repository interactions, issue/PR creation, GitLab CI/CD pipeline management
- Maturity: Stable
- Type: Official (GitLab)

**3. Azure DevOps MCP Server**
- Function: Full platform coverage including repositories, work items, builds, and releases
- Maturity: Actively developed
- Type: Official (Microsoft)
- Features: Multi-project management and directory-based authentication

## Docker and Kubernetes (4 servers)

**4. Docker Hub MCP Server**
- Function: Natural language discovery of container images and repository management
- Maturity: Recently launched
- Type: Official (Docker)
- Context: Works alongside Docker MCP Toolkit featuring 200+ MCP servers

**5. Kubernetes MCP Server**
- Function: Cluster visibility, pod troubleshooting, root cause analysis via log correlation
- Maturity: Established
- Type: Community (containers/kubernetes-mcp-server)
- Limitation: Requires kubeconfig configuration for cloud provider clusters

**6. Lens MCP Server**
- Function: Kubernetes management with native AWS EKS and Azure AKS integration
- Maturity: Production-ready
- Type: Official (Lens)

**7. ArgoCD MCP Server**
- Function: Application listing/creation/updating, resource management, cluster registration viewing
- Maturity: Established
- Type: Community (argoproj-labs)

## Infrastructure as Code (4 servers)

**8. Terraform MCP Server**
- Function: Workspace management, run triggering, state inspection, cost estimation, and registry browsing
- Maturity: Official release
- Type: Official (HashiCorp)
- Requirement: Terraform Cloud integration

**9. Spacelift Intent**
- Function: Infrastructure resource creation/updating/deletion and lifecycle tracking for AI-generated resources
- Maturity: Production-ready
- Type: Official (Spacelift)

**10. AWS MCP Server**
- Function: Real-time AWS knowledge, troubleshooting, infrastructure provisioning, cost management
- Maturity: Currently in Preview Mode
- Type: Official (AWS)
- Integration: Merges AWS Knowledge and AWS API MCP capabilities

**11. Azure MCP Server**
- Function: Resource management across Azure services
- Maturity: Production-ready
- Type: Official (Microsoft)

## Observability (3 servers)

**12. Grafana MCP Server**
- Function: Dashboard data querying, data source inspection, incident detail retrieval
- Maturity: Optimized for token efficiency
- Type: Official (Grafana)
- Design: "optimized how the server structures responses to minimize the context windows usage"

**13. Prometheus MCP Server**
- Function: Natural language to PromQL translation and query execution
- Maturity: Functional
- Type: Community

**14. Datadog MCP Server**
- Function: Unified access to metrics, logs, traces, and incident management
- Maturity: Production-ready
- Type: Official (Datadog)
- Use case: Reduces context-switching during incident response

## Security (4 servers)

**15. Trivy MCP Server**
- Function: Natural language vulnerability scanning for container images and infrastructure code
- Maturity: Established
- Type: Community (aquasecurity)

**16. Prowler MCP Server**
- Function: Cloud misconfiguration detection, risk assessment, automated remediation PR generation
- Maturity: Widely adopted
- Type: Community (prowler-cloud)

**17. Wiz MCP Server**
- Function: Cloud security posture management integration
- Maturity: Production-ready
- Type: Official (Wiz)

**18. Snyk MCP Server**
- Function: Code, configuration, and dependency security scanning
- Maturity: Production-ready
- Type: Official (Snyk)

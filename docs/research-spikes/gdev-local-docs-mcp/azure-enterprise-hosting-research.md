# Enterprise Azure Self-Hosting for Documentation Corpora with Entra ID SSO

## Executive Summary

For Highspring Digital's ~20-developer consulting firm using Azure and Entra ID, the optimal architecture for hosting large documentation corpora (74 GB Stack Overflow ZIM, 3.5 GB DevDocs, ~100 GB total) is a **hybrid approach**: Azure Blob Storage with BlobFuse2 local caching for ZIM/DevDocs files accessed by local MCP servers, with an optional kiwix-serve Azure Container Apps instance for teams that prefer remote access. Authentication uses DefaultAzureCredential, which transparently picks up `az login` tokens on developer machines and managed identities in Azure. Total monthly cost is approximately **$5-8/month** for storage-only (Blob Hot tier + minimal egress), or **$40-80/month** if running kiwix-serve as an always-on container service.

The key architectural insight is that BlobFuse2 — already packaged in nixpkgs as `blobfuse` v2.5.3 — can mount Azure Blob Storage as a local filesystem, making ZIM files transparently accessible to openzim-mcp and python-libzim without any code changes. Its file cache mode downloads entire files to a local SSD cache on first access, providing local-disk performance for subsequent reads. This preserves the "curated content" security benefit while solving the disk space problem for the full Stack Overflow corpus.

---

## 1. Azure Storage Options Comparison

### 1.1 Three Storage Services

| Aspect | Azure Blob Storage | Azure Files | Azure NetApp Files |
|--------|-------------------|-------------|-------------------|
| **Optimized for** | Sequential access, large-scale | Random access workloads | Ultra-low latency, enterprise NAS |
| **Protocols** | REST, NFSv3 (Data Lake) | SMB 2.1/3.x, NFSv4.1, REST | NFSv3/4.1, SMB, dual protocol |
| **Max IOPS** | 20,000 | 102,400 (SSD) / 50,000 (HDD) | 460,000 |
| **Latency** | Varies (REST-based) | 2-3ms (SSD small IO) | <1ms |
| **100 GB cost (LRS)** | ~$1.80/month (Hot) | ~$2.76/month (Hot) / $18.10 (Premium SSD) | ~$15-30/month (minimum 1 TiB pool) |
| **Authentication** | Entra ID via RBAC | SMB: Entra Kerberos; NFS: network only | AD DS / LDAP |
| **Linux mount** | BlobFuse2 (FUSE) | SMB mount / NFS mount | NFS mount |

### 1.2 Recommendation: Azure Blob Storage + BlobFuse2

**Azure Blob Storage is the clear winner for this use case**, for several reasons:

1. **Cost**: At $0.018/GB/month (Hot tier, LRS), 100 GB costs just $1.80/month. Azure Files Hot is $2.76/month, and Premium SSD is $18.10/month. NetApp Files requires a minimum 1 TiB capacity pool, making it wildly overprovisioned.

2. **BlobFuse2 solves the random access problem**: While Blob Storage is optimized for sequential access at the REST level, BlobFuse2's **file cache mode** downloads entire files to a local SSD cache directory. Once cached, openzim-mcp reads from local disk — identical performance to having the ZIM file stored locally. The 74 GB Stack Overflow ZIM would be cached on first access and served from SSD thereafter.

3. **Entra ID RBAC works well**: Blob Storage supports `Storage Blob Data Reader` role assignment via Entra ID. No shared keys needed. DefaultAzureCredential picks up `az login` tokens automatically.

4. **BlobFuse2 is in nixpkgs**: Available as `blobfuse` v2.5.3 (MIT license), supporting x86_64-linux and aarch64-linux. No custom packaging needed for NixOS.

**Azure Files is the runner-up** for teams that prefer native NFS/SMB mounts without FUSE overhead, but the cost is higher and NFS shares do not support Entra ID authentication (only network-level auth).

**Azure NetApp Files is overkill** — sub-millisecond latency and 460K IOPS are unnecessary for documentation serving, and the minimum 1 TiB pool makes it cost-prohibitive for 100 GB.

### 1.3 BlobFuse2 Deep Dive

BlobFuse2 is a FUSE virtual filesystem driver that translates Linux file operations into Azure Blob REST API calls.

**File Cache Mode (recommended for ZIM files)**:
- Downloads the **entire file** from Blob Storage into a local cache directory before making it available
- Subsequent reads operate on the local cache until eviction or invalidation
- Cache location, size, and retention policies are configurable
- Can **preload** entire containers at mount time — perfect for downloading ZIM files on first `gdev setup`
- After caching, performance equals local SSD (zero network latency for reads)

**Block Cache Mode (alternative for very large files)**:
- Streams data in chunks without downloading the full file
- Better for files that do not fit in local cache
- Has consistency limitations (concurrent read/write issues)
- Not recommended for ZIM files that need random access via Xapian search

**Key limitations**:
- Not fully POSIX compliant (rename operations are not atomic)
- Requires FUSE kernel module (standard on NixOS)
- File cache requires local disk space equal to the files being accessed (74 GB for full SO ZIM)

**Authentication methods supported**:
- Azure CLI credential (via `az login`)
- Managed identity
- Service principal (client secret or certificate)
- SAS token
- Storage account key (not recommended)

**NixOS compatibility**: BlobFuse2 is already packaged in nixpkgs (`blobfuse`, v2.5.3, MIT). It uses libfuse3 which is standard on NixOS. The `fuse` kernel module may need to be enabled in NixOS configuration:

```nix
boot.supportedFilesystems = [ "fuse" ];
environment.systemPackages = [ pkgs.blobfuse ];
```

### 1.4 Latency Analysis

| Access Pattern | Latency | Notes |
|----------------|---------|-------|
| Local SSD | <0.1ms | Baseline |
| BlobFuse2 file cache (after first access) | <0.1ms | Reading from local SSD cache |
| BlobFuse2 first access (downloading) | Seconds to minutes | Depends on file size and bandwidth. 74 GB @ 1 Gbps = ~10 minutes |
| Azure Files NFS mount | 2-3ms | Per small I/O operation |
| Azure Files SMB mount (with caching) | 2-3ms | `cache=strict` mount option recommended |
| Azure NetApp Files | <1ms | Overkill for this use case |
| BlobFuse2 block cache | 10-100ms | Per-block network round trip |

**For ZIM file Xapian search**: Xapian performs random I/O across the embedded search index. With BlobFuse2 file cache mode, the entire ZIM file is local after first access, so Xapian search operates at local SSD speed. This is critical — block cache mode would be unacceptably slow for Xapian's random access pattern.

---

## 2. Entra ID Authentication

### 2.1 DefaultAzureCredential Chain

The Azure Identity library's `DefaultAzureCredential` provides a unified authentication experience across development and production. On Linux (the NixOS developer machine case), it tries credentials in this order:

1. **EnvironmentCredential** — Service principal via `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID` environment variables
2. **WorkloadIdentityCredential** — Kubernetes workload identity
3. **ManagedIdentityCredential** — Azure managed identity (for Azure-hosted services)
4. **VisualStudioCodeCredential** — VS Code Azure extension
5. **AzureCliCredential** — `az login` token (the primary path for developers)
6. **AzurePowerShellCredential** — PowerShell login
7. **AzureDeveloperCliCredential** — `azd auth login`

For developer machines, the flow is: developer runs `az login` once (or `gdev auth` which wraps it), and all subsequent Azure SDK calls automatically use the cached token. Token refresh is automatic.

### 2.2 Authentication Flows for Different Scenarios

**Developer machine (local MCP server + BlobFuse2)**:
```
Developer runs `az login` → token cached in ~/.azure/
BlobFuse2 mount uses AzureCliCredential → reads from Blob Storage
openzim-mcp reads from BlobFuse2 mount → local file path
```

**Azure-hosted kiwix-serve (Container Apps)**:
```
Container Apps managed identity → ManagedIdentityCredential
kiwix-serve reads ZIM from Azure Files mount or Blob volume
Entra ID auth protects the kiwix-serve HTTP endpoint
```

**Automated pipeline (ZIM file upload/update)**:
```
Service principal with AZURE_CLIENT_ID/SECRET/TENANT_ID
EnvironmentCredential → uploads ZIM files to Blob Storage
```

### 2.3 BlobFuse2 Authentication Integration

BlobFuse2 supports multiple auth methods. For gdev, the recommended approach:

```yaml
# ~/.config/gdev/blobfuse-config.yaml
allow-other: true
logging:
  type: syslog
  level: log_warning

components:
  - libfuse
  - file_cache
  - attr_cache
  - azstorage

libfuse:
  attribute-expiration-sec: 120
  entry-expiration-sec: 120

file_cache:
  path: /tmp/gdev-blobfuse-cache
  timeout-sec: 86400  # 24 hours
  max-size-mb: 102400  # 100 GB cache limit

attr_cache:
  timeout-sec: 7200

azstorage:
  type: block
  account-name: highspringdocs
  container: documentation
  mode: msi  # or 'azcli' for developer machines
  # For developer machines using az login:
  # mode: azcli
```

Mount command (would be wrapped by `gdev setup`):
```bash
blobfuse2 mount ~/.local/share/gdev/azure-docs \
  --config-file ~/.config/gdev/blobfuse-config.yaml
```

### 2.4 Token Expiry and Offline Fallback

**Token expiry**: Azure CLI tokens typically last 1 hour but are auto-refreshed by the Azure CLI credential provider as long as the refresh token is valid (up to 90 days). If the refresh token expires, the user needs to run `az login` again. BlobFuse2 handles token refresh transparently.

**Offline fallback**: When Azure is unreachable:
- Files already in BlobFuse2's local cache remain accessible (configurable cache TTL, default 24h)
- openzim-mcp continues to read from cached ZIM files
- New files or cache-evicted files are unavailable
- gdev should detect mount failure and fall back to any locally-stored ZIM files

**Recommended approach**: gdev should maintain a "core docs" set (curated SE sites totaling ~5 GB) locally, with the full 74 GB Stack Overflow as an Azure-only supplement. This way, developers always have basic documentation even offline.

---

## 3. Terraform Module Design

### 3.1 Proposed Module Structure

```
modules/gdev-docs-storage/
├── main.tf           # Storage account, containers, RBAC
├── variables.tf      # Input configuration
├── outputs.tf        # Endpoints, mount commands
├── private-endpoint.tf  # Optional private endpoint
└── README.md
```

### 3.2 Core Resources

```hcl
# variables.tf
variable "resource_group_name" {
  type        = string
  description = "Resource group for documentation storage"
}

variable "location" {
  type        = string
  default     = "eastus2"
  description = "Azure region"
}

variable "storage_account_name" {
  type        = string
  default     = "highspringdocs"
  description = "Storage account name (globally unique)"
}

variable "storage_sku" {
  type        = string
  default     = "Standard_LRS"
  description = "Storage redundancy (Standard_LRS, Standard_GRS, Standard_ZRS)"
}

variable "entra_reader_group_id" {
  type        = string
  description = "Entra ID group object ID for developers (Storage Blob Data Reader)"
}

variable "entra_contributor_group_id" {
  type        = string
  default     = ""
  description = "Entra ID group object ID for admins (Storage Blob Data Contributor)"
}

variable "enable_private_endpoint" {
  type        = bool
  default     = false
  description = "Create a private endpoint for the storage account"
}

variable "private_endpoint_subnet_id" {
  type        = string
  default     = ""
  description = "Subnet ID for private endpoint (required if enable_private_endpoint is true)"
}

variable "allowed_ip_ranges" {
  type        = list(string)
  default     = []
  description = "IP ranges allowed to access storage (CIDR notation). Empty = all allowed."
}

variable "tags" {
  type = map(string)
  default = {
    managed-by = "gdev"
    purpose    = "developer-documentation"
  }
}
```

```hcl
# main.tf
terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = ">= 4.0"
    }
  }
}

resource "azurerm_storage_account" "docs" {
  name                      = var.storage_account_name
  resource_group_name       = var.resource_group_name
  location                  = var.location
  account_tier              = "Standard"
  account_replication_type  = replace(var.storage_sku, "Standard_", "")
  account_kind              = "StorageV2"
  access_tier               = "Hot"
  min_tls_version           = "TLS1_2"
  shared_access_key_enabled = false  # Force Entra ID auth only

  blob_properties {
    versioning_enabled = true
  }

  dynamic "network_rules" {
    for_each = length(var.allowed_ip_ranges) > 0 || var.enable_private_endpoint ? [1] : []
    content {
      default_action = var.enable_private_endpoint ? "Deny" : "Allow"
      ip_rules       = var.allowed_ip_ranges
      bypass         = ["AzureServices"]
    }
  }

  tags = var.tags
}

resource "azurerm_storage_container" "zim" {
  name                  = "zim-files"
  storage_account_id    = azurerm_storage_account.docs.id
  container_access_type = "private"
}

resource "azurerm_storage_container" "devdocs" {
  name                  = "devdocs"
  storage_account_id    = azurerm_storage_account.docs.id
  container_access_type = "private"
}

# RBAC: Storage Blob Data Reader for all developers
resource "azurerm_role_assignment" "reader" {
  scope                = azurerm_storage_account.docs.id
  role_definition_name = "Storage Blob Data Reader"
  principal_id         = var.entra_reader_group_id
}

# RBAC: Storage Blob Data Contributor for admins (upload/update)
resource "azurerm_role_assignment" "contributor" {
  count                = var.entra_contributor_group_id != "" ? 1 : 0
  scope                = azurerm_storage_account.docs.id
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = var.entra_contributor_group_id
}
```

```hcl
# private-endpoint.tf
resource "azurerm_private_endpoint" "docs" {
  count               = var.enable_private_endpoint ? 1 : 0
  name                = "${var.storage_account_name}-pe"
  location            = var.location
  resource_group_name = var.resource_group_name
  subnet_id           = var.private_endpoint_subnet_id

  private_service_connection {
    name                           = "${var.storage_account_name}-psc"
    private_connection_resource_id = azurerm_storage_account.docs.id
    is_manual_connection           = false
    subresource_names              = ["blob"]
  }

  tags = var.tags
}
```

```hcl
# outputs.tf
output "storage_account_name" {
  value = azurerm_storage_account.docs.name
}

output "blob_endpoint" {
  value = azurerm_storage_account.docs.primary_blob_endpoint
}

output "zim_container_name" {
  value = azurerm_storage_container.zim.name
}

output "devdocs_container_name" {
  value = azurerm_storage_container.devdocs.name
}

output "blobfuse_mount_command" {
  value = <<-EOT
    # Mount ZIM files via BlobFuse2 (requires az login)
    blobfuse2 mount ~/.local/share/gdev/azure-docs \
      --tmp-path /tmp/gdev-blobfuse-cache \
      --container-name zim-files \
      --account-name ${azurerm_storage_account.docs.name}
  EOT
}

output "mcp_config_snippet" {
  value = jsonencode({
    mcpServers = {
      "azure-docs" = {
        command = "openzim-mcp"
        env = {
          OPENZIM_MCP_ZIM_DIR  = "~/.local/share/gdev/azure-docs"
          OPENZIM_MCP_TOOL_MODE = "simple"
        }
      }
    }
  })
}
```

### 3.3 gdev Profile Integration

gdev generates Terraform configs from profiles. The documentation storage module would be invoked from a profile:

```hcl
# In gdev-generated Terraform
module "docs_storage" {
  source = "./modules/gdev-docs-storage"

  resource_group_name        = azurerm_resource_group.gdev.name
  location                   = var.azure_location
  storage_account_name       = "${var.org_prefix}docs"
  storage_sku                = var.docs_storage_sku  # Default: Standard_LRS
  entra_reader_group_id      = data.azuread_group.developers.object_id
  entra_contributor_group_id = data.azuread_group.devops.object_id
  enable_private_endpoint    = var.enable_private_endpoints
  private_endpoint_subnet_id = var.enable_private_endpoints ? azurerm_subnet.private.id : ""
  allowed_ip_ranges          = var.office_ip_ranges

  tags = merge(var.default_tags, {
    managed-by = "gdev"
    cost-center = "engineering"
  })
}
```

### 3.4 Automation for Uploading/Updating Documentation

```bash
#!/usr/bin/env bash
# gdev-docs-update.sh — Upload/update ZIM and DevDocs files
# Run by DevOps team or CI pipeline

STORAGE_ACCOUNT="highspringdocs"
ZIM_CONTAINER="zim-files"
DEVDOCS_CONTAINER="devdocs"

# Authenticate (CI uses service principal via env vars; local uses az login)
# DefaultAzureCredential handles this automatically via az cli

# Download latest ZIM files from Kiwix
ZIM_MIRROR="https://download.kiwix.org/zim"
declare -A ZIM_FILES=(
  ["unix"]="stackexchange/unix.stackexchange.com_en_all"
  ["serverfault"]="stackexchange/serverfault.com_en_all"
  ["devops"]="stackexchange/devops.stackexchange.com_en_all"
  ["softwareengineering"]="stackexchange/softwareengineering.stackexchange.com_en_all"
)

for name in "${!ZIM_FILES[@]}"; do
  # Find latest ZIM file URL from Kiwix library
  LATEST=$(curl -s "${ZIM_MIRROR}/${ZIM_FILES[$name]}/" | \
    grep -oP 'href="[^"]+\.zim"' | tail -1 | tr -d 'href="')

  echo "Uploading ${name}: ${LATEST}"
  # Download and upload to Azure Blob Storage
  curl -sL "${ZIM_MIRROR}/${ZIM_FILES[$name]}/${LATEST}" | \
    az storage blob upload \
      --account-name "$STORAGE_ACCOUNT" \
      --container-name "$ZIM_CONTAINER" \
      --name "${name}.zim" \
      --data @- \
      --auth-mode login \
      --overwrite
done

# DevDocs: extract from Docker image and upload
docker pull ghcr.io/freecodecamp/devdocs:latest
CONTAINER_ID=$(docker create ghcr.io/freecodecamp/devdocs:latest)
docker cp "${CONTAINER_ID}:/devdocs/public/docs" /tmp/devdocs-data
docker rm "$CONTAINER_ID"

az storage blob upload-batch \
  --account-name "$STORAGE_ACCOUNT" \
  --destination "$DEVDOCS_CONTAINER" \
  --source /tmp/devdocs-data \
  --auth-mode login \
  --overwrite

rm -rf /tmp/devdocs-data
```

---

## 4. Architecture Options Compared

### Option A: BlobFuse2 Local Mount (Recommended)

```
Developer Machine (NixOS)
├── az login → token cached
├── blobfuse2 mount ~/.local/share/gdev/azure-docs
│   └── (local SSD cache of Azure Blob Storage)
├── openzim-mcp → reads ZIM files from mount point
├── devdocs-mcp → reads JSON files from mount point
└── Claude Code → queries MCP servers
```

**Pros**:
- Transparent to MCP servers (just a file path)
- Local-disk performance after first cache fill
- Works offline with cached files
- Lowest Azure cost (~$2-5/month)
- No additional Azure compute needed

**Cons**:
- Requires local disk space for cache (up to 100 GB)
- First access to uncached files is slow (network download)
- BlobFuse2 FUSE overhead (negligible for this use case)

### Option B: kiwix-serve as Azure Container Apps Service

```
Azure Container Apps
├── kiwix-serve container (reads ZIM from mounted volume)
├── Entra ID authentication (built-in)
├── HTTPS endpoint: kiwix.highspring.io
└── Azure Blob Storage volume mount

Developer Machine
├── az login → token cached
├── mcp-server → HTTP queries to kiwix.highspring.io
│   └── Sends bearer token from az login
└── Claude Code → queries MCP server
```

**Pros**:
- No local storage needed (all data stays in Azure)
- Built-in Entra ID auth on Container Apps
- Can be shared across all developers without per-machine setup
- Centralized updates (deploy new ZIM files once)

**Cons**:
- Network latency for every search query (~20-100ms per round trip)
- Always-on compute cost (~$36-72/month)
- Requires custom MCP server that wraps kiwix-serve HTTP API
- Internet required (no offline access)

### Option C: Hybrid (Recommended for gdev)

```
Developer Machine (NixOS)
├── Local curated ZIM files (~5 GB, Nix-managed)
│   └── openzim-mcp reads these directly (no Azure needed)
├── BlobFuse2 mount for large/optional ZIM files
│   └── Full Stack Overflow (74 GB), etc.
└── DevDocs JSON files via BlobFuse2 or local subset

Azure (Optional)
├── Blob Storage with all ZIM + DevDocs files
├── Entra ID RBAC for access control
└── CI pipeline updates documentation periodically
```

**Pros**:
- Core docs always available offline
- Large corpora available on-demand via Azure
- Progressive: start local-only, add Azure when needed
- Minimal Azure cost
- Preserves security benefits of local-first

**Cons**:
- More complex setup (two data paths)
- Need to manage which docs are local vs Azure

---

## 5. kiwix-serve as Azure Service (Option B Deep Dive)

### 5.1 Container Apps Deployment

Azure Container Apps is preferred over raw Container Instances for kiwix-serve because it provides built-in Entra ID authentication, automatic scaling, and HTTPS ingress.

```hcl
# Terraform for kiwix-serve on Container Apps
resource "azurerm_container_app_environment" "docs" {
  name                = "docs-env"
  location            = var.location
  resource_group_name = var.resource_group_name
}

resource "azurerm_container_app" "kiwix" {
  name                         = "kiwix-serve"
  container_app_environment_id = azurerm_container_app_environment.docs.id
  resource_group_name          = var.resource_group_name
  revision_mode                = "Single"

  template {
    container {
      name   = "kiwix"
      image  = "ghcr.io/kiwix/kiwix-tools:latest"
      cpu    = 1.0
      memory = "2Gi"

      command = [
        "kiwix-serve",
        "--port", "8080",
        "--library", "/data/*.zim"
      ]

      volume_mounts {
        name      = "zim-data"
        mount_path = "/data"
      }
    }

    volume {
      name         = "zim-data"
      storage_type = "AzureFile"
      storage_name = "zim-storage"
    }

    min_replicas = 0  # Scale to zero when unused
    max_replicas = 2
  }

  ingress {
    external_enabled = true
    target_port      = 8080
    transport        = "http"
  }

  # Entra ID authentication configured via Azure Portal or ARM
}
```

### 5.2 Entra ID Protection

Container Apps has built-in Entra ID authentication:
1. Register an app in Entra ID
2. Configure Container Apps authentication provider (Microsoft)
3. Set to require authentication for all requests
4. Unauthenticated requests are redirected to Entra ID sign-in
5. Authenticated requests include bearer token in `X-MS-TOKEN-AAD-ACCESS-TOKEN` header

For MCP server access (daemon/service pattern):
1. Register a daemon app (no redirect URI)
2. Create client secret
3. Request tokens via OAuth 2.0 client credentials flow
4. Present bearer token to kiwix-serve endpoint

### 5.3 Alternative: Application Gateway with WAF

For enhanced security, kiwix-serve can be placed behind an Azure Application Gateway:
- JWT validation at the gateway (no custom code needed)
- Web Application Firewall (WAF) rules
- SSL termination
- However, adds ~$175/month (Application Gateway cost)
- **Overkill for internal documentation serving** — Container Apps built-in auth is sufficient

### 5.4 Latency Impact

kiwix-serve search query latency over the network:
- Local kiwix-serve: <10ms per search
- Same-region Azure: 20-50ms per search
- Cross-region: 50-200ms per search
- With CDN caching (Azure Front Door): 5-20ms for cached content

For Claude Code tool calls, 50ms added latency per documentation lookup is acceptable — each MCP tool call already involves LLM processing time measured in seconds.

---

## 6. DevDocs as Azure-Hosted Service

### 6.1 Option A: JSON Files in Blob Storage (Recommended)

DevDocs data is just JSON files (`index.json`, `db.json`, `meta.json` per doc set). These are small enough to serve directly from Blob Storage:

```
Azure Blob Storage / devdocs container
├── python~3.12/
│   ├── index.json    (~100 KB)
│   ├── db.json       (~5 MB)
│   └── meta.json     (~1 KB)
├── typescript~5.4/
│   ├── index.json
│   ├── db.json
│   └── meta.json
└── ... (100+ doc sets)
```

MCP server access pattern:
1. MCP server uses DefaultAzureCredential to get a bearer token
2. Uses Azure Storage SDK (`azure-storage-blob`) to download JSON files
3. Caches downloaded JSON files locally (they change rarely)
4. Searches `index.json` in memory, returns content from `db.json`

```python
from azure.identity import DefaultAzureCredential
from azure.storage.blob import BlobServiceClient

credential = DefaultAzureCredential()
blob_service = BlobServiceClient(
    account_url="https://highspringdocs.blob.core.windows.net",
    credential=credential
)

# Download index.json for a doc set
container = blob_service.get_container_client("devdocs")
blob = container.get_blob_client("python~3.12/index.json")
index_data = json.loads(blob.download_blob().readall())
```

Or, with BlobFuse2, simply:
```python
with open("/mnt/gdev-docs/devdocs/python~3.12/index.json") as f:
    index_data = json.load(f)
```

### 6.2 Option B: DevDocs Docker Image on Container Apps

Running the full DevDocs web application in Azure:
- Image: `ghcr.io/freecodecamp/devdocs:latest` (~3.5 GB)
- Provides search UI and content serving
- Can be protected with Entra ID auth
- **Not recommended**: DevDocs has no API (MCP servers would scrape the web UI), heavy image size, Ruby runtime overhead. Direct JSON file access is simpler and faster.

### 6.3 Caching Strategy

DevDocs data changes infrequently (monthly Docker image updates). Recommended caching:
- BlobFuse2 file cache with 7-day TTL for JSON files
- Or: MCP server downloads JSON files to `~/.cache/gdev/devdocs/` on startup, checks blob metadata for changes periodically
- A 20-doc-set subset (typical team needs) is ~200-500 MB — feasible for local caching

---

## 7. End-to-End Developer Experience

### 7.1 Initial Setup (`gdev setup`)

```
1. Developer runs `gdev setup`
2. gdev detects Azure profile, runs `az login` if not authenticated
3. gdev applies Terraform module (creates storage account, RBAC, etc.)
4. gdev configures BlobFuse2:
   a. Writes blobfuse config to ~/.config/gdev/blobfuse.yaml
   b. Creates mount point at ~/.local/share/gdev/azure-docs
   c. Mounts via systemd user service (auto-mount on login)
5. gdev downloads "core docs" locally (~5 GB curated ZIM files)
6. gdev configures MCP servers in .mcp.json:
   - openzim-mcp pointing to BlobFuse2 mount + local ZIM dir
   - devdocs-mcp pointing to BlobFuse2 mount devdocs path
7. Developer starts Claude Code — documentation available immediately
```

### 7.2 Daily Use

```
1. Developer logs in to NixOS
2. Systemd user service auto-mounts BlobFuse2 (uses cached az login token)
3. Claude Code queries documentation via MCP servers
4. MCP servers read from local cache (fast) or download on first access
5. Token refresh is automatic (handled by Azure CLI credential)
```

### 7.3 Token Expiry Handling

- **Azure CLI access token**: Expires after ~1 hour, auto-refreshed by Azure CLI
- **Azure CLI refresh token**: Expires after ~90 days of inactivity
- **If refresh token expires**: BlobFuse2 mount fails silently on new file access; cached files still work
- **gdev health check**: `gdev status` should verify `az account show` succeeds and warn if re-auth needed
- **Graceful degradation**: MCP servers fall back to locally-cached ZIM files if mount is unavailable

### 7.4 Offline Behavior

When Azure is unreachable:
1. BlobFuse2 serves files from local cache (if previously accessed)
2. openzim-mcp continues to read cached ZIM files normally
3. New/uncached files are unavailable
4. gdev falls back to locally-stored "core docs" set
5. No error in Claude Code — MCP servers simply have reduced coverage

---

## 8. Cost Analysis (~20 Developers)

### 8.1 Storage-Only (Option A: BlobFuse2)

| Item | Monthly Cost | Notes |
|------|-------------|-------|
| Blob Storage (Hot, LRS) | $1.80 | 100 GB @ $0.018/GB |
| Transactions (reads) | ~$0.50 | ~100K read operations @ $0.005/10K |
| Egress (first fill) | $8.70 | 100 GB @ $0.087/GB (one-time per developer) |
| Egress (ongoing) | ~$1-3 | Minimal after caching; only for updates |
| **Monthly total** | **~$4-6** | After initial cache fill |

**Initial fill cost per developer**: ~$8.70 for 100 GB download (one-time). With 20 developers: ~$174 one-time. After that, ongoing egress is minimal because BlobFuse2 caches locally.

### 8.2 Container Service (Option B: kiwix-serve)

| Item | Monthly Cost | Notes |
|------|-------------|-------|
| Blob Storage | $1.80 | Same as above |
| Container Apps (1 vCPU, 2 GB, always-on) | ~$36 | $29.57 vCPU + $6.50 memory |
| Azure Files (for volume mount) | ~$2.80 | 100 GB Hot tier |
| Egress (API responses) | ~$5-10 | Search results, article content |
| **Monthly total** | **~$46-51** | |

With scale-to-zero (Container Apps min_replicas=0):
| Item | Monthly Cost | Notes |
|------|-------------|-------|
| Container Apps (usage-based) | ~$5-15 | Depends on usage patterns |
| **Monthly total** | **~$15-25** | With scale-to-zero |

### 8.3 Hybrid (Option C: Recommended)

| Item | Monthly Cost | Notes |
|------|-------------|-------|
| Blob Storage | $1.80 | 100 GB |
| Transactions | ~$0.50 | |
| Egress | ~$2-5 | Mostly cached |
| **Monthly total** | **~$5-8** | |

Plus: developers have ~5 GB of core docs locally (no Azure cost).

### 8.4 Comparison Summary

| Approach | Monthly Cost | Offline? | Setup Complexity | Latency |
|----------|-------------|----------|-----------------|---------|
| BlobFuse2 only | $5-8 | Yes (cached) | Medium | Local SSD after cache |
| kiwix-serve always-on | $46-51 | No | Low | 20-50ms per query |
| kiwix-serve scale-to-zero | $15-25 | No | Low | 20-50ms + cold start |
| Hybrid (recommended) | $5-8 | Yes | Medium | Local SSD |
| Local-only (no Azure) | $0 | Yes | Low | Local SSD |

### 8.5 Azure Front Door (if Multi-Office)

If Highspring has multiple offices and wants CDN-cached access:
- Azure Front Door Standard: $35/month base
- Plus per-request charges: ~$0.01/10K requests
- Reduces cross-region latency for kiwix-serve API
- **Only worthwhile if running kiwix-serve as a service**
- Not needed for BlobFuse2 approach (each developer has local cache)

---

## 9. Network Security

### 9.1 Public Access with IP Restriction (Simple)

For a consulting firm with known office IPs:
```hcl
network_rules {
  default_action = "Deny"
  ip_rules       = ["203.0.113.0/24"]  # Office IP range
  bypass         = ["AzureServices"]
}
```

Developers on VPN or in-office can access directly. Remote developers need VPN.

### 9.2 Private Endpoint (Enterprise)

For maximum security:
- Private endpoint in the firm's Azure VNet
- No public internet exposure
- Developers access via VPN or ExpressRoute
- More complex setup but eliminates public attack surface

### 9.3 Recommendation

**Start with public access + IP restriction** (simple, works with BlobFuse2 from any network). Upgrade to private endpoints if security requirements demand it. The documentation content is not sensitive — it is publicly available Stack Overflow and DevDocs content — so the primary risk is unauthorized Azure resource consumption, not data exfiltration.

---

## 10. Architectural Recommendations

### 10.1 Recommended Architecture (Hybrid, Option C)

1. **Azure Blob Storage** with Hot tier, LRS redundancy, Entra ID RBAC
2. **BlobFuse2** on developer machines for transparent Azure mount
3. **Local "core docs"** (~5 GB curated SE sites) managed via Nix
4. **gdev Terraform module** provisions storage, RBAC, containers
5. **DefaultAzureCredential** (via `az login`) for authentication
6. **Systemd user service** auto-mounts BlobFuse2 on login
7. **CI pipeline** updates ZIM/DevDocs files periodically

### 10.2 Migration Path

Phase 1: Local-only (current spike recommendation)
- openzim-mcp with locally-stored curated ZIM files (~5 GB)
- DevDocs JSON files stored locally
- No Azure dependency

Phase 2: Azure supplement (this research)
- Add Blob Storage for large corpora (full SO, all DevDocs)
- BlobFuse2 mount for transparent access
- Terraform module in gdev

Phase 3: Optional centralized service
- kiwix-serve on Container Apps (if demand justifies cost)
- Entra ID auth on the endpoint
- Useful for web-based access or non-NixOS developers

### 10.3 Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Storage service | Blob Storage | Cheapest, BlobFuse2 solves random access, Entra ID RBAC |
| Mount technology | BlobFuse2 (file cache) | In nixpkgs, transparent to MCP servers, local-disk perf |
| Authentication | DefaultAzureCredential | Unified chain: az login for devs, managed identity for Azure |
| Network security | Public + IP restriction | Simple, sufficient for public documentation content |
| Redundancy | LRS | Documentation is reproducible; no need for geo-redundancy |
| Container service | Container Apps | Only if needed; built-in Entra ID auth, scale-to-zero |
| DevDocs access | BlobFuse2 (same mount) | Consistent access pattern, no separate service needed |

---

## Sources

All raw sources saved to `docs/`:
- `azure-nfs-storage-comparison.md` — Azure Blob vs Files vs NetApp for NFS (Microsoft Learn)
- `blobfuse2-overview.md` — BlobFuse2 architecture and caching modes (Microsoft Learn + GitHub Wiki)
- `azure-default-credential-python.md` — DefaultAzureCredential Python API reference (Microsoft Learn)
- `azure-mcp-server-python.md` — Azure MCP Server with Python integration (Microsoft Learn)
- `azure-storage-pricing-2026.md` — Azure Storage pricing breakdown (nOps)
- `azure-terraform-storage-rbac.md` — Terraform storage account RBAC examples (codewithme.cloud)
- `blobfuse-nix-package.md` — BlobFuse Nix package details (MyNixOS)
- `azure-container-apps-entra-auth.md` — Container Apps Entra ID authentication (Microsoft Learn)
- `azure-container-instances-pricing.md` — ACI pricing details (pump.co)
- `azure-files-vs-netapp-files.md` — Azure Files vs NetApp Files comparison (Microsoft Learn)

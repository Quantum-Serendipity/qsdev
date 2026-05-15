# Multi-Cloud Terraform Abstraction for Self-Hosted Documentation Corpora

## Executive Summary

Hosting documentation corpora (ZIM files for Stack Overflow/SE, DevDocs JSON, ~100 GB total) can be abstracted across cloud providers, but the practical best approach is **not** a single cloud-agnostic Terraform module. Instead, gdev should generate **per-provider Terraform modules** that share a common interface contract, with a **universal S3 API storage layer** as the unifying abstraction. The S3 API is supported natively by AWS, has interoperability modes on GCP and (via proxy) Azure, and is the native protocol for MinIO, DigitalOcean Spaces, Hetzner Object Storage, Oracle Cloud, and dozens of other providers. Rclone (with `--vfs-cache-mode full`) is the recommended universal FUSE mount tool, supporting 50+ backends with a single binary. For SSO/IAM, each cloud has its own credential chain pattern (boto3 chain, DefaultAzureCredential, ADC) that follows the same "try env vars → CLI token → managed identity" cascade — gdev should detect which cloud CLI is configured and use it, not try to abstract auth.

**Cost for ~100 GB, ~20 developers**: $2-5/month storage on any provider; compute for hosted kiwix-serve adds $18-35/month for always-on Fargate/Cloud Run; FUSE-mount approach has zero compute cost beyond developer workstations.

---

## 1. AWS Equivalent Architecture

### Storage Options

| Option | Type | Random Read | Cost (100 GB/mo) | Best For |
|--------|------|-------------|-------------------|----------|
| **S3 Standard** | Object | Via FUSE mount | $2.30 | Default choice, cheapest |
| **S3 Standard-IA** | Object | Via FUSE mount | $1.25 | Infrequently accessed corpora |
| **S3 Files** (new) | Managed NFS 4.2 | Native | ~$30+ (EFS-backed) | Full POSIX, highest performance |
| **EFS** | Managed NFS | Native | ~$30/mo (Standard) | Multi-instance shared access |
| **FSx for Lustre** | High-perf parallel FS | Native, sub-ms | $140+/mo (1.2 TB min) | Overkill for this use case |

**Recommendation**: S3 Standard. The dataset is read-heavy (ZIM files are written once, read many times) and 100 GB at $2.30/month is an order of magnitude cheaper than file-system options. S3 Files would be ideal for POSIX compatibility but is new and expensive. EFS/FSx are grossly over-provisioned for serving static documentation.

### S3 FUSE Mount Options for ZIM Random I/O

ZIM files require random read access (seeking to cluster offsets within a 74 GB file). This is a critical compatibility question.

| Tool | Random Read | Performance | S3-Compatible | Notes |
|------|------------|-------------|---------------|-------|
| **Mountpoint for S3** | Yes (random reads supported) | 6-8x faster than s3fs | AWS only | No random writes, limited POSIX. Official AWS. |
| **s3fs-fuse** | Yes | ~100 MB/s max, high latency per op | Any S3 endpoint | Full POSIX, but slowest option |
| **rclone mount** | Yes (with `--vfs-cache-mode full`) | Good with local SSD cache | 50+ backends | Universal, sparse file caching, configurable read-ahead |
| **Goofys** | Yes | Fastest raw throughput | AWS S3 | Minimal POSIX, no maintenance since 2022 |

**Critical finding for ZIM access**: All FUSE tools support random reads on S3, but performance varies dramatically. ZIM's access pattern is: read header (fixed offset), then seek to cluster pointer table, then seek to specific cluster, decompress ~1-2 MiB. This is a sparse random read pattern, not sequential.

- **Mountpoint for S3**: Supports random reads natively. Best for AWS-only deployments. No local caching of its own, but kernel page cache helps for repeated reads.
- **rclone mount with `--vfs-cache-mode full`**: Downloads file regions on demand into sparse local cache files. With `--vfs-read-ahead 128M` and cache on local SSD, ZIM random access would be fast after first access (cached) and tolerable on first access (~100-500ms per seek to S3). This is the universal option.
- **s3fs-fuse**: Works but slowest. Every seek hits S3 unless local cache is configured. 10-100ms per operation.

**Recommendation**: rclone mount for cross-provider compatibility, Mountpoint for AWS-only deployments.

### IAM: AWS IAM Identity Center (SSO)

Developer authentication flow:
1. Admin configures IAM Identity Center with org's IdP (Okta, Azure AD, etc.)
2. Developer's `~/.aws/config` has SSO session profile:
   ```ini
   [profile gdev-docs]
   sso_session = corp-sso
   sso_account_id = 111122223333
   sso_role_name = GdevDocsReader

   [sso-session corp-sso]
   sso_region = us-east-1
   sso_start_url = https://myorg.awsapps.com/start
   sso_registration_scopes = sso:account:access
   ```
3. `aws sso login --profile gdev-docs` opens browser for SSO
4. Token cached to `~/.aws/sso/cache/` with auto-refresh
5. boto3/AWS SDK/rclone/s3fs all transparently use cached SSO credentials

**For local MCP servers**: The boto3 credential chain automatically resolves SSO credentials: environment variables → config file → SSO token → instance profile. No special code needed — the MCP server just uses `boto3.Session(profile_name='gdev-docs')`.

### Compute: Hosted kiwix-serve / DevDocs

| Option | Monthly Cost (0.5 vCPU, 1 GB) | Storage | Cold Start | Auth |
|--------|-------------------------------|---------|------------|------|
| **ECS Fargate** | $17.87 (on-demand), $8.94 (Savings Plan) | 20 GB ephemeral free, EFS mount for data | None (always-on) | ALB + Cognito or IAM auth |
| **ECS Fargate Spot** | ~$5-11 | Same | 2-min interruption risk | Same |
| **Lambda + container** | Pay-per-invoke | 10 GB /tmp | Yes (seconds) | API Gateway + IAM |
| **App Runner** | ~$18/mo | No persistent storage | Auto scale-to-zero | Built-in auth via IAM |

**Recommendation**: ECS Fargate with EFS mount for the ZIM/DevDocs data. The 74 GB ZIM file rules out Lambda (10 GB limit) and App Runner (no persistent storage). Fargate + EFS is the cleanest pattern: the container runs kiwix-serve, mounts EFS for data, sits behind an ALB with Cognito for SSO authentication.

### Cost Estimate: AWS, ~100 GB, ~20 Developers

| Component | Monthly Cost |
|-----------|-------------|
| S3 Standard (100 GB) | $2.30 |
| S3 GET requests (~50K/mo) | $0.02 |
| Data transfer (intra-region, FUSE mount) | $0.00 |
| **FUSE-mount approach total** | **~$2.32/mo** |
| | |
| ECS Fargate (0.5 vCPU, 1 GB, always-on) | $17.87 |
| EFS (100 GB, Standard) | $30.00 |
| ALB + Cognito | $16.20 + ~$1 |
| **Hosted-service approach total** | **~$65/mo** |

---

## 2. GCP Equivalent Architecture

### Storage Options

| Option | Type | Random Read | Cost (100 GB/mo) | Best For |
|--------|------|-------------|-------------------|----------|
| **Cloud Storage Standard** | Object | Via gcsfuse/rclone | $2.00 | Default choice |
| **Cloud Storage Nearline** | Object | Via gcsfuse/rclone | $1.00 | Infrequent access |
| **Filestore Basic HDD** | Managed NFS | Native | ~$200/mo (1 TB min) | Overkill, expensive minimum |
| **Filestore Basic SSD** | Managed NFS | Native | ~$750/mo (2.5 TB min) | Overkill |

**Recommendation**: Cloud Storage Standard at $2.00/month. Filestore's 1 TB minimum makes it absurdly expensive for 100 GB of documentation.

### gcsfuse for ZIM Random I/O

gcsfuse (Google's official FUSE driver) mounts GCS buckets as local filesystems:

- **Random reads**: Supported but "poor performance due to API call overhead" for small random reads. Each seek requires a new HTTP range request.
- **File caching**: gcsfuse supports local file caching that can achieve "up to 2.3x faster" access and "3.4x higher throughput" by caching hot data to local SSD.
- **Parallel downloads**: Configurable but "applications with high read-parallelism (>8 threads) may encounter lower performance."
- **Metadata caching**: Configurable to avoid repeated metadata requests.

**Configuration for ZIM workload**:
```bash
gcsfuse --file-cache-max-size-mb=80000 \
        --file-cache-dir=/tmp/gcsfuse-cache \
        --stat-cache-ttl=1h \
        --type-cache-ttl=1h \
        gdev-docs-bucket /mnt/gdev-docs
```

**Alternative**: rclone mount with `--vfs-cache-mode full` works with GCS and provides comparable performance with a simpler cross-provider story.

### IAM: GCP Authentication

Developer flow:
1. Admin configures Workforce Identity Federation with org's IdP (or uses Google Workspace accounts directly)
2. Developer runs `gcloud auth login` (opens browser for Google/SSO login)
3. For Application Default Credentials: `gcloud auth application-default login`
4. ADC JSON stored at `~/.config/gcloud/application_default_credentials.json`
5. All GCP client libraries automatically discover ADC

**ADC resolution order**: `GOOGLE_APPLICATION_CREDENTIALS` env var → `~/.config/gcloud/application_default_credentials.json` → metadata server (on GCE/GKE).

**For local MCP servers**: Any GCP client library (Python, Node.js, Go) automatically uses ADC. rclone also supports GCS natively with service account or ADC auth.

**GCS S3 compatibility**: GCS provides an XML API compatible with S3 tools using HMAC keys. Endpoint: `https://storage.googleapis.com`. s3fs-fuse and rclone can access GCS via S3 protocol, but native GCS access is preferred for better performance and feature support.

### Compute: Hosted kiwix-serve

| Option | Monthly Cost | Storage | Auth |
|--------|-------------|---------|------|
| **Cloud Run** (1 vCPU, 1 GB, always-on) | ~$30-45/mo | GCS via gcsfuse CSI, or Filestore NFS mount | IAP (Identity-Aware Proxy) |
| **Cloud Run** (scale-to-zero) | Pay-per-request, ~$5-15/mo | Same | Same |
| **GKE Autopilot** | ~$75+/mo (min cluster cost) | Filestore, GCS CSI | IAP |

**Recommendation**: Cloud Run with min-instances=1 for always-on. Cloud Run supports mounting GCS buckets via the gcsfuse CSI driver (built-in), and supports Filestore NFS mounts. Identity-Aware Proxy (IAP) provides SSO-gated access with zero application-level auth code.

### Cost Estimate: GCP, ~100 GB, ~20 Developers

| Component | Monthly Cost |
|-----------|-------------|
| Cloud Storage Standard (100 GB) | $2.00 |
| GCS operations (~50K Class B reads) | $0.02 |
| **FUSE-mount approach total** | **~$2.02/mo** |
| | |
| Cloud Run (1 vCPU, 1 GB, always-on) | ~$40.00 |
| GCS (same bucket) | $2.00 |
| IAP | Free (included) |
| **Hosted-service approach total** | **~$42/mo** |

---

## 3. Other Cloud Providers

### DigitalOcean Spaces

- **Type**: S3-compatible object storage
- **Pricing**: $5/month flat includes 250 GB storage + 1 TB outbound transfer. Additional: $0.02/GB/mo storage, $0.01/GB egress. No per-request fees.
- **FUSE mount**: s3fs-fuse with `--url=https://nyc3.digitaloceanspaces.com`, or rclone
- **Auth**: API key/secret (no SSO integration)
- **Compute**: DigitalOcean App Platform or Droplets for kiwix-serve
- **For gdev**: Good budget option. 100 GB fits in the $5/month base plan. No enterprise SSO but fine for small teams.

### Hetzner

**Object Storage** (S3-compatible):
- ~$5.20/TB/month, includes 1 TB storage + 1 TB egress in base plan
- Free API calls (PUT, GET, DELETE)
- S3-compatible API, works with s3fs-fuse, rclone, boto3
- EU-only regions (Germany, Finland) — good for GDPR
- 100 GB: ~$0.52/month (cheapest option)

**Storage Box** (file-based):
- 1 TB for ~EUR 3.49/mo, 10 TB for ~EUR 20.80/mo
- Protocols: SFTP, SCP, Samba/CIFS, rsync, WebDAV, rclone — **no NFS**
- No S3 API
- Mountable via rclone or CIFS but not suitable for random I/O on ZIM files

**For gdev**: Hetzner Object Storage is compelling for EU-based teams: cheapest storage, S3-compatible, GDPR-compliant. Storage Box lacks NFS and random access performance.

### Oracle Cloud Infrastructure (OCI)

- **Always Free tier**: 20 GB object storage (never expires), 10 TB/month transfer, 50K API requests
- **S3 compatibility**: Full S3-compatible API via Customer Secret Keys
- **Paid tier**: $0.0255/GB/mo Standard, but egress is only $0.0085/GB (10x cheaper than AWS)
- **Compute**: Always Free includes 2 ARM VMs (4 cores, 24 GB RAM total) — could run kiwix-serve for free
- **For gdev**: The Always Free tier is too small for 100 GB of docs, but the free ARM VMs are interesting for running kiwix-serve. Paid storage is slightly more expensive than AWS/GCP.

### MinIO (Self-Hosted S3-Compatible)

- **Type**: Self-hosted object storage with full S3 API
- **License**: AGPLv3 (changed from Apache 2.0 — commercial use requires license or AGPL compliance)
- **Deployment**: Single binary, Docker, Kubernetes, NixOS module available
- **Performance**: Native disk I/O — no cloud latency, fastest option for FUSE mount
- **Auth**: Built-in IAM with S3-compatible policies, LDAP/AD integration
- **s3fs/rclone**: Works by changing endpoint URL: `--url=http://minio:9000`
- **For gdev**: Best option for on-prem/homelab. Zero egress fees, full control. The AGPL license change is a concern — alternatives like [GarageHQ](https://garagehq.deuxfleurs.fr/) (AGPL but designed for self-hosting) or SeaweedFS (Apache 2.0) exist.

### S3-Compatible Provider Summary

Any provider with S3-compatible API can be targeted by a single Terraform S3 module:

| Provider | S3-Compatible | Endpoint Format | Notes |
|----------|--------------|-----------------|-------|
| AWS S3 | Native | `s3.amazonaws.com` | Reference implementation |
| GCS | Yes (XML API) | `storage.googleapis.com` | HMAC keys required |
| Azure Blob | No (needs proxy) | N/A | AZS3-Proxy or BlobFuse2 instead |
| DigitalOcean Spaces | Yes | `nyc3.digitaloceanspaces.com` | Full S3 compat |
| Hetzner Object Storage | Yes | `fsn1.your-objectstorage.com` | Full S3 compat |
| Oracle OCI | Yes | `*.compat.objectstorage.*.oraclecloud.com` | Customer Secret Keys |
| MinIO | Native S3 | `minio-host:9000` | Self-hosted reference |
| Backblaze B2 | Yes | `s3.us-west-004.backblazeb2.com` | $6/TB/mo |
| Wasabi | Yes | `s3.wasabisys.com` | $6.99/TB/mo, no egress fees |
| Cloudflare R2 | Yes | `*.r2.cloudflarestorage.com` | No egress fees |

---

## 4. Terraform Module Abstraction Patterns

### Pattern Comparison

| Pattern | Pros | Cons | Fit for gdev |
|---------|------|------|-------------|
| **Single module with `cloud_provider` variable** | Simple API, one interface | Complex conditionals, hard to test, provider limitation (can't conditionally select providers in Terraform) | Poor — Terraform doesn't support conditional providers well |
| **Separate modules per cloud** | Clean, testable, full provider features | Code duplication, no shared interface enforcement | Good — matches Terraform's design philosophy |
| **Wrapper module dispatching to sub-modules** | Common interface, `count`-based selection | Complexity, all providers initialized even if unused, provider config issues | Fair — works but adds indirection |
| **CDKTF (TypeScript)** | Full programming language, loops/conditionals, gdev is already TypeScript | Transpiles to Terraform JSON, extra build step, less mature ecosystem | Interesting — matches gdev's stack |
| **Pulumi** | Real programming languages, runtime provider selection | Different tool entirely, separate state management, learning curve | Poor — adds a new tool |
| **Crossplane** | Kubernetes-native, declarative | Requires Kubernetes cluster, heavy for this use case | Poor — wrong abstraction level |

### HashiCorp's Official Guidance

From the Terraform module composition docs: "Terraform deliberately avoids abstracting over similar services from different vendors." Instead, HashiCorp recommends:

1. Define common **object types** representing your concepts (e.g., `storage_config`, `auth_config`)
2. Create **per-provider modules** that accept the same input variable shapes
3. Use **dependency inversion** — modules receive dependencies, don't create them
4. Keep module hierarchies **flat** (one level of child modules)

### Prior Art: CloudPosse, Gruntwork, terraform-aws-modules

- **CloudPosse**: 600+ modules, all AWS-specific. Uses a shared "context" object passed between modules for consistent labeling/tagging. Resource Factory pattern uses YAML-driven configuration. No multi-cloud modules.
- **Gruntwork/Terragrunt**: Terragrunt adds orchestration on top of Terraform (DRY configs, dependency management, multi-account). Gruntwork's modules are per-provider. Multi-cloud is achieved by having parallel Terragrunt configs that invoke different provider modules.
- **terraform-aws-modules**: Community modules, AWS-only. Pattern: focused modules per service (VPC, EKS, S3, etc.), composed in root modules.

**Key insight**: None of the major Terraform module ecosystems attempt true cloud-agnostic modules. They all provide per-provider modules with consistent design patterns. The abstraction happens at the configuration/orchestration layer (Terragrunt, CDKTF, or application code like gdev).

### Recommended Pattern for gdev

**gdev is the abstraction layer, not Terraform**. Since gdev already generates Terraform configs from profiles and templates, the abstraction should live in gdev's Go code:

```
gdev/
  templates/
    terraform/
      docs-hosting/
        aws/
          main.tf        # S3 bucket + IAM policy + optional Fargate
          variables.tf   # Common interface contract
          outputs.tf     # storage_endpoint, mount_command, mcp_config
        azure/
          main.tf        # Blob Storage + SAS token + optional ACI
          variables.tf
          outputs.tf
        gcp/
          main.tf        # GCS bucket + IAM + optional Cloud Run
          variables.tf
          outputs.tf
        s3-compatible/
          main.tf        # Generic S3 bucket (works for DO, Hetzner, MinIO, etc.)
          variables.tf
          outputs.tf
        local/
          main.tf        # No-op (just outputs local paths)
          variables.tf
          outputs.tf
```

gdev selects the right template based on `--profile` or detected cloud CLI:
- `qsdev init --profile enterprise-aws` → generates `aws/` Terraform
- `qsdev init --profile enterprise-azure` → generates `azure/` Terraform
- `qsdev init --profile enterprise-gcp` → generates `gcp/` Terraform
- `qsdev init --profile self-hosted` → generates `s3-compatible/` targeting MinIO
- `qsdev init --profile local` → generates `local/` (no cloud, local paths only)

---

## 5. Storage Abstraction Layer

### S3 API as the Universal Protocol

The S3 API is the de facto standard for object storage. Coverage:

| Provider | S3 API Support | Limitations |
|----------|---------------|-------------|
| AWS S3 | Native, complete | None |
| GCS | XML API interop, HMAC keys | Not all S3 ops supported, needs HMAC config |
| Azure Blob | **Not native** — requires AZS3-Proxy or MinIO gateway (deprecated) | Significant gap; BlobFuse2 is Azure's native FUSE |
| DigitalOcean Spaces | Full | Minor missing features |
| Hetzner | Full | EU regions only |
| Oracle OCI | Full | Customer Secret Keys config |
| MinIO | Native, complete | Self-hosted only |
| Backblaze B2 | Full | — |
| Wasabi | Full | — |
| Cloudflare R2 | Full | No egress, some missing ops |

**Could gdev target S3 API exclusively?** Almost. The S3 API covers AWS, GCS (with HMAC), MinIO, DigitalOcean, Hetzner, Oracle, Backblaze, Wasabi, Cloudflare, and others — probably 80-90% of potential adopters. The gap is **Azure**, which requires BlobFuse2 or a proxy. Since Azure is a major enterprise cloud, gdev needs explicit Azure support (handled by the parallel agent's research) alongside the S3-compatible path.

**Recommended strategy**: Two storage paths:
1. **S3-compatible path** (default): Works for AWS, GCS, MinIO, DO, Hetzner, Oracle, etc. One Terraform module, one FUSE tool (rclone or s3fs).
2. **Azure-native path**: BlobFuse2 for FUSE mount, Azure Blob SDK for API access. Separate Terraform module.

### FUSE Mount Abstraction

| Tool | Backends | Random Read | Caching | Cross-Platform | ZIM Suitability |
|------|----------|-------------|---------|----------------|----------------|
| **rclone mount** | 50+ | Yes (with `--vfs-cache-mode full`) | Sparse file cache on local disk | Linux, macOS, Windows | **Best universal option** |
| **s3fs-fuse** | Any S3 endpoint | Yes | Optional local cache | Linux, macOS | Good for S3-only; slow |
| **Mountpoint for S3** | AWS S3 only | Yes | Metadata only | Linux only | Best for AWS-only |
| **gcsfuse** | GCS only | Yes | File cache | Linux, macOS (beta) | Best for GCP-only |
| **BlobFuse2** | Azure Blob only | Yes | File cache + streaming | Linux only | Best for Azure-only |

**rclone mount is the universal FUSE tool for gdev.** Configuration for ZIM workloads:

```bash
# Mount S3-compatible storage (works for AWS, GCS, MinIO, DO, Hetzner, etc.)
rclone mount gdev-docs: /mnt/gdev-docs \
  --vfs-cache-mode full \
  --vfs-cache-max-size 100G \
  --vfs-read-ahead 128M \
  --cache-dir /var/cache/gdev-docs \
  --dir-cache-time 1h \
  --no-modtime \
  --read-only \
  --daemon
```

After first access, ZIM file regions are cached locally in sparse files. Subsequent reads hit the local SSD cache — near-native performance. The 100 GB `--vfs-cache-max-size` ensures the entire corpus can be cached locally if disk space allows.

**Performance expectation for ZIM random I/O via rclone**:
- First access to a ZIM cluster: 100-500ms (HTTP range request to cloud storage)
- Subsequent access to same cluster: <1ms (local SSD cache)
- Full corpus cached after normal usage patterns: hours to days depending on access breadth
- For 20 developers sharing the same S3 bucket, each workstation maintains its own local cache

### Rclone vs Provider-Native FUSE

gdev should prefer rclone as the default FUSE mount tool (one tool, all providers) but allow provider-native FUSE for performance-critical deployments:

```yaml
# .gdev/config.yaml
docs_hosting:
  provider: aws
  mount_tool: rclone    # or: mountpoint-s3, gcsfuse, blobfuse2, native
  storage_endpoint: s3://gdev-docs-corp
  cache_dir: /var/cache/gdev-docs
  cache_size: 100G
```

---

## 6. SSO/IAM Abstraction

### Credential Chain Patterns (Strikingly Similar)

All three major clouds use the same cascading credential resolution pattern:

| Step | AWS (boto3) | Azure (DefaultAzureCredential) | GCP (ADC) |
|------|------------|-------------------------------|-----------|
| 1 | Environment variables (`AWS_ACCESS_KEY_ID`, etc.) | EnvironmentCredential | `GOOGLE_APPLICATION_CREDENTIALS` env var |
| 2 | Shared config file (`~/.aws/config`) | WorkloadIdentityCredential | — |
| 3 | SSO cached token (`~/.aws/sso/cache/`) | ManagedIdentityCredential | — |
| 4 | Instance profile (EC2/ECS) | AzureCliCredential (`az login`) | ADC file (`gcloud auth application-default login`) |
| 5 | — | AzureDeveloperCliCredential | Metadata server (GCE/GKE) |

**The pattern is universal**: try env vars → try cached SSO/CLI token → try managed identity. For local MCP servers on developer workstations, step 3-4 is what matters: the developer has already logged in via their cloud CLI, and the SDK picks up the cached token automatically.

### SSO Login Commands

| Cloud | Login Command | Token Location | Expiry |
|-------|--------------|----------------|--------|
| AWS | `aws sso login --profile gdev-docs` | `~/.aws/sso/cache/` | Configurable (up to 12h, auto-refresh) |
| Azure | `az login` | `~/.azure/msal_token_cache.json` | 1-24h depending on tenant config |
| GCP | `gcloud auth application-default login` | `~/.config/gcloud/application_default_credentials.json` | 1h (auto-refresh via refresh token) |

### Is There a Universal Auth Pattern?

**No practical one.** Options considered:

1. **Vault as credential broker**: HashiCorp Vault has secrets engines for AWS, Azure, and GCP that dynamically generate short-lived credentials. This is powerful but adds a significant dependency (running Vault infrastructure). Overkill for gdev's use case.

2. **OIDC token exchange**: Each cloud supports OIDC-based federation, but the configuration is cloud-specific (AWS IAM Identity Center, Azure Entra ID, GCP Workforce Identity Federation). No universal OIDC client covers all three.

3. **Detect-and-dispatch**: Check which cloud CLIs are configured and use the corresponding credential chain. This is the most practical approach.

### Recommended Pattern for gdev

gdev should detect which cloud credentials are available and configure the MCP server accordingly:

```go
// Pseudocode for gdev's credential detection
func detectCloudAuth() CloudAuth {
    if awsProfileExists("gdev-docs") || envSet("AWS_PROFILE") {
        return AWSAuth{Profile: "gdev-docs"}
    }
    if azCLILoggedIn() {
        return AzureAuth{UseDefaultCredential: true}
    }
    if gcloudADCExists() {
        return GCPAuth{UseADC: true}
    }
    if envSet("S3_ENDPOINT") {
        return S3Auth{Endpoint: os.Getenv("S3_ENDPOINT")}
    }
    return LocalAuth{} // No cloud, use local paths
}
```

The MCP server config generated by gdev includes the right environment variables:

```json
{
  "mcpServers": {
    "local-docs": {
      "command": "openzim-mcp",
      "env": {
        "OPENZIM_MCP_ZIM_DIR": "/mnt/gdev-docs/zim",
        "AWS_PROFILE": "gdev-docs"
      }
    }
  }
}
```

For rclone-mounted storage, the auth is handled at mount time (rclone uses the cloud CLI credentials), so the MCP server sees local file paths and needs no cloud credentials itself.

---

## 7. Hosted Service Abstraction

### Container-Based Deployment Per Cloud

All three major clouds offer serverless container platforms. A single Docker image runs kiwix-serve + DevDocs MCP on all of them:

```dockerfile
FROM python:3.12-slim
RUN pip install openzim-mcp
# DevDocs data and ZIM files mounted at /data
EXPOSE 8080
CMD ["openzim-mcp", "--transport", "http", "--port", "8080"]
```

| Cloud | Service | Mount Storage | Auth Gateway | Always-On Cost (0.5 vCPU, 1 GB) |
|-------|---------|--------------|-------------|--------------------------------|
| AWS | ECS Fargate | EFS mount | ALB + Cognito | ~$18/mo (+$30 EFS +$16 ALB) |
| Azure | ACI | Azure Files mount | App Gateway + Entra ID | ~$20/mo (+$5 Azure Files) |
| GCP | Cloud Run | GCS via gcsfuse CSI or Filestore NFS | IAP (free) | ~$35-45/mo |
| Any | Nomad | Any FUSE mount | Traefik + OIDC | Self-hosted only |

### Serverless vs Always-On

| Approach | Pros | Cons | Best For |
|----------|------|------|----------|
| **Always-on** (min-instances=1) | No cold start, consistent perf | Higher cost ($18-45/mo) | Teams using kiwix-serve as primary interface |
| **Scale-to-zero** | Near-zero idle cost | Cold start (5-30s), ZIM file load time | Infrequent access, cost-sensitive |
| **FUSE mount only** (no hosted service) | Zero compute cost, lowest complexity | Every developer mounts independently | Default recommended approach |

**Recommendation**: The FUSE-mount approach (developers mount remote storage locally, MCP servers read local paths) should be the default. A hosted kiwix-serve is optional for teams that want a shared web UI or need to support thin clients that can't run local FUSE mounts.

### Nomad as Cloud-Agnostic Orchestration

HashiCorp Nomad is relevant for gdev because:
- Single binary, no external dependencies (unlike Kubernetes which needs etcd)
- Cloud-agnostic: runs on AWS, GCP, Azure, on-prem, edge
- Docker driver for containers, exec driver for binaries
- Built-in service discovery (Consul integration)
- Proven at scale (10K+ nodes)

However, Nomad adds operational complexity that most gdev adopters won't want. It makes sense only for organizations already running Nomad or wanting a self-hosted orchestration layer. For most teams, the cloud-native serverless option (Fargate / Cloud Run / ACI) is simpler.

---

## 8. Concrete Module Design Recommendation

### Terraform Module Structure

```
gdev/templates/terraform/docs-hosting/
├── common/
│   ├── variables.tf          # Shared variable definitions
│   └── outputs.tf            # Shared output contract
├── aws/
│   ├── main.tf               # S3 bucket, IAM policy, optional Fargate task
│   ├── variables.tf          # extends common + AWS-specific (region, profile)
│   └── outputs.tf            # storage_endpoint, mount_command, mcp_config
├── azure/
│   ├── main.tf               # Blob container, SAS policy, optional ACI
│   ├── variables.tf          # extends common + Azure-specific (resource_group, subscription)
│   └── outputs.tf
├── gcp/
│   ├── main.tf               # GCS bucket, IAM binding, optional Cloud Run
│   ├── variables.tf          # extends common + GCP-specific (project, region)
│   └── outputs.tf
├── s3-compatible/
│   ├── main.tf               # Generic S3 bucket via AWS provider with custom endpoint
│   ├── variables.tf          # extends common + endpoint, access_key, secret_key
│   └── outputs.tf
└── local/
    ├── main.tf               # null_resource, just validates local paths exist
    ├── variables.tf          # local_data_dir
    └── outputs.tf            # Same output contract with local paths
```

### Common Variable Interface

```hcl
# common/variables.tf
variable "project_name" {
  type        = string
  description = "Name prefix for all resources"
  default     = "gdev-docs"
}

variable "storage_size_gb" {
  type        = number
  description = "Total storage for documentation corpora"
  default     = 100
}

variable "enable_hosted_service" {
  type        = bool
  description = "Deploy kiwix-serve/DevDocs as a hosted service"
  default     = false
}

variable "allowed_cidr_blocks" {
  type        = list(string)
  description = "CIDR blocks allowed to access storage (VPN ranges, office IPs)"
  default     = []
}

variable "allowed_users" {
  type        = list(string)
  description = "User identifiers (email, SSO group) allowed access"
  default     = []
}

variable "container_image" {
  type        = string
  description = "Docker image for hosted documentation service"
  default     = "ghcr.io/gdev/docs-server:latest"
}

variable "tags" {
  type        = map(string)
  description = "Resource tags/labels"
  default     = {}
}
```

### Common Output Contract

```hcl
# common/outputs.tf
output "storage_endpoint" {
  description = "Storage endpoint URL (s3://, gs://, https://, or local path)"
}

output "mount_command" {
  description = "Command to mount remote storage locally via rclone/FUSE"
}

output "mcp_config" {
  description = "JSON snippet for .mcp.json configuration"
}

output "hosted_service_url" {
  description = "URL of hosted documentation service (if enabled)"
}

output "estimated_monthly_cost" {
  description = "Estimated monthly cost in USD"
}
```

### How gdev Generates Terraform

```
$ qsdev init --profile enterprise-aws --region us-east-1
  → Detects AWS SSO profile in ~/.aws/config
  → Copies aws/ template to .gdev/terraform/docs-hosting/
  → Generates terraform.tfvars with detected values
  → Runs terraform init && terraform plan
  → Outputs: mount command, MCP config snippet

$ qsdev init --profile enterprise-azure --subscription abc123
  → Detects az login credentials
  → Copies azure/ template
  → Same flow

$ qsdev init --profile self-hosted --s3-endpoint http://minio.internal:9000
  → Copies s3-compatible/ template
  → Configures custom endpoint

$ qsdev init --profile local --data-dir /srv/gdev-docs
  → Copies local/ template
  → No cloud resources, just validates paths
  → Outputs MCP config pointing to local directory
```

---

## 9. On-Premises / Air-Gapped Option

### MinIO on Local Infrastructure

For organizations without cloud access or with strict data residency requirements:

```hcl
# s3-compatible/main.tf with MinIO-specific values
# MinIO can run on any machine with disk space

# terraform.tfvars
s3_endpoint     = "http://minio.internal:9000"
s3_access_key   = "minioadmin"  # From org's MinIO deployment
s3_secret_key   = "minioadmin"
bucket_name     = "gdev-docs"
use_path_style  = true
```

MinIO deployment is out of scope for gdev's Terraform (the org already has it or deploys it separately), but gdev provides the S3-compatible module that targets any MinIO endpoint.

### NFS Share on Existing File Server

The simplest enterprise option. No cloud, no S3, no FUSE:

```yaml
# .gdev/config.yaml
docs_hosting:
  provider: local
  data_dir: /mnt/corp-nfs/gdev-docs    # Already mounted via /etc/fstab
```

MCP servers read directly from the NFS path. The gdev `local/` Terraform module is a no-op that validates the path exists and outputs the MCP config.

### Direct Disk Attachment (Air-Gapped)

For truly air-gapped environments (classified networks, submarines, etc.):

1. ZIM files and DevDocs JSON downloaded on an internet-connected machine
2. Transferred via USB drive / sneakernet to the air-gapped network
3. Placed on a local or shared disk

```yaml
# .gdev/config.yaml
docs_hosting:
  provider: local
  data_dir: /opt/gdev-docs    # Populated by USB transfer
  auto_update: false           # No internet, manual updates only
```

### Unified MCP Configuration

The critical insight: **the MCP server doesn't care where the files come from**. Whether mounted from S3 via rclone, Azure via BlobFuse2, NFS, or a local USB drive, the MCP server sees local file paths:

```json
{
  "mcpServers": {
    "local-docs": {
      "command": "openzim-mcp",
      "env": {
        "OPENZIM_MCP_ZIM_DIR": "/mnt/gdev-docs/zim"
      }
    },
    "devdocs": {
      "command": "devdocs-mcp",
      "env": {
        "DEVDOCS_DATA_DIR": "/mnt/gdev-docs/devdocs"
      }
    }
  }
}
```

This is the same config regardless of provider. The only thing that changes is how `/mnt/gdev-docs` is populated — cloud FUSE mount, NFS, or local disk.

---

## Cross-Cloud Cost Comparison Summary

### FUSE-Mount Approach (Storage Only, ~100 GB)

| Provider | Storage/mo | Egress | Total/mo | Notes |
|----------|-----------|--------|----------|-------|
| Hetzner Object Storage | ~$0.52 | Free (internal) | **~$0.52** | Cheapest, EU only |
| DigitalOcean Spaces | $5.00 (flat) | Included 1 TB | **$5.00** | Simple pricing |
| GCP Standard | $2.00 | $0.08-0.12/GB | **~$2.02** | No egress for FUSE in same region |
| AWS S3 Standard | $2.30 | $0.09/GB | **~$2.32** | No egress for FUSE in same region |
| Azure Blob Hot | $1.80 | $0.087/GB | **~$1.82** | Cheapest major cloud |
| Oracle OCI | $2.55 | $0.0085/GB | **~$2.56** | Cheap egress |
| MinIO (self-hosted) | Hardware only | $0 | **$0** | Org provides infrastructure |

### Hosted Service Approach (Storage + Compute + Auth, Always-On)

| Provider | Total/mo | Breakdown |
|----------|----------|-----------|
| AWS (Fargate + EFS + ALB) | ~$65 | $18 compute + $30 EFS + $16 ALB + $2 S3 |
| Azure (ACI + Files) | ~$25 | $20 ACI + $5 Azure Files |
| GCP (Cloud Run + GCS + IAP) | ~$42 | $40 Cloud Run + $2 GCS + free IAP |

---

## Architectural Recommendations

### Default: FUSE Mount with rclone

For most gdev adopters, the recommended architecture is:

1. **Storage**: S3-compatible object storage (any provider)
2. **Mount**: rclone with `--vfs-cache-mode full` and local SSD cache
3. **Auth**: Cloud CLI SSO (detected by gdev)
4. **MCP servers**: Read from local mount path
5. **Terraform**: Per-provider module generated by gdev from profile

This gives zero compute cost, universal provider support, and the MCP servers don't need to know anything about cloud APIs.

### Enterprise: Hosted Service Add-On

For teams wanting a shared web UI or supporting thin clients:

1. Everything above, plus:
2. **Compute**: Cloud-native serverless container (Fargate / Cloud Run / ACI)
3. **Auth gateway**: Cloud-native SSO integration (Cognito / IAP / Entra ID)
4. **Container**: Single Docker image with kiwix-serve + DevDocs

### Air-Gapped: Local Paths

1. **Storage**: Local disk or NFS share
2. **Mount**: None needed (direct paths)
3. **Auth**: None (local access)
4. **MCP servers**: Same config, different paths
5. **Terraform**: No-op `local/` module

---

## Sources

All raw sources saved to `docs/`:
- `mountpoint-s3-vs-fusion-seqera.md` — Mountpoint for S3 performance
- `s3-mount-tools-benchmark-comparison.md` — s3fs vs Mountpoint vs Goofys benchmarks
- `s3-files-vs-mountpoint-vs-s3fs.md` — S3 Files comparison with benchmarks
- `aws-fargate-pricing-2026.md` — ECS Fargate pricing
- `aws-iam-identity-center-sso-credentials.md` — AWS SSO credential chain
- `gcs-s3-interoperability.md` — GCS S3-compatible XML API
- `gcp-cloud-storage-pricing-2026.md` — GCP storage pricing
- `cross-cloud-storage-pricing-comparison-2026.md` — AWS vs Azure vs GCP vs OCI pricing
- `rclone-mount-technical-details.md` — rclone FUSE mount capabilities
- `terraform-multi-cloud-module-patterns.md` — Multi-cloud Terraform patterns
- `terraform-module-composition-hashicorp.md` — HashiCorp official composition guidance
- `hetzner-storage-box-pricing.md` — Hetzner Storage Box details
- `hetzner-object-storage-pricing.md` — Hetzner S3-compatible storage

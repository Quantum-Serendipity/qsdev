package security

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	"golang.org/x/oauth2/google"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
)

const (
	// defaultCredentialTTL is the requested lifetime when the caller omits ttl.
	defaultCredentialTTL = time.Hour
	// minAWSCredentialTTL / maxAWSCredentialTTL bound the AWS STS duration to the
	// service's accepted range (15 minutes .. 12 hours).
	minAWSCredentialTTL = 15 * time.Minute
	maxAWSCredentialTTL = 12 * time.Hour
	// maxGCPCredentialTTL is the IAM Credentials generateAccessToken ceiling.
	maxGCPCredentialTTL = time.Hour
	// credentialProbeTimeout bounds the whole vend operation so an unreachable
	// metadata endpoint (a non-cloud host) cannot stall the tool. The provider
	// network calls run within this deadline and a timeout degrades to
	// not_configured rather than hanging.
	credentialProbeTimeout = 8 * time.Second
)

// awsScope / gcpScope / azureScope are the default audiences requested when the
// caller does not override them.
const (
	gcpCloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"
	azureManagementScope  = "https://management.azure.com/.default"
)

// credentialVendor vends short-lived cloud credentials. It holds no long-lived
// secrets: every provider path resolves ambient credentials (environment,
// profile, workload identity, or instance metadata) and exchanges them for a
// time-boxed token, which is the only secret returned.
type credentialVendor struct{}

func newCredentialVendor() *credentialVendor { return &credentialVendor{} }

// handle dispatches to the requested provider. A missing or unknown provider, or
// a provider whose ambient credentials are absent, degrades to not_configured
// rather than crashing.
func (cv *credentialVendor) handle(ctx context.Context, _ *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	provider := strings.ToLower(strings.TrimSpace(toolutil.StringArgOr(req.Arguments, "provider", "")))
	ttl := parseTTL(req.Arguments)

	ctx, cancel := context.WithTimeout(ctx, credentialProbeTimeout)
	defer cancel()

	switch provider {
	case "aws":
		return cv.vendAWS(ctx, req.Arguments, ttl)
	case "gcp":
		return cv.vendGCP(ctx, req.Arguments, ttl)
	case "azure":
		return cv.vendAzure(ctx, req.Arguments)
	case "":
		return toolutil.NotConfigured("provider is required",
			map[string]any{"allowed": []string{"aws", "gcp", "azure"}}), nil
	default:
		return toolutil.NotConfigured("unknown provider",
			map[string]any{"got": provider, "allowed": []string{"aws", "gcp", "azure"}}), nil
	}
}

// parseTTL reads the optional ttl argument, accepting either a Go duration string
// (e.g. "1h", "30m") or a number of seconds, and defaults to one hour.
func parseTTL(args map[string]any) time.Duration {
	if s, ok := toolutil.StringArg(args, "ttl"); ok && s != "" {
		if d, err := time.ParseDuration(s); err == nil && d > 0 {
			return d
		}
	}
	if secs, ok := toolutil.IntArg(args, "ttl"); ok && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return defaultCredentialTTL
}

// vendAWS exchanges ambient AWS credentials for a temporary STS session. With a
// role_arn it calls AssumeRole; otherwise it calls GetSessionToken. The absence
// of any resolvable credential is reported as not_configured.
func (cv *credentialVendor) vendAWS(ctx context.Context, args map[string]any, ttl time.Duration) (*spi.ToolResult, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return toolutil.NotConfigured("AWS configuration could not be loaded",
			map[string]any{"error": err.Error()}), nil
	}
	if _, err := cfg.Credentials.Retrieve(ctx); err != nil {
		return toolutil.NotConfigured("no AWS credentials available (set AWS_ACCESS_KEY_ID/profile/role or run on an instance with an IAM role)",
			map[string]any{"error": err.Error()}), nil
	}

	if ttl > maxAWSCredentialTTL {
		ttl = maxAWSCredentialTTL
	}
	if ttl < minAWSCredentialTTL {
		ttl = minAWSCredentialTTL
	}
	durSecs := int32(ttl.Seconds())
	client := sts.NewFromConfig(cfg)

	var creds *ststypes.Credentials
	roleARN, hasRole := toolutil.StringArg(args, "role_arn")
	if hasRole && roleARN != "" {
		out, err := client.AssumeRole(ctx, &sts.AssumeRoleInput{
			RoleArn:         awssdk.String(roleARN),
			RoleSessionName: awssdk.String(fmt.Sprintf("qsdev-mcp-%d", time.Now().Unix())),
			DurationSeconds: awssdk.Int32(durSecs),
		})
		if err != nil {
			return toolutil.ErrorResult("AWS STS AssumeRole failed",
				map[string]any{"role_arn": roleARN, "error": err.Error()}), nil
		}
		creds = out.Credentials
	} else {
		out, err := client.GetSessionToken(ctx, &sts.GetSessionTokenInput{
			DurationSeconds: awssdk.Int32(durSecs),
		})
		if err != nil {
			return toolutil.ErrorResult("AWS STS GetSessionToken failed",
				map[string]any{"error": err.Error()}), nil
		}
		creds = out.Credentials
	}

	expiry := ""
	if creds.Expiration != nil {
		expiry = creds.Expiration.UTC().Format(time.RFC3339)
	}
	structured := map[string]any{
		"provider":          "aws",
		"access_key_id":     awssdk.ToString(creds.AccessKeyId),
		"secret_access_key": awssdk.ToString(creds.SecretAccessKey),
		"session_token":     awssdk.ToString(creds.SessionToken),
		"expiration":        expiry,
	}
	return toolutil.Result("aws: vended temporary STS credentials (expires "+expiry+")", structured), nil
}

// vendGCP impersonates a service account via the IAM Credentials REST API,
// returning a short-lived OAuth2 access token. It calls the REST endpoint
// directly over an Application-Default-Credentials-authorized net/http client
// (golang.org/x/oauth2/google) rather than the google.golang.org/api SDK,
// which unconditionally pulls in the gRPC, genproto, and OpenTelemetry stacks
// (~100 vendored packages) for what is a single REST call. ADC must be present;
// its absence is reported as not_configured.
func (cv *credentialVendor) vendGCP(ctx context.Context, args map[string]any, ttl time.Duration) (*spi.ToolResult, error) {
	serviceAccount, ok := toolutil.StringArg(args, "service_account")
	if !ok || serviceAccount == "" {
		return toolutil.NotConfigured("GCP credential vending requires a service_account email to impersonate",
			map[string]any{"remediation": "pass service_account=<sa>@<project>.iam.gserviceaccount.com"}), nil
	}

	// DefaultClient resolves Application Default Credentials (env key file,
	// gcloud ADC, or instance metadata) and returns an *http.Client that attaches
	// and refreshes the bearer token. No gRPC.
	client, err := google.DefaultClient(ctx, gcpCloudPlatformScope)
	if err != nil {
		return toolutil.NotConfigured("no GCP application default credentials available",
			map[string]any{
				"error":       err.Error(),
				"remediation": "configure GOOGLE_APPLICATION_CREDENTIALS or run `gcloud auth application-default login`",
			}), nil
	}

	if ttl > maxGCPCredentialTTL {
		ttl = maxGCPCredentialTTL
	}
	name := "projects/-/serviceAccounts/" + serviceAccount
	endpoint := "https://iamcredentials.googleapis.com/v1/" + name + ":generateAccessToken"
	reqBody, err := json.Marshal(map[string]any{
		"scope":    []string{gcpCloudPlatformScope},
		"lifetime": fmt.Sprintf("%ds", int(ttl.Seconds())),
	})
	if err != nil {
		return toolutil.ErrorResult("encoding GCP generateAccessToken request",
			map[string]any{"error": err.Error()}), nil
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return toolutil.ErrorResult("building GCP generateAccessToken request",
			map[string]any{"error": err.Error()}), nil
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return toolutil.ErrorResult("GCP IAM generateAccessToken failed",
			map[string]any{"service_account": serviceAccount, "error": err.Error()}), nil
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return toolutil.ErrorResult("GCP IAM generateAccessToken returned a non-OK status",
			map[string]any{"service_account": serviceAccount, "status": resp.StatusCode, "body": string(body)}), nil
	}

	var out struct {
		AccessToken string `json:"accessToken"`
		ExpireTime  string `json:"expireTime"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return toolutil.ErrorResult("decoding GCP generateAccessToken response",
			map[string]any{"error": err.Error()}), nil
	}

	structured := map[string]any{
		"provider":        "gcp",
		"service_account": serviceAccount,
		"access_token":    out.AccessToken,
		"expiration":      out.ExpireTime,
		"scope":           gcpCloudPlatformScope,
	}
	return toolutil.Result("gcp: vended impersonated access token for "+serviceAccount, structured), nil
}

// vendAzure requests an access token from the instance's Managed Identity. The
// optional identity argument selects a user-assigned identity by client id. The
// token lifetime is fixed by the platform; the absence of a managed identity
// (not running on Azure) is reported as not_configured.
func (cv *credentialVendor) vendAzure(ctx context.Context, args map[string]any) (*spi.ToolResult, error) {
	opts := &azidentity.ManagedIdentityCredentialOptions{}
	if id, ok := toolutil.StringArg(args, "identity"); ok && id != "" {
		opts.ID = azidentity.ClientID(id)
	}
	cred, err := azidentity.NewManagedIdentityCredential(opts)
	if err != nil {
		return toolutil.NotConfigured("Azure managed identity credential could not be constructed",
			map[string]any{"error": err.Error()}), nil
	}

	scope := toolutil.StringArgOr(args, "scope", azureManagementScope)
	tok, err := cred.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{scope}})
	if err != nil {
		return toolutil.NotConfigured("no Azure managed identity available (not running on Azure or MI disabled)",
			map[string]any{
				"error":       err.Error(),
				"remediation": "run on an Azure resource with a system- or user-assigned managed identity",
			}), nil
	}

	structured := map[string]any{
		"provider":     "azure",
		"access_token": tok.Token,
		"expiration":   tok.ExpiresOn.UTC().Format(time.RFC3339),
		"scope":        scope,
	}
	return toolutil.Result("azure: vended managed-identity access token", structured), nil
}

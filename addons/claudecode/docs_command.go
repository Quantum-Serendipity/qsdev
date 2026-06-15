package claudecode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/contentsign"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
)

func docsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Manage local documentation corpus",
		Long: `Download, track, and manage local documentation sets for offline use.

Documentation sets include DevDocs API references and Stack Exchange ZIM
archives. Use subcommands to download, check status, update, or clean
the local corpus.`,
	}

	cmd.AddCommand(docsDownloadCmd())
	cmd.AddCommand(docsStatusCmd())
	cmd.AddCommand(docsVerifyCmd())
	cmd.AddCommand(docsOutdatedCmd())
	cmd.AddCommand(docsUpdateCmd())
	cmd.AddCommand(docsCleanCmd())
	cmd.AddCommand(docsEnableCmd())
	cmd.AddCommand(docsDisableCmd())

	return cmd
}

// errCorpusVerifyFailed is returned by `docs verify` when any documentation set
// fails verification, so the process exits non-zero for CI gating.
var errCorpusVerifyFailed = errors.New("documentation corpus verification failed")

// docVerifyResult is the per-doc-set outcome of `qsdev docs verify`.
type docVerifyResult struct {
	Slug     string `json:"slug"`
	Type     string `json:"type"`
	Status   string `json:"status"` // signed-verified | hash-verified | failed
	Verified bool   `json:"verified"`
	KeyID    string `json:"key_id,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

func docsVerifyCmd() *cobra.Command {
	var (
		jsonOutput     bool
		keysDir        string
		requireTrusted bool
	)

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify the integrity (and signatures, if present) of the documentation corpus",
		Long: `Verify each installed documentation set against the manifest. When a set's
files carry detached Minisign signatures (<file>.minisig), those signatures are
verified against trusted keys; otherwise the set's recorded combined SHA-256 is
recomputed and compared. Exits non-zero when any set fails, for CI gating.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := mcpregistry.NewDocsCorpusManager(
				mcpregistry.DefaultDocsDataDir(),
				http.DefaultClient,
			)
			manifest, err := mgr.LoadManifest()
			if err != nil {
				return err
			}
			return runDocsVerify(cmd, mgr, manifest, keysDir, requireTrusted, jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().StringVar(&keysDir, "keys", "", "Directory of trusted public keys (default: ~/.qsdev/keys)")
	cmd.Flags().BoolVar(&requireTrusted, "require-trusted", false, "Require a verified signature; treat hash-only or failed sets as failures")

	return cmd
}

// runDocsVerify verifies every doc set in stable order, prints the results, and
// returns a non-nil error when any set failed (or, under requireTrusted, was not
// signed-verified).
func runDocsVerify(cmd *cobra.Command, mgr *mcpregistry.DocsCorpusManager, manifest *mcpregistry.DocsManifest, keysDir string, requireTrusted, jsonOutput bool) error {
	if len(manifest.DocSets) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No documentation sets installed.")
		return nil
	}

	keys := make([]string, 0, len(manifest.DocSets))
	for k := range manifest.DocSets {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	results := make([]docVerifyResult, 0, len(keys))
	for _, k := range keys {
		results = append(results, verifyDocSet(cmd.Context(), mgr, manifest.DocSets[k], keysDir, requireTrusted))
	}

	if err := printVerifyResults(cmd, results, jsonOutput); err != nil {
		return err
	}

	failed := 0
	for _, r := range results {
		if !r.Verified || (requireTrusted && r.Status != contentsign.StatusSignedVerified) {
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%w: %d of %d set(s) failed", errCorpusVerifyFailed, failed, len(results))
	}
	return nil
}

// verifyDocSet verifies a single doc set: when every file has a .minisig sidecar
// it verifies the signatures against trusted keys; otherwise it falls back to
// the recorded combined-hash check.
func verifyDocSet(ctx context.Context, mgr *mcpregistry.DocsCorpusManager, entry *mcpregistry.DocSetEntry, keysDir string, requireTrusted bool) docVerifyResult {
	res := docVerifyResult{Slug: entry.Slug, Type: entry.Type.String()}
	if len(entry.Files) > 0 && allSigned(entry.Files) {
		return verifyDocSetSignatures(ctx, entry, keysDir, requireTrusted, res)
	}

	ok, _, err := mgr.VerifyHash(entry)
	switch {
	case err != nil:
		res.Status, res.Reason = contentsign.StatusFailed, err.Error()
	case ok:
		res.Status, res.Verified = contentsign.StatusHashVerified, true
	default:
		res.Status, res.Reason = contentsign.StatusFailed, "hash mismatch or missing file"
	}
	return res
}

// verifyDocSetSignatures verifies each file's detached signature; the set is
// signed-verified only when every file verifies, recording the first failure
// reason otherwise and the verifying key ID when it is uniform across files.
func verifyDocSetSignatures(ctx context.Context, entry *mcpregistry.DocSetEntry, keysDir string, requireTrusted bool, res docVerifyResult) docVerifyResult {
	keys, err := contentsign.LoadTrustedKeys(keysDir)
	if err != nil {
		res.Status, res.Reason = contentsign.StatusFailed, fmt.Sprintf("loading trusted keys: %v", err)
		return res
	}
	opts := contentsign.VerifyOptions{TrustedKeys: keys, RequireTrusted: requireTrusted}

	keyID := ""
	for i, f := range entry.Files {
		vr, verr := contentsign.Verify(ctx, f, opts)
		if verr != nil {
			res.Status, res.Reason = contentsign.StatusFailed, fmt.Sprintf("verifying %s: %v", f, verr)
			return res
		}
		if !vr.Verified {
			res.Status, res.Reason = contentsign.StatusFailed, fmt.Sprintf("%s: %s", f, vr.Reason)
			return res
		}
		switch {
		case i == 0:
			keyID = string(vr.KeyID)
		case string(vr.KeyID) != keyID:
			keyID = ""
		}
	}
	res.Status, res.Verified, res.KeyID = contentsign.StatusSignedVerified, true, keyID
	return res
}

// allSigned reports whether every path has a <path>.minisig sidecar on disk.
func allSigned(files []string) bool {
	for _, f := range files {
		if _, err := os.Stat(f + ".minisig"); err != nil {
			return false
		}
	}
	return true
}

// printVerifyResults writes the verification results as indented JSON when
// jsonOutput is set, otherwise as one human line per set plus a summary count.
func printVerifyResults(cmd *cobra.Command, results []docVerifyResult, jsonOutput bool) error {
	w := cmd.OutOrStdout()
	if jsonOutput {
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling verify results: %w", err)
		}
		fmt.Fprintln(w, string(data))
		return nil
	}

	fmt.Fprintf(w, "Documentation Corpus Verification (%d)\n", len(results))
	fmt.Fprintln(w, "----------------------------------------")
	verified := 0
	for _, r := range results {
		detail := r.Reason
		if r.KeyID != "" {
			detail = "key=" + r.KeyID
		}
		fmt.Fprintf(w, "  %-30s  %-8s  %-15s  %s\n", r.Slug, r.Type, r.Status, detail)
		if r.Verified {
			verified++
		}
	}
	fmt.Fprintf(w, "\n%d of %d set(s) verified.\n", verified, len(results))
	return nil
}

func docsDownloadCmd() *cobra.Command {
	var (
		zimOnly     bool
		devdocsOnly bool
	)

	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download configured documentation sets",
		Long: `Download documentation sets for offline use. By default downloads both
DevDocs API references and ZIM archives. Use --zim or --devdocs to
download only one type.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := mcpregistry.NewDocsCorpusManager(
				mcpregistry.DefaultDocsDataDir(),
				http.DefaultClient,
			)
			mgr.Ingest = func(ctx context.Context, dir string) error {
				_, err := contentsign.IngestDevDocs(ctx, dir, contentsign.DefaultSanitizeOptions())
				return err
			}
			ctx := cmd.Context()

			cat, err := catalog.Default()
			if err != nil {
				return fmt.Errorf("loading catalog: %w", err)
			}

			downloadZIM := !devdocsOnly
			downloadDevDocs := !zimOnly

			if downloadDevDocs {
				baseURL := cat.DevDocsBaseURL()
				allSlugs := cat.DevDocsSlugs()
				if len(allSlugs) == 0 {
					allSlugs = mcpregistry.LanguageToDevDocsSlugs
				}
				projectEcosystems := projectEcosystemSet()
				fmt.Fprintln(cmd.OutOrStdout(), "Downloading DevDocs documentation sets...")
				for lang, langSlugs := range allSlugs {
					if len(projectEcosystems) > 0 && !projectEcosystems[lang] {
						continue
					}
					for _, slug := range langSlugs {
						fmt.Fprintf(cmd.OutOrStdout(), "  %s (%s)...", slug, lang)
						if err := mgr.DownloadDevDocs(ctx, slug, baseURL); err != nil {
							fmt.Fprintf(cmd.OutOrStdout(), " FAILED: %s\n", err)
							continue
						}
						fmt.Fprintln(cmd.OutOrStdout(), " OK")
					}
				}
			}

			if downloadZIM {
				fmt.Fprintln(cmd.OutOrStdout(), "Downloading ZIM archives...")
				if err := downloadZIMEntries(ctx, cmd, mgr, cat); err != nil {
					return err
				}
			}

			fmt.Fprintln(cmd.OutOrStdout(), "\nDownload complete. Run 'qsdev docs status' to see installed sets.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&zimOnly, "zim", false, "Download only ZIM archives")
	cmd.Flags().BoolVar(&devdocsOnly, "devdocs", false, "Download only DevDocs sets")

	return cmd
}

func downloadZIMEntries(ctx context.Context, cmd *cobra.Command, mgr *mcpregistry.DocsCorpusManager, cat *catalog.Catalog) error {
	entries := catalogZIMEntries(cat)
	for _, entry := range entries {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s...", entry.DisplayName)
		if err := mgr.DownloadZIM(ctx, entry); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), " FAILED: %s\n", err)
			continue
		}
		fmt.Fprintln(cmd.OutOrStdout(), " OK")
	}
	return nil
}

func docsStatusCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show installed vs configured documentation sets",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := mcpregistry.NewDocsCorpusManager(
				mcpregistry.DefaultDocsDataDir(),
				http.DefaultClient,
			)

			manifest, err := mgr.LoadManifest()
			if err != nil {
				return err
			}

			if jsonOutput {
				data, err := json.MarshalIndent(manifest, "", "  ")
				if err != nil {
					return fmt.Errorf("marshaling manifest: %w", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			if len(manifest.DocSets) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No documentation sets installed.")
				fmt.Fprintln(cmd.OutOrStdout(), "Run 'qsdev docs download' to get started.")
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Installed Documentation Sets (%d)\n", len(manifest.DocSets))
			fmt.Fprintln(cmd.OutOrStdout(), "----------------------------------------")

			keys := make([]string, 0, len(manifest.DocSets))
			for k := range manifest.DocSets {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, key := range keys {
				entry := manifest.DocSets[key]
				sizeMB := float64(entry.SizeBytes) / (1024 * 1024)
				fmt.Fprintf(cmd.OutOrStdout(), "  %-40s  %s  %.1f MB  %s\n",
					key, entry.Type, sizeMB, entry.InstalledAt.Format("2006-01-02"))
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}

func docsOutdatedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "outdated",
		Short: "Check for newer documentation versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := mcpregistry.NewDocsCorpusManager(
				mcpregistry.DefaultDocsDataDir(),
				http.DefaultClient,
			)

			cat, err := catalog.Default()
			if err != nil {
				return fmt.Errorf("loading catalog: %w", err)
			}

			outdated, err := mgr.CheckOutdated(catalogZIMEntries(cat))
			if err != nil {
				return err
			}

			if len(outdated) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "All documentation sets are up to date.")
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Outdated Documentation Sets (%d)\n", len(outdated))
			fmt.Fprintln(cmd.OutOrStdout(), "----------------------------------------")
			for _, o := range outdated {
				fmt.Fprintf(cmd.OutOrStdout(), "  %-30s  %s -> %s\n",
					o.Slug, o.InstalledVersion, o.AvailableVersion)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "\nRun 'qsdev docs update' to download newer versions.")

			return nil
		},
	}

	return cmd
}

func docsUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Download newer versions of outdated documentation",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := mcpregistry.NewDocsCorpusManager(
				mcpregistry.DefaultDocsDataDir(),
				http.DefaultClient,
			)
			ctx := cmd.Context()

			cat, err := catalog.Default()
			if err != nil {
				return fmt.Errorf("loading catalog: %w", err)
			}

			zimEntries := catalogZIMEntries(cat)
			outdated, err := mgr.CheckOutdated(zimEntries)
			if err != nil {
				return err
			}

			if len(outdated) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "All documentation sets are up to date.")
				return nil
			}

			baseURL := cat.DevDocsBaseURL()
			fmt.Fprintf(cmd.OutOrStdout(), "Updating %d documentation set(s)...\n", len(outdated))

			for _, o := range outdated {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s...", o.Slug)
				switch o.Type {
				case mcpregistry.DocSetZIM:
					for _, entry := range zimEntries {
						if entry.Slug == o.AvailableVersion {
							if err := mgr.DownloadZIM(ctx, entry); err != nil {
								fmt.Fprintf(cmd.OutOrStdout(), " FAILED: %s\n", err)
								continue
							}
							fmt.Fprintln(cmd.OutOrStdout(), " OK")
							break
						}
					}
				case mcpregistry.DocSetDevDocs:
					if err := mgr.DownloadDevDocs(ctx, o.Slug, baseURL); err != nil {
						fmt.Fprintf(cmd.OutOrStdout(), " FAILED: %s\n", err)
						continue
					}
					fmt.Fprintln(cmd.OutOrStdout(), " OK")
				}
			}

			return nil
		},
	}

	return cmd
}

func projectEcosystemSet() map[string]bool {
	val := os.Getenv("QSDEV_ECOSYSTEMS")
	if val == "" {
		return nil
	}
	set := make(map[string]bool)
	for _, eco := range strings.Split(val, ",") {
		eco = strings.TrimSpace(eco)
		if eco != "" {
			set[eco] = true
		}
	}
	return set
}

func catalogZIMEntries(cat *catalog.Catalog) []mcpregistry.ZIMEntry {
	defs := cat.ZIMArchives()
	if len(defs) == 0 {
		return mcpregistry.BuiltinZIMCatalog
	}
	entries := make([]mcpregistry.ZIMEntry, len(defs))
	for i, d := range defs {
		entries[i] = mcpregistry.ZIMEntry{
			Slug:        d.Slug,
			DisplayName: d.DisplayName,
			URL:         d.URL,
			SizeBytes:   d.SizeBytes,
			Ecosystems:  d.Ecosystems,
		}
	}
	return entries
}

func docsCleanCmd() *cobra.Command {
	var (
		zimOnly     bool
		devdocsOnly bool
		all         bool
	)

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove downloaded documentation data",
		Long: `Remove downloaded documentation files and update the manifest. Use flags
to target specific types, or --all to remove everything.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !all && !zimOnly && !devdocsOnly {
				return fmt.Errorf("specify --zim, --devdocs, or --all")
			}

			mgr := mcpregistry.NewDocsCorpusManager(
				mcpregistry.DefaultDocsDataDir(),
				http.DefaultClient,
			)

			opts := mcpregistry.CleanOptions{
				ZIMOnly:     zimOnly,
				DevDocsOnly: devdocsOnly,
				All:         all,
			}

			if err := mgr.Clean(opts); err != nil {
				return err
			}

			switch {
			case all:
				fmt.Fprintln(cmd.OutOrStdout(), "Removed all documentation data.")
			case zimOnly:
				fmt.Fprintln(cmd.OutOrStdout(), "Removed ZIM archives.")
			case devdocsOnly:
				fmt.Fprintln(cmd.OutOrStdout(), "Removed DevDocs documentation.")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&zimOnly, "zim", false, "Remove only ZIM archives")
	cmd.Flags().BoolVar(&devdocsOnly, "devdocs", false, "Remove only DevDocs documentation")
	cmd.Flags().BoolVar(&all, "all", false, "Remove all documentation data")

	return cmd
}

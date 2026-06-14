package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
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
	cmd.AddCommand(docsOutdatedCmd())
	cmd.AddCommand(docsUpdateCmd())
	cmd.AddCommand(docsCleanCmd())
	cmd.AddCommand(docsEnableCmd())
	cmd.AddCommand(docsDisableCmd())

	return cmd
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

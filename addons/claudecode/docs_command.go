package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/spf13/cobra"

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

			downloadZIM := !devdocsOnly
			downloadDevDocs := !zimOnly

			if downloadDevDocs {
				fmt.Fprintln(cmd.OutOrStdout(), "Downloading DevDocs documentation sets...")
				for lang, slugs := range mcpregistry.LanguageToDevDocsSlugs {
					for _, slug := range slugs {
						fmt.Fprintf(cmd.OutOrStdout(), "  %s (%s)...", slug, lang)
						if err := mgr.DownloadDevDocs(ctx, slug); err != nil {
							fmt.Fprintf(cmd.OutOrStdout(), " FAILED: %s\n", err)
							continue
						}
						fmt.Fprintln(cmd.OutOrStdout(), " OK")
					}
				}
			}

			if downloadZIM {
				fmt.Fprintln(cmd.OutOrStdout(), "Downloading ZIM archives...")
				if err := downloadZIMEntries(ctx, cmd, mgr); err != nil {
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

func downloadZIMEntries(ctx context.Context, cmd *cobra.Command, mgr *mcpregistry.DocsCorpusManager) error {
	for _, entry := range mcpregistry.BuiltinZIMCatalog {
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

			outdated, err := mgr.CheckOutdated()
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

			outdated, err := mgr.CheckOutdated()
			if err != nil {
				return err
			}

			if len(outdated) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "All documentation sets are up to date.")
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Updating %d documentation set(s)...\n", len(outdated))

			for _, o := range outdated {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s...", o.Slug)
				switch o.Type {
				case mcpregistry.DocSetZIM:
					// Find the matching catalog entry to get the full URL.
					for _, entry := range mcpregistry.BuiltinZIMCatalog {
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
					if err := mgr.DownloadDevDocs(ctx, o.Slug); err != nil {
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

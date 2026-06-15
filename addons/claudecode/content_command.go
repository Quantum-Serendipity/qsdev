package claudecode

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/contentsign"
)

// errVerificationFailed is returned by `content verify` when the content is not
// verified, so the process exits non-zero for CI gating.
var errVerificationFailed = errors.New("content verification failed")

func contentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "content",
		Short: "Sign and verify local content with Minisign",
		Long: `Sign, verify, and manage signing keys for local content.

Content signing protects the documentation corpus and other artifacts against
tampering. Signatures are detached Minisign signatures (<file>.minisig) verified
against a set of trusted public keys.`,
	}

	cmd.AddCommand(contentSignCmd())
	cmd.AddCommand(contentVerifyCmd())
	cmd.AddCommand(contentKeygenCmd())
	cmd.AddCommand(contentKeysCmd())

	return cmd
}

func contentSignCmd() *cobra.Command {
	var (
		keyPath  string
		comment  string
		password string
		force    bool
	)

	cmd := &cobra.Command{
		Use:   "sign <path>",
		Short: "Sign a content file, producing a detached Minisign signature",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := contentsign.SignOptions{
				KeyPath:        keyPath,
				Password:       password,
				TrustedComment: comment,
				Force:          force,
			}
			sigPath, err := contentsign.Sign(cmd.Context(), args[0], opts)
			if err != nil {
				return fmt.Errorf("signing %q: %w", args[0], err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), sigPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&keyPath, "key", "", "Path to the Minisign secret key (required)")
	cmd.Flags().StringVar(&comment, "comment", "", "Trusted (authenticated) comment to embed in the signature")
	cmd.Flags().StringVar(&password, "password", "", "Password for an encrypted secret key")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite an existing signature file")
	_ = cmd.MarkFlagRequired("key")

	return cmd
}

func contentVerifyCmd() *cobra.Command {
	var (
		keysDir        string
		requireTrusted bool
		jsonOutput     bool
	)

	cmd := &cobra.Command{
		Use:   "verify <path>",
		Short: "Verify a content file against trusted keys",
		Long: `Verify a content file's detached Minisign signature against the trusted key
set. Exits non-zero when the content is not verified, for use in CI gating.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			keys, err := contentsign.LoadTrustedKeys(keysDir)
			if err != nil {
				return fmt.Errorf("loading trusted keys: %w", err)
			}
			opts := contentsign.VerifyOptions{TrustedKeys: keys, RequireTrusted: requireTrusted}
			res, err := contentsign.Verify(cmd.Context(), args[0], opts)
			if err != nil {
				return fmt.Errorf("verifying %q: %w", args[0], err)
			}
			if err := printVerification(cmd, res, jsonOutput); err != nil {
				return err
			}
			if !res.Verified {
				return fmt.Errorf("%w: %s (%s)", errVerificationFailed, res.Path, res.Status)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&keysDir, "keys", "", "Directory of trusted public keys (default: ~/.qsdev/keys)")
	cmd.Flags().BoolVar(&requireTrusted, "require-trusted", false, "Treat unsigned or untrusted content as failure")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output the verification result as JSON")

	return cmd
}

// printVerification writes the verification result to the command's output, as
// indented JSON when jsonOutput is set, otherwise as a single human line.
func printVerification(cmd *cobra.Command, res contentsign.VerificationResult, jsonOutput bool) error {
	if jsonOutput {
		data, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling verification result: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s  status=%s  verified=%t  key=%s\n",
		res.Path, res.Status, res.Verified, res.KeyID)
	return nil
}

func contentKeygenCmd() *cobra.Command {
	var (
		out      string
		password string
	)

	cmd := &cobra.Command{
		Use:   "keygen",
		Short: "Generate a new Minisign key pair",
		Long: `Generate a new Minisign key pair, writing <out>.pub and <out>.key. The secret
key is encrypted when --password is supplied.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pubPath := out + ".pub"
			secPath := out + ".key"
			pub, err := contentsign.GenerateKeyPair(pubPath, secPath, password)
			if err != nil {
				return fmt.Errorf("generating key pair: %w", err)
			}
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Public key:  %s\n", pub.String())
			fmt.Fprintf(w, "Public key:  written to %s\n", pubPath)
			fmt.Fprintf(w, "Secret key:  written to %s\n", secPath)
			fmt.Fprintf(w, "\nWARNING: protect the secret key (%s). Never commit it to version control\n", secPath)
			fmt.Fprintln(w, "or share it. Only the public key (.pub) may be distributed and committed.")
			return nil
		},
	}

	cmd.Flags().StringVar(&out, "out", "qsdev", "Output basename; writes <out>.pub and <out>.key")
	cmd.Flags().StringVar(&password, "password", "", "Password to encrypt the secret key (recommended)")

	return cmd
}

func contentKeysCmd() *cobra.Command {
	var (
		keysDir    string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "keys",
		Short: "List trusted public keys",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			keys, err := contentsign.LoadTrustedKeys(keysDir)
			if err != nil {
				return fmt.Errorf("loading trusted keys: %w", err)
			}
			if jsonOutput {
				return printKeysJSON(cmd, keys)
			}
			w := cmd.OutOrStdout()
			if len(keys) == 0 {
				fmt.Fprintln(w, "No trusted keys found.")
				return nil
			}
			fmt.Fprintf(w, "Trusted Keys (%d)\n", len(keys))
			for _, k := range keys {
				fmt.Fprintf(w, "  %s  %s\n", k.ID(), k.String())
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&keysDir, "keys", "", "Directory of trusted public keys (default: ~/.qsdev/keys)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output the key list as JSON")

	return cmd
}

// printKeysJSON writes the trusted keys as a JSON array of {id, key} objects.
func printKeysJSON(cmd *cobra.Command, keys []contentsign.PublicKey) error {
	type keyView struct {
		ID  string `json:"id"`
		Key string `json:"key"`
	}
	views := make([]keyView, len(keys))
	for i, k := range keys {
		views[i] = keyView{ID: string(k.ID()), Key: k.String()}
	}
	data, err := json.MarshalIndent(views, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling keys: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

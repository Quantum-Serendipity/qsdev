# Raises go_1_26 to at least the security-fixed release below. A newer
# go_1_26 from nixpkgs is kept as is: a fixed override would downgrade Go for
# every package built from this nixpkgs once nixpkgs moves past the pin (a
# 1.26.7 pin under a nixpkgs that ships 1.26.8 broke syft, which requires
# go >= 1.26.8). go.mod's go line must not exceed the version below.
final: prev:
let
  version = "1.26.9";
in
{
  go_1_26 =
    if prev.lib.versionAtLeast prev.go_1_26.version version then
      prev.go_1_26
    else
      prev.go_1_26.overrideAttrs (old: {
        inherit version;
        src = prev.fetchurl {
          url = "https://go.dev/dl/go${version}.src.tar.gz";
          hash = "sha256-lzXX3Ntls10/pXfwQGRzfAO4nPGitx5uaf4vPG+f1Mo=";
        };
      });
}

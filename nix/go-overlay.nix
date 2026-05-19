final: prev: {
  go_1_26 = prev.go_1_26.overrideAttrs (old: {
    version = "1.26.3";
    src = prev.fetchurl {
      url = "https://go.dev/dl/go1.26.3.src.tar.gz";
      hash = "sha256-HGRoddCqh5kTMYTtV895/yS97+jIggRwYCqdPW2Rkrg=";
    };
  });
}

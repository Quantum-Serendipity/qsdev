final: prev: {
  go_1_26 = prev.go_1_26.overrideAttrs (old: {
    version = "1.26.7";
    src = prev.fetchurl {
      url = "https://go.dev/dl/go1.26.7.src.tar.gz";
      hash = "sha256-DtJOrHVRBQhbif6cq8J0K5GgrXuUtZ0602SRjryJVq0=";
    };
  });
}

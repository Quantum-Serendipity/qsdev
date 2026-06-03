final: prev: {
  go_1_26 = prev.go_1_26.overrideAttrs (old: {
    version = "1.26.4";
    src = prev.fetchurl {
      url = "https://go.dev/dl/go1.26.4.src.tar.gz";
      hash = "sha256-T2aKMvv8ETLmqIH7lowvHa2mMUkqM5IRc1+7JVpCYC0=";
    };
  });
}

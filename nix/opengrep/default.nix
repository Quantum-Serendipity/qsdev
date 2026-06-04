{ pkgs, lib, version ? "1.21.0", ... }:

let
  platform = pkgs.stdenv.hostPlatform.system;

  sources = {
    x86_64-linux = {
      url = "https://github.com/opengrep/opengrep/releases/download/v${version}/opengrep-core-${version}-linux-x86_64";
      # TODO: Replace with real hash after first successful fetch.
      # Run: nix-prefetch-url <url> to obtain the correct hash.
      hash = lib.fakeHash;
    };
    aarch64-linux = {
      url = "https://github.com/opengrep/opengrep/releases/download/v${version}/opengrep-core-${version}-linux-aarch64";
      # TODO: Replace with real hash after first successful fetch.
      # Run: nix-prefetch-url <url> to obtain the correct hash.
      hash = lib.fakeHash;
    };
  };

  src = sources.${platform} or (throw "opengrep: unsupported platform ${platform}");
in

pkgs.stdenv.mkDerivation {
  pname = "opengrep";
  inherit version;

  src = pkgs.fetchurl {
    inherit (src) url hash;
    executable = true;
  };

  dontUnpack = true;

  nativeBuildInputs = [
    pkgs.autoPatchelfHook
  ];

  buildInputs = [
    pkgs.stdenv.cc.cc.lib
    pkgs.pcre2
  ];

  installPhase = ''
    runHook preInstall
    install -Dm755 $src $out/bin/opengrep
    ln -s opengrep $out/bin/opengrep-core
    runHook postInstall
  '';

  meta = {
    description = "Prebuilt OpenGrep static analysis engine";
    homepage = "https://github.com/opengrep/opengrep";
    license = lib.licenses.lgpl21;
    platforms = [ "x86_64-linux" "aarch64-linux" ];
    mainProgram = "opengrep";
  };
}

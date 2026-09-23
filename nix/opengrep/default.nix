{ pkgs ? import <nixpkgs> { }, lib ? pkgs.lib, version ? "1.21.0", ... }:

let
  platform = pkgs.stdenv.hostPlatform.system;

  # The standalone `opengrep` CLI (the one `opengrep scan` runs), not the
  # opengrep-core OCaml engine. Hashes are the release asset digests; update
  # them together with `version` (the digest is listed per asset by
  # `gh api repos/opengrep/opengrep/releases/tags/v<version>`).
  sources = {
    x86_64-linux = {
      asset = "opengrep_manylinux_x86";
      hash = "sha256-ntDO7ko6QG0n1AiUvM6F6hUb4h5tSxgGiWiSJPr/Kj4=";
    };
    aarch64-linux = {
      asset = "opengrep_manylinux_aarch64";
      hash = "sha256-638R0VO1XelIJ5Ww9aYziAGLK+qYKrXSUfjhFmJa+JI=";
    };
  };

  src = sources.${platform} or (throw "opengrep: unsupported platform ${platform}");
in

pkgs.stdenv.mkDerivation {
  pname = "opengrep";
  inherit version;

  src = pkgs.fetchurl {
    url = "https://github.com/opengrep/opengrep/releases/download/v${version}/${src.asset}";
    # Flat file hash, so it equals the published release asset digest.
    inherit (src) hash;
  };

  dontUnpack = true;

  nativeBuildInputs = [
    pkgs.autoPatchelfHook
  ];

  buildInputs = [
    pkgs.stdenv.cc.cc.lib
    pkgs.zlib
  ];

  # The release binary is a Nuitka onefile bundle: at run time it unpacks a
  # standalone tree (opengrep.bin, its Python extension modules and
  # opengrep-core) into the user cache and execs it. That unpacked tree
  # expects an FHS loader, so it cannot run on NixOS. Unpack it here instead
  # (the launcher's own exec of the unpatched child fails, which is expected)
  # and install the tree, which autoPatchelfHook then fixes up.
  buildPhase = ''
    runHook preBuild
    install -m755 $src ./opengrep-onefile
    patchelf --set-interpreter "$(cat $NIX_CC/nix-support/dynamic-linker)" ./opengrep-onefile
    XDG_CACHE_HOME=$PWD/cache HOME=$PWD ./opengrep-onefile --version || true
    test -x cache/opengrep/v${version}/opengrep.bin
    runHook postBuild
  '';

  installPhase = ''
    runHook preInstall
    mkdir -p $out/libexec $out/bin
    cp -r cache/opengrep/v${version} $out/libexec/opengrep
    ln -s ../libexec/opengrep/opengrep.bin $out/bin/opengrep
    runHook postInstall
  '';

  doInstallCheck = true;
  installCheckPhase = ''
    runHook preInstallCheck
    HOME=$TMPDIR $out/bin/opengrep --version | grep -F ${version}
    runHook postInstallCheck
  '';

  meta = {
    description = "Prebuilt OpenGrep static analysis CLI";
    homepage = "https://github.com/opengrep/opengrep";
    license = lib.licenses.lgpl21;
    platforms = [ "x86_64-linux" "aarch64-linux" ];
    sourceProvenance = [ lib.sourceTypes.binaryNativeCode ];
    mainProgram = "opengrep";
  };
}

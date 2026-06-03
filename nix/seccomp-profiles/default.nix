{ pkgs }:

let
  compiler = pkgs.stdenv.mkDerivation {
    pname = "seccomp-filter-compiler";
    version = "0.1.0";

    src = ./hook-blocklist.c;
    unpackPhase = "true";

    buildInputs = [ pkgs.libseccomp ];

    buildPhase = ''
      $CC -O2 -Wall -o gen-filter $src -lseccomp
    '';

    installPhase = ''
      mkdir -p $out/bin
      install -m755 gen-filter $out/bin/
    '';

    meta.platforms = pkgs.lib.platforms.linux;
  };

  filter = pkgs.runCommand "hook-seccomp-filter" {
    nativeBuildInputs = [ compiler ];
  } ''
    mkdir -p $out
    gen-filter > $out/hook-blocklist.bpf
  '';
in
{
  inherit compiler filter;
}

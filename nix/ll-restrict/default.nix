{ pkgs }:

pkgs.pkgsStatic.stdenv.mkDerivation {
  pname = "ll-restrict";
  version = "0.1.0";

  src = ./ll-restrict.c;
  unpackPhase = "true";

  buildPhase = ''
    $CC -O2 -Wall -Wextra -o ll-restrict $src
  '';

  installPhase = ''
    mkdir -p $out/bin
    install -m755 ll-restrict $out/bin/
  '';

  meta = with pkgs.lib; {
    description = "Landlock filesystem restriction helper for hook sandboxing";
    platforms = platforms.linux;
  };
}

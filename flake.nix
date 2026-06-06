{
  description = "qsdev — secure developer environment bootstrapping tool";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    let
      goOverlay = import ./nix/go-overlay.nix;
    in
    {
      overlays.default = final: prev: {
        qsdev = self.packages.${prev.system}.qsdev;
      };

      nixosModules.default = { config, lib, pkgs, ... }: {
        programs.direnv.enable = lib.mkDefault true;
        programs.direnv.nix-direnv.enable = lib.mkDefault true;

        environment.systemPackages = [
          self.packages.${pkgs.system}.qsdev
          pkgs.devenv
        ];
      };
    }
    //
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ goOverlay ];
        };
        go = pkgs.go_1_26;

        baseVersion = builtins.replaceStrings [ "\n" " " ] [ "" "" ]
          (builtins.readFile ./VERSION);

        version =
          if (self ? shortRev)
          then "${baseVersion}+${self.shortRev}"
          else "${baseVersion}+dirty";

        commit = self.shortRev or "dirty";
        date = self.lastModifiedDate or "unknown";

        ldflags = [
          "-s" "-w"
          "-X" "github.com/Quantum-Serendipity/qsdev/internal/version.version=${version}"
          "-X" "github.com/Quantum-Serendipity/qsdev/internal/version.commit=${commit}"
          "-X" "github.com/Quantum-Serendipity/qsdev/internal/version.date=${date}"
          "-X" "github.com/Quantum-Serendipity/qsdev/internal/version.builtBy=nix"
        ];

        sandboxPkgs = if pkgs.stdenv.isLinux then {
          ll-restrict = import ./nix/ll-restrict { inherit pkgs; };
          seccomp-profiles = import ./nix/seccomp-profiles { inherit pkgs; };
        } else {
          ll-restrict = null;
          seccomp-profiles = null;
        };
      in
      {
        packages = rec {
          qsdev = pkgs.buildGoModule.override { inherit go; } {
            pname = "qsdev";
            inherit version;

            src = pkgs.lib.cleanSourceWith {
              src = ./.;
              filter = path: type:
                let name = baseNameOf path;
                in name != "go.work" && name != "go.work.sum";
            };

            vendorHash = null;

            env.CGO_ENABLED = "0";

            ldflags = ldflags ++ pkgs.lib.optionals (sandboxPkgs.ll-restrict != null) [
              "-X" "github.com/Quantum-Serendipity/qsdev/internal/sandbox.llRestrictPath=${sandboxPkgs.ll-restrict}/bin/ll-restrict"
              "-X" "github.com/Quantum-Serendipity/qsdev/internal/sandbox.seccompFilterPath=${sandboxPkgs.seccomp-profiles.filter}/hook-blocklist.bpf"
            ];
            flags = [ "-trimpath" ];

            subPackages = [ "cmd/qsdev" ];

            doCheck = false;

            nativeBuildInputs = [ pkgs.git pkgs.installShellFiles pkgs.makeWrapper pkgs.syft ];

            postInstall = ''
              installShellCompletion --cmd qsdev \
                --bash <($out/bin/qsdev completion bash) \
                --zsh  <($out/bin/qsdev completion zsh) \
                --fish <($out/bin/qsdev completion fish)

              mkdir -p $out/share/sbom
              ${pkgs.syft}/bin/syft dir:. -o cyclonedx-json=$out/share/sbom/qsdev.cdx.json
            '';

            postFixup = ''
              wrapProgram $out/bin/qsdev \
                --prefix PATH : ${pkgs.lib.makeBinPath ([ pkgs.git ]
                  ++ pkgs.lib.optionals pkgs.stdenv.isLinux [ pkgs.bubblewrap ])}
            '';

            meta = with pkgs.lib; {
              description = "Secure developer environment bootstrapping tool";
              homepage = "https://github.com/Quantum-Serendipity/qsdev";
              mainProgram = "qsdev";
              platforms = platforms.linux ++ platforms.darwin;
            };
          };

          default = qsdev;
        } // pkgs.lib.optionalAttrs pkgs.stdenv.isLinux {
          inherit (sandboxPkgs) ll-restrict;
          seccomp-filter = sandboxPkgs.seccomp-profiles.filter;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = [
            go
            pkgs.git
            pkgs.goreleaser
            pkgs.golangci-lint
            pkgs.gopls
            pkgs.delve
            pkgs.syft
          ];
        };
      });
}

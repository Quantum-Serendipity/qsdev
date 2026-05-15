{
  description = "qsdev — secure developer environment bootstrapping tool";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    {
      overlays.default = final: prev: {
        qsdev = self.packages.${prev.system}.qsdev;
      };
    }
    //
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        go = pkgs.go_1_26;

        version =
          if (self ? shortRev)
          then "0.3.2+${self.shortRev}"
          else "0.3.2+dirty";

        commit = self.shortRev or "dirty";
        date = self.lastModifiedDate or "unknown";

        ldflags = [
          "-s" "-w"
          "-X" "github.com/Quantum-Serendipity/qsdev/internal/version.version=${version}"
          "-X" "github.com/Quantum-Serendipity/qsdev/internal/version.commit=${commit}"
          "-X" "github.com/Quantum-Serendipity/qsdev/internal/version.date=${date}"
          "-X" "github.com/Quantum-Serendipity/qsdev/internal/version.builtBy=nix"
        ];
      in
      {
        packages = rec {
          qsdev = pkgs.buildGoModule.override { go = go; } {
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

            inherit ldflags;
            flags = [ "-trimpath" ];

            subPackages = [ "cmd/qsdev" ];

            doCheck = false;

            nativeBuildInputs = [ pkgs.git pkgs.installShellFiles pkgs.makeWrapper ];

            postInstall = ''
              installShellCompletion --cmd qsdev \
                --bash <($out/bin/qsdev completion bash) \
                --zsh  <($out/bin/qsdev completion zsh) \
                --fish <($out/bin/qsdev completion fish)
            '';

            postFixup = ''
              wrapProgram $out/bin/qsdev \
                --prefix PATH : ${pkgs.lib.makeBinPath [ pkgs.git ]}
            '';

            meta = with pkgs.lib; {
              description = "Secure developer environment bootstrapping tool";
              homepage = "https://github.com/Quantum-Serendipity/qsdev";
              mainProgram = "qsdev";
              platforms = platforms.linux ++ platforms.darwin;
            };
          };

          default = qsdev;
        };

        devShells.default = pkgs.mkShell {
          packages = [
            go
            pkgs.gopls
            pkgs.gotools
            pkgs.golangci-lint
            pkgs.delve
            pkgs.goreleaser
          ];

          shellHook = ''
            export GOPATH="$PWD/.go"
            export PATH="$GOPATH/bin:$PATH"
            echo "qsdev dev environment loaded"
            echo "  Go: $(go version)"
          '';
        };
      });
}

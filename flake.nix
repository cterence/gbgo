{
  description = "A Nix-flake-based Go development environment";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-unstable";

    pre-commit-hooks = {
      url = "github:cachix/git-hooks.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      pre-commit-hooks,
    }:
    let
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forEachSupportedSystem =
        f:
        nixpkgs.lib.genAttrs supportedSystems (
          system:
          f {
            pkgs = import nixpkgs {
              inherit system;
            };
          }
        );
    in
    {
      devShells = forEachSupportedSystem (
        { pkgs }:
        {
          default = pkgs.mkShell {
            shellHook = ''
              export LD_LIBRARY_PATH=${pkgs.lib.getLib pkgs.alsa-lib}/lib:${pkgs.lib.getLib pkgs.wayland}/lib:${pkgs.lib.getLib pkgs.libGL}/lib:${pkgs.lib.getLib pkgs.libdecor}/lib:${pkgs.lib.getLib pkgs.libxkbcommon}/lib:$LD_LIBRARY_PATH
              ${self.checks.${pkgs.stdenv.hostPlatform.system}.pre-commit-check.shellHook}
            '';
            hardeningDisable = [ "fortify" ]; # Make delve work with direnv IDE extension
            nativeBuildInputs = with pkgs; [
              go
            ];
            packages = with pkgs; [
              air
              gotools
              gopls
              self.checks.${stdenv.hostPlatform.system}.pre-commit-check.enabledPackages
            ];
          };
        }
      );

      checks = forEachSupportedSystem (
        { pkgs }:
        {
          pre-commit-check = pre-commit-hooks.lib.${pkgs.stdenv.hostPlatform.system}.run {
            src = ./.;
            hooks = {
              gofmt.enable = true;
              golangci-lint.enable = true;
              govet.enable = true;
              betteralign = {
                enable = true;
                name = "betteralign";
                entry = "betteralign ./...";
                pass_filenames = false;
              };
            };
          };
        }
      );
    };
}

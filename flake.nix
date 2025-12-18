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
            CGO_ENABLED = 1;
            LD_LIBRARY_PATH = pkgs.lib.makeLibraryPath [
              pkgs.libdecor
            ];
            shellHook = ''
              ${self.checks.${pkgs.stdenv.hostPlatform.system}.pre-commit-check.shellHook}
            '';
            hardeningDisable = [ "fortify" ]; # Make delve work with direnv IDE extension
            buildInputs = with pkgs; [
              go
              wayland
              libxkbcommon
              xorg.libX11
              xorg.libXcursor
              xorg.libXrandr
              xorg.libXinerama
              xorg.libXi
              libGL
            ];
            packages = with pkgs; [
              air
              gotools
              gopls
              rgbds
              sameboy
              wla-dx
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

      packages = forEachSupportedSystem (
        { pkgs }:
        {
          default = pkgs.buildGoModule {
            pname = "gbgo";
            version = "0.1.0";
            src = ./.;
            vendorHash = "sha256-uDOGPUAVrQiFyJaNaIviu5w+9kfMNA/9rwgVidpc/Js=";
            doCheck = false;

            buildInputs = with pkgs; [
              go
              wayland
              libxkbcommon
              xorg.libX11
              xorg.libXcursor
              xorg.libXrandr
              xorg.libXinerama
              xorg.libXi
              libGL
            ];
          };
        }
      );
    };
}

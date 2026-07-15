{
  description = "ARC development flake";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";

    go-overlay = {
      url = "github:purpleclay/go-overlay";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-utils.follows = "flake-utils";
    };

    dev-kit = {
      url = "github:opendefensecloud/dev-kit";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.go-overlay.follows = "go-overlay";
      inputs.flake-utils.follows = "flake-utils";
    };
  };

  outputs = { nixpkgs, flake-utils, dev-kit, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        devShells.default = dev-kit.lib.mkShell {
          inherit system;
          goVersion = "1.26.5";
          packages = [
            pkgs.cosign
            pkgs.trivy
          ];
          preCommitHooks = {
            commitlint.enable = true;
          };
        };
      }
    );
}

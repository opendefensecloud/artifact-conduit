{
  description = "ARC development flake";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    go-overlay = {
      url = "github:purpleclay/go-overlay";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    dev-kit = {
      url = "github:opendefensecloud/dev-kit?ref=8f600cc0fed51689db015a0efefe8f127e4cce43";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.go-overlay.follows = "go-overlay";
    };
  };

  outputs = { nixpkgs, flake-utils, dev-kit, ... }@inputs:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        devShells.default = dev-kit.lib.mkShell {
          inherit system;
          goVersion = "1.26.2";
          packages = [
            pkgs.cosign
            pkgs.trivy
          ];
        };
      }
    );
}

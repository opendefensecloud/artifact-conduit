{
  pkgs,
  lib,
  config,
  inputs,
  ...
}:
let
  pkgs-unstable = import inputs.nixpkgs-unstable { system = pkgs.stdenv.system; };
in
{
  # https://devenv.sh/packages/
  packages = [
    # required by make target
    pkgs.gnumake
    pkgs.jq
    pkgs.shellcheck
    pkgs.yq-go
    pkgs.kind
    pkgs.kubectl
    pkgs.kubernetes-helm
    # for local testing of workflows etc.
    pkgs.cosign
    pkgs.trivy
    pkgs.osv-scanner
  ];

  # https://devenv.sh/languages/
  languages.go.enable = true;
  languages.go.version = "1.25.7";

  git-hooks.hooks = {
    gofmt.enable = true;
    golangci-lint.enable = true;
    osv-scanner = {
      enable = true;
      name = "osv-scanner";
      entry = "osv-scanner scan -r .";
      files = "\\.(mod|sum)$";
      pass_filenames = false;
    };
  };
  # See full reference at https://devenv.sh/reference/options/

  difftastic.enable = true;
  delta.enable = true;
}

{ pkgs, lib, config, inputs, ... }:

{
  # https://devenv.sh/basics/
  env.GREET = "ARC DevEnv";

  # https://devenv.sh/packages/
  packages = [
    pkgs.git
    pkgs.gnumake
    pkgs.cobra-cli
    pkgs.golangci-lint
    pkgs.govulncheck
    pkgs.addlicense

    pkgs.kubernetes-code-generator
  ];

  # https://devenv.sh/languages/
  languages.go.enable = true;
  languages.go.package = pkgs.go_1_25;

  git-hooks.hooks = {
    govet = {
      enable = true;
      pass_filenames = false;
    };
    golangci-lint = {
      enable = true;
      pass_filenames = false;
    };
  };
  # See full reference at https://devenv.sh/reference/options/
}

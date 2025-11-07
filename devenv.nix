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

  scripts.gss.exec = ''
    git status --short
  '';
  scripts.gp.exec = ''
    git push
  '';
  scripts.gl.exec = ''
    git pull
  '';
  scripts.gcam.exec = ''
    git commit --all --message "$@";
  '';

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

  difftastic.enable = true;
  delta.enable = true;
}

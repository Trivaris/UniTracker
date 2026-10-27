{ pkgs ? import <nixpkgs> { } }:
pkgs.mkShell {
  packages = [
    pkgs.go
    pkgs.gopls
    pkgs.jq
    pkgs.docker
  ];

shellHook = ''
  mkdir -p .vscode
  [ ! -f .vscode/settings.json ] && echo "{}" > .vscode/settings.json
  jq '.["go.goroot"] = "${pkgs.go}/share/go"' .vscode/settings.json > .vscode/settings.json
'';
}
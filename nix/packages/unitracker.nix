{ pkgs ? import <nixpkgs> { } }:
pkgs.stdenv.mkDerivation {
  name = "UniTracker";
  version = "1.0.0";
  src = ../../.;

  nativeBuildInputs = [ pkgs.go ];

  buildPhase = ''
    export GOPATH=$TMPDIR
    export GOCACHE=$TMPDIR/go-cache
    go build -o UniTracker ./cmd/server
  '';

  installPhase = ''
    mkdir -p $out/bin
    cp UniTracker $out/bin/UniTracker
    chmod +x $out/bin/UniTracker
  '';
}
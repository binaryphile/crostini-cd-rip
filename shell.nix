{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = with pkgs; [
    go
    libusb1
    pkg-config  # for gousb CGO
    lame
  ];

  shellHook = ''
    echo "crostini-cd-rip Go development environment"
    echo "Run: go build ./cmd/cd-rip && ./cd-rip --help"
  '';
}

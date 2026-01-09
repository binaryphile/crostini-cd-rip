{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = with pkgs; [
    python3
    python3Packages.pyusb
    lame  # For MP3 encoding
  ];

  shellHook = ''
    echo "crostini-cd-rip development environment"
    echo "Run: ./crostini-cd-rip.py --help"
  '';
}

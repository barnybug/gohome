{
  description = "gohome home automation";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        packages.default = pkgs.buildGoModule {
          pname = "gohome";
          version = "0.0.1";
          src = ./.;
          vendorHash = "sha256-JxToHZUROCvffMN9hZy/W7m2G9w/GaxHU9fY4sQqCuQ=";
        };
      }
    );
}

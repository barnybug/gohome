{
  description = "gohome home automation";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        # System specific outputs becomes: packages.<system>.default
        packages.default = pkgs.buildGoModule {
          pname = "gohome";
          version = "0.0.1";
          src = ./.;
          vendorHash = "sha256-JxToHZUROCvffMN9hZy/W7m2G9w/GaxHU9fY4sQqCuQ=";
        };
      }
    )
    // {
      # Flake-wide outputs
      homeManagerModules = {
        default =
          {
            config,
            lib,
            pkgs,
            ...
          }:
          with lib;
          let
            cfg = config.programs.gohome;
            pkg = self.packages.${pkgs.hostPlatform.system}.default;
            serviceOptionsType =
              with types;
              let
                primitive = oneOf [
                  bool
                  int
                  str
                  path
                ];
              in
              attrsOf (attrsOf (attrsOf (either primitive (listOf primitive))));
          in
          {
            options = {
              programs.gohome = {
                enable = mkEnableOption "Gohome";
                extraServiceOptions = mkOption {
                  type = serviceOptionsType;
                  default = { };
                  description = "Extra systemd options";
                };
                mqtt = mkOption {
                  type = types.str;
                  default = "tcp://mqtt:1883";
                  description = "mqtt url";
                };
                path = mkOption {
                  type = types.str;
                  default = "";
                  description = "Set PATH environment variable";
                };
                services = mkOption {
                  type = types.listOf types.str;
                  default = [ ];
                  description = "List of gohome services to enable";
                };
              };
            };
            config = mkMerge [
              (mkIf cfg.enable {
                home.packages = [
                  pkg
                  pkgs.coreutils
                  pkgs.systemd
                  pkgs.tcpdump
                ];
              })
              (mkIf (length cfg.services != 0) {
                systemd.user.enable = true;
                # home manager services doesn't appear to support templated services, so
                # produce a service unit per instance.
                systemd.user.services =
                  let
                    makeService = (
                      n: {
                        name = "gohome@${n}";
                        value = {
                          Install.WantedBy = [ "default.target" ];
                          Unit = {
                            Description = "gohome service ${n}";
                            StartLimitIntervalSec = "0";
                          };
                          Service = {
                            Environment = [
                              "GOHOME_MQTT=${cfg.mqtt}"
                              "GOHOME_API=http://localhost:8723/"
                              "PATH=${cfg.path}"
                            ];
                            ExecStart = "${pkg}/bin/gohome run ${n}";
                            Restart = "always";
                            RestartSec = "5s";
                            WatchdogSec = "120s";
                            Type = "notify";
                            NotifyAccess = "main";
                          };
                        } // cfg.extraServiceOptions;
                      }
                    );
                  in
                  listToAttrs (map makeService cfg.services);
              })
            ];
          };
      };
    };
}

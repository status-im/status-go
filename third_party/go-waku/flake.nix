{
  description = "Nix flake for Go implementaion of Waku v2 node.";

  inputs.nixpkgs.url = github:NixOS/nixpkgs/nixos-24.11;

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [
        "x86_64-linux" "i686-linux" "aarch64-linux"
        "x86_64-darwin" "aarch64-darwin"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs supportedSystems (system: f system);
      pkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
    in rec {
      packages = forAllSystems (system: let
        pkgs = pkgsFor.${system};
        os = pkgs.stdenv.hostPlatform.uname.system;
        sttLibExtMap = { Windows = "lib"; Darwin = "a";     Linux = "a";  };
        dynLibExtMap = { Windows = "dll"; Darwin = "dylib"; Linux = "so"; };
        buildPackage = pkgs.callPackage ./default.nix;
      in rec {
        default = node;
        node = buildPackage {
          inherit self;
          subPkgs = ["cmd/waku"];
        };
        static-library = buildPackage {
          inherit self;
          subPkgs = ["library/c"];
          ldflags = ["-buildmode=c-archive"];
          output = "libgowaku.${sttLibExtMap.${os}}";
        };
        # FIXME: Compilation fails with:
        #   relocation R_X86_64_TPOFF32 against runtime.tlsg can not be
        #   used when making a shared object; recompile with -fPIC
        dynamic-library = buildPackage {
          inherit self;
          subPkgs = ["library/c"];
          ldflags = ["-buildmode=c-shared"];
          output = "libgowaku.${dynLibExtMap.${os}}";
        };
      });

      devShells = forAllSystems (system: let
        pkgs = pkgsFor.${system};
        inherit (pkgs) lib stdenv mkShell;
      in {
        default = mkShell {
          GOFLAGS = "-trimpath"; # Drop -mod=vendor
          inputsFrom = [ packages.${system}.node ];
          buildInputs = with pkgs; [ golangci-lint ];
          nativeBuildInputs = lib.optional stdenv.isDarwin [
            (pkgs.xcodeenv.composeXcodeWrapper { version = "14.2"; allowHigher = true; })
          ];
        };

        fpm = mkShell {
          buildInputs = with pkgs; [ fpm ];
        };
      });
  };
}

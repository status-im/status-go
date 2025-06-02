# Description

This folder contains configuration for [Nix](https://nixos.org/), a purely functional package manager used by the Status Go for its build process.

## Configuration

The main config file is [`flake.nix`](./flake.nix) and its main purpose is defining the development shells, packages and other configuration options among which [binary caches](https://nixos.org/nix/manual/#ch-basic-package-mgmt) which allow download of packages to avoid having to compile them yourself locally.


## Shell

In order to access an interactive Nix shell a user should run `make shell` or `nix develop`.

The Nix shell is started in this repo via the `nix develop` command.

:warning: __WARNING__: To have Nix pick up all changes a new `nix develop` needs to be spawned.

## Resources

You can learn more about Nix by watching these presentations:

* [Nix Fundamentals](https://www.youtube.com/watch?v=m4sv2M9jRLg) ([PDF](https://drive.google.com/file/d/1Tt5R7QOubudGiSuZIGxuFWB1OYgcThcL/view?usp=sharing), [src](https://github.com/status-im/infra-docs/tree/master/presentations/nix_basics))
* [Nix in Status](https://www.youtube.com/watch?v=rEQ1EvRG8Wc) ([PDF](https://drive.google.com/file/d/1Ti0wppMoj40icCPdHy7mJcQj__DeaYBE/view?usp=sharing), [src](https://github.com/status-im/infra-docs/tree/master/presentations/nix_in_status))
* [Nix Flakes](https://wiki.nixos.org/wiki/Flakes)

## Known Issues

See [`KNOWN_ISSUES.md`](./KNOWN_ISSUES.md).

# Description

This folder contains configuration for [Nix](https://nixos.org/), a purely functional package manager used by the Status Go for its build process.

## Configuration

The main config file is [`nix/nix.conf`](/nix/nix.conf) and its main purpose is defining the [binary caches](https://nixos.org/nix/manual/#ch-basic-package-mgmt) which allow download of packages to avoid having to compile them yourself locally.


## Shell

In order to access an interactive Nix shell a user should run `make shell`.

The Nix shell is started in this repo via the [`nix/scripts/shell.sh`](/nix/scripts/shell.sh) script, which is a wrapper around the `nix-shell` command and is intended for use with our main [`Makefile`](/Makefile). This allows for an implicit use of `nix-shell` as the default shell in the `Makefile`.

:warning: __WARNING__: To have Nix pick up all changes a new `nix-shell` needs to be spawned.

## Resources

You can learn more about Nix by watching these presentations:

* [Nix Fundamentals](https://www.youtube.com/watch?v=m4sv2M9jRLg) ([PDF](https://drive.google.com/file/d/1Tt5R7QOubudGiSuZIGxuFWB1OYgcThcL/view?usp=sharing), [src](https://github.com/status-im/infra-docs/tree/master/presentations/nix_basics))
* [Nix in Status](https://www.youtube.com/watch?v=rEQ1EvRG8Wc) ([PDF](https://drive.google.com/file/d/1Ti0wppMoj40icCPdHy7mJcQj__DeaYBE/view?usp=sharing), [src](https://github.com/status-im/infra-docs/tree/master/presentations/nix_in_status))

And you can read [`nix/DETAILS.md`](./DETAILS.md) for more information.

## Known Issues

See [`KNOWN_ISSUES.md`](./KNOWN_ISSUES.md).

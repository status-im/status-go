# Nimble srcDir placeholder

`sds_go_bindings.nimble` needs a `srcDir`, and this repository has no Nim source. Pointing it here
keeps the installed Nimble package empty: only the `.nimble` file itself is installed, so nothing
reaches a dependent's Nim module path.

Do not put Nim modules here.

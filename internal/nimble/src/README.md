# Nimble srcDir placeholder

`status_go.nimble` needs a `srcDir`, and status-go has no Nim source. Pointing it here keeps the
installed Nimble package empty: only the `.nimble` file itself is installed, so nothing of status-go
reaches a dependent's Nim module path.

Do not put Nim modules here.

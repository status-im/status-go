# No Nim code here

`status_go.nimble` points `srcDir` at this empty directory. The installed package carries
status-go's full source (to build libstatus from it), but a dependent's Nim module path gets only
this directory, so no status-go file is importable from Nim.

Do not put Nim modules here.

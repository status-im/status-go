### Library bindings 

This unholy namespace generates C bindings for `mobile/`.
It does so by parsing the go files, identify any public function in the namespace
that is not a test file, and generate an equivalent function with the correct 
C signature, which will then just call the version in `mobile/`.

This method is ad-hoc and not bullet-proof, the main limitation is that only some 
types are supported for now, `C.int`, `*C.char` and `unsafer.Pointer`. 
Functions that  do not use these types will be ignored.

The problem this sledgehammer solves is to have to keep in sync the two namespaces,
which is fiddly and prone to error as they are modified by either teams.

//go:build android && go1.27

package statusgo

// runtime.adviseUnused (madvise_android.go) is verified against Go 1.26 only. Check that
// runtime/mem_linux.go still has it with the same semantics, then bump the constraint above.
var _ = adviseUnusedNotVerifiedForThisGoVersion

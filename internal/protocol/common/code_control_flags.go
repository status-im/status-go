package common

type CodeControlFlags struct {
	// AutoRequestHistoricMessages indicates whether we should automatically request
	// historic messages on getting online, connecting to store node, etc.
	AutoRequestHistoricMessages bool
}

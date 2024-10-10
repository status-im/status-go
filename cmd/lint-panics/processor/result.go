package processor

type Result struct {
	paths []string
}

func NewResult() *Result {
	return &Result{}
}

func (r *Result) Paths() []string {
	return r.paths
}

func (r *Result) Add(uri string) {
	r.paths = append(r.paths, uri)
}

func (r *Result) Merge(other *Result) {
	if other != nil && r != nil {
		r.paths = append(r.paths, other.paths...)
	}
}

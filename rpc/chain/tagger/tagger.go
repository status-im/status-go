package tagger

type Tagger interface {
	Tag() string
	SetTag(tag string)
	GroupTag() string
	SetGroupTag(tag string)
	DeepCopyTag() Tagger
}

func DeepCopyTagger(t Tagger) Tagger {
	return t.DeepCopyTag()
}

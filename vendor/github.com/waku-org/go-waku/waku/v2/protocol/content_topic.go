package protocol

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidFormat = errors.New("invalid content topic format")
var ErrMissingGeneration = errors.New("missing part: generation")
var ErrInvalidGeneration = errors.New("generation should be a number")

// ContentTopic is used for content based.
type ContentTopic struct {
	ContentTopicParams
	ApplicationName    string
	ApplicationVersion uint32
	ContentTopicName   string
	Encoding           string
}

// ContentTopicParams contains all the optional params for a content topic
type ContentTopicParams struct {
	Generation int
}

// Equal method used to compare 2 contentTopicParams
func (ctp ContentTopicParams) Equal(ctp2 ContentTopicParams) bool {
	return ctp.Generation == ctp2.Generation
}

// ContentTopicOption is following the options pattern to define optional params
type ContentTopicOption func(*ContentTopicParams)

// String formats a content topic in string format as per RFC 23.
func (ct ContentTopic) String() string {
	return fmt.Sprintf("/%s/%d/%s/%s", ct.ApplicationName, ct.ApplicationVersion, ct.ContentTopicName, ct.Encoding)
}

// NewContentTopic creates a new content topic based on params specified.
// Returns ErrInvalidGeneration if an unsupported generation is specified.
func NewContentTopic(applicationName string, applicationVersion uint32,
	contentTopicName string, encoding string, opts ...ContentTopicOption) (ContentTopic, error) {

	params := new(ContentTopicParams)
	optList := DefaultOptions()
	optList = append(optList, opts...)
	for _, opt := range optList {
		opt(params)
	}
	if params.Generation > 0 {
		return ContentTopic{}, ErrInvalidGeneration
	}
	return ContentTopic{
		ContentTopicParams: *params,
		ApplicationName:    applicationName,
		ApplicationVersion: applicationVersion,
		ContentTopicName:   contentTopicName,
		Encoding:           encoding,
	}, nil
}

// WithGeneration option can be used to specify explicitly a generation for contentTopic
func WithGeneration(generation int) ContentTopicOption {
	return func(params *ContentTopicParams) {
		params.Generation = generation
	}
}

// DefaultOptions sets default values for contentTopic optional params.
func DefaultOptions() []ContentTopicOption {
	return []ContentTopicOption{
		WithGeneration(0),
	}
}

// Equal to compare 2 content topics.
func (ct ContentTopic) Equal(ct2 ContentTopic) bool {
	return ct.ApplicationName == ct2.ApplicationName && ct.ApplicationVersion == ct2.ApplicationVersion &&
		ct.ContentTopicName == ct2.ContentTopicName && ct.Encoding == ct2.Encoding &&
		ct.ContentTopicParams.Equal(ct2.ContentTopicParams)
}

// StringToContentTopic can be used to create a ContentTopic object from a string
func StringToContentTopic(s string) (ContentTopic, error) {
	p := strings.Split(s, "/")
	switch len(p) {
	case 5:
		vNum, err := strconv.ParseUint(p[2], 10, 32)
		if err != nil {
			return ContentTopic{}, ErrInvalidFormat
		}

		return ContentTopic{
			ApplicationName:    p[1],
			ApplicationVersion: uint32(vNum),
			ContentTopicName:   p[3],
			Encoding:           p[4],
		}, nil
	case 6:
		if len(p[1]) == 0 {
			return ContentTopic{}, ErrMissingGeneration
		}
		generation, err := strconv.Atoi(p[1])
		if err != nil || generation > 0 {
			return ContentTopic{}, ErrInvalidGeneration
		}
		vNum, err := strconv.ParseUint(p[3], 10, 32)
		if err != nil {
			return ContentTopic{}, ErrInvalidFormat
		}

		return ContentTopic{
			ContentTopicParams: ContentTopicParams{Generation: generation},
			ApplicationName:    p[2],
			ApplicationVersion: uint32(vNum),
			ContentTopicName:   p[4],
			Encoding:           p[5],
		}, nil
	default:
		return ContentTopic{}, ErrInvalidFormat
	}
}

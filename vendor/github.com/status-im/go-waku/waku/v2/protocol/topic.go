package protocol

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidFormat = errors.New("invalid format")

type ContentTopic struct {
	ApplicationName    string
	ApplicationVersion uint
	ContentTopicName   string
	Encoding           string
}

func (ct ContentTopic) String() string {
	return fmt.Sprintf("/%s/%d/%s/%s", ct.ApplicationName, ct.ApplicationVersion, ct.ContentTopicName, ct.Encoding)
}

func NewContentTopic(applicationName string, applicationVersion uint, contentTopicName string, encoding string) ContentTopic {
	return ContentTopic{
		ApplicationName:    applicationName,
		ApplicationVersion: applicationVersion,
		ContentTopicName:   contentTopicName,
		Encoding:           encoding,
	}
}

func (ct ContentTopic) Equal(ct2 ContentTopic) bool {
	return ct.ApplicationName == ct2.ApplicationName && ct.ApplicationVersion == ct2.ApplicationVersion &&
		ct.ContentTopicName == ct2.ContentTopicName && ct.Encoding == ct2.Encoding
}

func StringToContentTopic(s string) (ContentTopic, error) {
	p := strings.Split(s, "/")

	if len(p) != 5 || p[0] != "" || p[1] == "" || p[2] == "" || p[3] == "" || p[4] == "" {
		return ContentTopic{}, ErrInvalidFormat
	}

	vNum, err := strconv.ParseUint(p[2], 10, 32)
	if err != nil {
		return ContentTopic{}, ErrInvalidFormat
	}

	return ContentTopic{
		ApplicationName:    p[1],
		ApplicationVersion: uint(vNum),
		ContentTopicName:   p[3],
		Encoding:           p[4],
	}, nil
}

type PubsubTopic struct {
	Name     string
	Encoding string
}

func (t PubsubTopic) String() string {
	return fmt.Sprintf("/waku/2/%s/%s", t.Name, t.Encoding)
}

func DefaultPubsubTopic() PubsubTopic {
	return NewPubsubTopic("default-waku", "proto")
}

func NewPubsubTopic(name string, encoding string) PubsubTopic {
	return PubsubTopic{
		Name:     name,
		Encoding: encoding,
	}
}

func (t PubsubTopic) Equal(t2 PubsubTopic) bool {
	return t.Name == t2.Name && t.Encoding == t2.Encoding
}

func StringToPubsubTopic(s string) (PubsubTopic, error) {
	p := strings.Split(s, "/")
	if len(p) != 5 || p[0] != "" || p[1] != "waku" || p[2] != "2" || p[3] == "" || p[4] == "" {
		return PubsubTopic{}, ErrInvalidFormat
	}

	return PubsubTopic{
		Name:     p[3],
		Encoding: p[4],
	}, nil
}

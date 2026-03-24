package communities

import "github.com/status-im/status-go/protocol/protobuf"

// NoopDescriptionEncryptor implements DescriptionEncryptor with no-op behavior.
// Intended for tests that operate on unencrypted communities. Kept in a non-test
// file so it can be referenced from test files in other packages (e.g. protocol).
type NoopDescriptionEncryptor struct{}

func (*NoopDescriptionEncryptor) encryptCommunityDescription(*Community, *protobuf.CommunityDescription) (string, []byte, error) {
	return "", nil, nil
}

func (*NoopDescriptionEncryptor) encryptCommunityDescriptionChannel(*Community, string, *protobuf.CommunityDescription) (string, []byte, error) {
	return "", nil, nil
}

func (*NoopDescriptionEncryptor) decryptCommunityDescription(string, []byte) (*DecryptCommunityResponse, error) {
	return &DecryptCommunityResponse{}, nil
}

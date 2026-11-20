package communities

import (
	"fmt"
	"math/rand"

	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/status-im/status-go/internal/protocol/protobuf"
)

// wrapDescriptionPayloads builds a description wrapper that carries every
// payload as its own payload field, the way a publisher can craft one: the
// signature covers only the payload that protobuf decoding keeps, the last.
func wrapDescriptionPayloads(payloads ...[]byte) []byte {
	wrapped := protowire.AppendBytes(protowire.AppendTag(nil, 1, protowire.BytesType), make([]byte, 65))
	for _, payload := range payloads {
		wrapped = protowire.AppendBytes(protowire.AppendTag(wrapped, 2, protowire.BytesType), payload)
	}
	return protowire.AppendVarint(protowire.AppendTag(wrapped, 3, protowire.VarintType),
		uint64(protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION))
}

func (s *PersistenceSuite) marshal(message proto.Message) []byte {
	encoded, err := proto.Marshal(message)
	s.Require().NoError(err)
	return encoded
}

func (s *PersistenceSuite) saveRawDescription(id byte, description []byte) []byte {
	communityID := []byte{0x02, 0x20, id}
	s.Require().NoError(s.db.saveRawCommunityRowWithoutSyncedAt(RawCommunityRow{ID: communityID, Description: description}))
	return communityID
}

func bannerOf(payload string) *protobuf.ChatIdentity {
	return &protobuf.ChatIdentity{Images: map[string]*protobuf.IdentityImage{"banner": {Payload: []byte(payload)}}}
}

// requireReadsMatchDecode checks that the clock and the images read from the
// stored description are the ones of the decoded description.
func (s *PersistenceSuite) requireReadsMatchDecode(communityID []byte, clock uint64, images map[string]string) {
	row, err := s.db.getRawCommunityRow(communityID)
	s.Require().NoError(err)
	decoded, err := decodeWrappedCommunityDescription(row.Description)
	s.Require().NoError(err)
	s.Require().Equal(clock, decoded.Clock, "the decoded description")
	s.Require().Len(decoded.GetIdentity().GetImages(), len(images), "the decoded description")

	readClock, found, err := s.db.CommunityClock(communityID)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(clock, readClock)

	readImages, found, pendingEvents, err := s.db.CommunityImages(communityID)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().False(pendingEvents)
	s.Require().Len(readImages, len(images))
	for name, payload := range images {
		s.Require().Contains(readImages, name)
		s.Require().Equal(payload, string(readImages[name].Payload), name)
	}
}

// A wrapper can carry a forged payload in front of the signed one.
func (s *PersistenceSuite) TestDescriptionReadsIgnoreForgedLeadingPayload() {
	forged := s.marshal(&protobuf.CommunityDescription{Clock: 999999, Identity: bannerOf("FORGED")})
	genuine := s.marshal(&protobuf.CommunityDescription{Clock: 7, Identity: bannerOf("GENUINE")})

	communityID := s.saveRawDescription(1, wrapDescriptionPayloads(forged, genuine))

	s.requireReadsMatchDecode(communityID, 7, map[string]string{"banner": "GENUINE"})
}

// The signed payload can lie far into the stored description.
func (s *PersistenceSuite) TestDescriptionReadsIgnoreLargeForgedLeadingPayload() {
	forged := s.marshal(&protobuf.CommunityDescription{
		Clock:    999999,
		Members:  communityMembers(100),
		Identity: bannerOf("FORGED"),
	})
	genuine := s.marshal(&protobuf.CommunityDescription{Clock: 7, Identity: bannerOf("GENUINE")})

	communityID := s.saveRawDescription(2, wrapDescriptionPayloads(forged, genuine))

	s.requireReadsMatchDecode(communityID, 7, map[string]string{"banner": "GENUINE"})
}

func (s *PersistenceSuite) TestDescriptionReadsOfDuplicateClock() {
	payload := protowire.AppendVarint(protowire.AppendTag(nil, 1, protowire.VarintType), 5)
	payload = append(payload, s.marshal(&protobuf.CommunityDescription{Identity: bannerOf("GENUINE")})...)
	payload = protowire.AppendVarint(protowire.AppendTag(payload, 1, protowire.VarintType), 6)

	communityID := s.saveRawDescription(3, wrapDescriptionPayloads(payload))

	s.requireReadsMatchDecode(communityID, 6, map[string]string{"banner": "GENUINE"})
}

// Protobuf decoding merges a repeated embedded message.
func (s *PersistenceSuite) TestDescriptionReadsOfDuplicateIdentity() {
	payload := protowire.AppendVarint(protowire.AppendTag(nil, 1, protowire.VarintType), 7)
	payload = protowire.AppendBytes(protowire.AppendTag(payload, 5, protowire.BytesType), s.marshal(&protobuf.ChatIdentity{
		Images: map[string]*protobuf.IdentityImage{
			"thumbnail": {Payload: []byte("FIRST-THUMBNAIL")},
			"banner":    {Payload: []byte("FIRST-BANNER")},
		},
	}))
	payload = protowire.AppendBytes(protowire.AppendTag(payload, 5, protowire.BytesType), s.marshal(bannerOf("SECOND-BANNER")))

	communityID := s.saveRawDescription(4, wrapDescriptionPayloads(payload))

	s.requireReadsMatchDecode(communityID, 7, map[string]string{
		"thumbnail": "FIRST-THUMBNAIL",
		"banner":    "SECOND-BANNER",
	})
}

// rawDescriptions numbers the descriptions stored by requireReadsAgreeWithDecode.
var rawDescriptions uint32

// requireReadsAgreeWithDecode stores the description and checks that the
// clock and the images read from it are the ones of its normal decode. A
// description the decode rejects must be rejected by both reads when
// rejected is set.
func (s *PersistenceSuite) requireReadsAgreeWithDecode(description []byte, rejected bool, name string) {
	rawDescriptions++
	communityID := []byte{0x02, 0x30, byte(rawDescriptions >> 16), byte(rawDescriptions >> 8), byte(rawDescriptions)}
	s.Require().NoError(s.db.saveRawCommunityRowWithoutSyncedAt(RawCommunityRow{ID: communityID, Description: description}), name)

	decoded, decodeErr := decodeWrappedCommunityDescription(description)
	clock, found, clockErr := s.db.CommunityClock(communityID)
	images, _, _, imagesErr := s.db.CommunityImages(communityID)

	if decodeErr != nil {
		if rejected {
			s.Require().Error(clockErr, name)
			s.Require().False(found, name)
			s.Require().Error(imagesErr, name)
			s.Require().Empty(images, name)
		}
		return
	}

	s.Require().NoError(clockErr, name)
	s.Require().True(found, name)
	s.Require().Equal(decoded.Clock, clock, name)

	s.Require().NoError(imagesErr, name)
	decodedImages := decoded.GetIdentity().GetImages()
	s.Require().Len(images, len(decodedImages), name)
	for imageName, image := range decodedImages {
		s.Require().Contains(images, imageName, name)
		s.Require().True(proto.Equal(image, images[imageName]), "%s: image %s", name, imageName)
	}
}

func (s *PersistenceSuite) smallWrappedDescription() []byte {
	return wrapDescriptionPayloads(s.marshal(&protobuf.CommunityDescription{
		Clock:   7,
		Members: communityMembers(3),
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status",
			Images: map[string]*protobuf.IdentityImage{
				"thumbnail": {Payload: []byte("THUMBNAIL"), ImageFormat: protobuf.ImageFormat_JPEG},
				"banner":    {Payload: []byte("BANNER")},
			},
		},
		IntroMessage: "welcome",
	}))
}

// A description cut short at any length is read as its normal decode reads
// it: as an error, or as a shorter valid description.
func (s *PersistenceSuite) TestDescriptionReadsOfDescriptionCutAtEveryLength() {
	wrapped := s.smallWrappedDescription()

	for length := 0; length <= len(wrapped); length++ {
		s.requireReadsAgreeWithDecode(wrapped[:length], true, fmt.Sprintf("length %d of %d", length, len(wrapped)))
	}
}

func (s *PersistenceSuite) TestDescriptionReadsOfUnusualDescriptions() {
	identity := s.marshal(bannerOf("BANNER"))
	clock := func(value uint64) []byte {
		return protowire.AppendVarint(protowire.AppendTag(nil, 1, protowire.VarintType), value)
	}
	bytesField := func(number protowire.Number, content []byte) []byte {
		return protowire.AppendBytes(protowire.AppendTag(nil, number, protowire.BytesType), content)
	}
	join := func(parts ...[]byte) []byte {
		var joined []byte
		for _, part := range parts {
			joined = append(joined, part...)
		}
		return joined
	}
	group := protowire.AppendTag(protowire.AppendTag(nil, 20, protowire.StartGroupType), 20, protowire.EndGroupType)

	descriptions := map[string][]byte{
		"empty":                            {},
		"no payload":                       wrapDescriptionPayloads(),
		"empty payload":                    wrapDescriptionPayloads([]byte{}),
		"no clock":                         wrapDescriptionPayloads(bytesField(5, identity)),
		"clock zero":                       wrapDescriptionPayloads(join(clock(0), bytesField(5, identity))),
		"no identity":                      wrapDescriptionPayloads(clock(7)),
		"empty identity":                   wrapDescriptionPayloads(join(clock(7), bytesField(5, nil))),
		"clock after identity":             wrapDescriptionPayloads(join(bytesField(5, identity), clock(7))),
		"clock three times":                wrapDescriptionPayloads(join(clock(5), clock(6), bytesField(5, identity), clock(4))),
		"clock as bytes":                   wrapDescriptionPayloads(join(bytesField(1, []byte("x")), bytesField(5, identity))),
		"clock as bytes then as varint":    wrapDescriptionPayloads(join(bytesField(1, []byte("x")), clock(7))),
		"clock as fixed64":                 wrapDescriptionPayloads(protowire.AppendFixed64(protowire.AppendTag(nil, 1, protowire.Fixed64Type), 9)),
		"identity as varint":               wrapDescriptionPayloads(join(clock(7), protowire.AppendVarint(protowire.AppendTag(nil, 5, protowire.VarintType), 1))),
		"identity as varint and as bytes":  wrapDescriptionPayloads(join(clock(7), protowire.AppendVarint(protowire.AppendTag(nil, 5, protowire.VarintType), 1), bytesField(5, identity))),
		"group in the payload":             wrapDescriptionPayloads(join(clock(7), group, bytesField(5, identity))),
		"unknown fields":                   wrapDescriptionPayloads(join(bytesField(200, []byte("unknown")), clock(7), bytesField(5, identity))),
		"payload as varint":                protowire.AppendVarint(protowire.AppendTag(nil, 2, protowire.VarintType), 1),
		"payload as varint and as bytes":   join(protowire.AppendVarint(protowire.AppendTag(nil, 2, protowire.VarintType), 1), bytesField(2, join(clock(7), bytesField(5, identity)))),
		"group in the wrapper":             join(group, bytesField(2, join(clock(7), bytesField(5, identity)))),
		"payload three times":              wrapDescriptionPayloads(clock(1), join(clock(2), bytesField(5, identity)), clock(3)),
		"payload that is not a message":    wrapDescriptionPayloads([]byte{0xff, 0xff}),
		"field number zero in the payload": wrapDescriptionPayloads(join(clock(7), []byte{0x00, 0x00})),
		"field number above the maximum in the payload": wrapDescriptionPayloads(join(clock(7), bytesField(5, identity), protowire.AppendVarint(protowire.AppendTag(nil, protowire.MaxValidNumber+1, protowire.VarintType), 1))),
		"field number above the maximum in the wrapper": join(wrapDescriptionPayloads(join(clock(7), bytesField(5, identity))), protowire.AppendVarint(protowire.AppendTag(nil, protowire.MaxValidNumber+1, protowire.VarintType), 1)),
		"clock varint too long":                         wrapDescriptionPayloads(append([]byte{0x08}, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01)),
		"clock varint not minimal":                      wrapDescriptionPayloads([]byte{0x08, 0x87, 0x80, 0x00}),
	}

	for name, description := range descriptions {
		s.requireReadsAgreeWithDecode(description, true, name)
	}
}

// The scan does not look inside the fields it passes over, so a description
// broken inside such a field is rejected by its decode and still answered by
// the reads. The image read decodes the identity: it rejects a description
// whose identity is broken, and the clock read does not.
func (s *PersistenceSuite) TestDescriptionImagesOfIdentityThatIsNotAMessage() {
	payload := protowire.AppendVarint(protowire.AppendTag(nil, 1, protowire.VarintType), 7)
	payload = protowire.AppendBytes(protowire.AppendTag(payload, 5, protowire.BytesType), []byte{0xff, 0xff})
	wrapped := wrapDescriptionPayloads(payload)
	_, err := decodeWrappedCommunityDescription(wrapped)
	s.Require().Error(err)

	communityID := s.saveRawDescription(5, wrapped)

	images, _, _, err := s.db.CommunityImages(communityID)
	s.Require().ErrorIs(err, errMalformedDescription)
	s.Require().Empty(images)
}

// Whatever the decode accepts, the reads tell the same.
func (s *PersistenceSuite) TestDescriptionReadsOfMutatedDescriptions() {
	wrapped := s.smallWrappedDescription()
	random := rand.New(rand.NewSource(21528)) // #nosec G404

	for i := 0; i < 3000; i++ {
		mutated := append([]byte{}, wrapped...)
		for flips := 1 + random.Intn(3); flips > 0; flips-- {
			switch position := random.Intn(len(mutated)); random.Intn(3) {
			case 0:
				mutated[position] = byte(random.Intn(256))
			case 1:
				mutated = append(mutated[:position], mutated[position+1:]...)
			default:
				// Repeat a slice of the description at a random place.
				length := random.Intn(len(mutated) - position)
				at := random.Intn(len(mutated))
				repeated := append([]byte{}, mutated[position:position+length]...)
				mutated = append(mutated[:at], append(repeated, mutated[at:]...)...)
			}
		}
		s.requireReadsAgreeWithDecode(mutated, false, fmt.Sprintf("mutation %d: %x", i, mutated))
	}
}

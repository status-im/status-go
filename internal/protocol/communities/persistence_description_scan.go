package communities

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/status-im/status-go/internal/protocol/protobuf"
)

const (
	wrappedDescriptionPayloadField = protowire.Number(2)
	descriptionClockField          = protowire.Number(1)
	descriptionIdentityField       = protowire.Number(5)
)

var errMalformedDescription = errors.New("malformed community description")

// CommunityClock returns the clock of the stored community description
// without building the community. The clock is the one decoding the
// description gives.
func (p *Persistence) CommunityClock(id []byte) (clock uint64, found bool, err error) {
	rows, err := p.db.Query(`SELECT description FROM communities_communities WHERE id = ?`, id)
	if err != nil {
		return 0, false, err
	}
	defer rows.Close()

	if !rows.Next() {
		return 0, false, rows.Err()
	}

	// Valid until the rows are closed: the description is not copied.
	var wrapped sql.RawBytes
	if err := rows.Scan(&wrapped); err != nil {
		return 0, false, err
	}

	if scanned, ok := scanWrappedDescription(wrapped); ok {
		return scanned.clock, true, nil
	}

	description, err := decodeStoredDescription(wrapped)
	if err != nil {
		return 0, false, err
	}
	return description.Clock, true, nil
}

// CommunityImages returns the images of the stored community description
// without building the community. The images are the ones decoding the
// description gives. Community events can replace the identity, so with
// pending events the images are not read and pendingEvents is set.
func (p *Persistence) CommunityImages(id []byte) (images map[string]*protobuf.IdentityImage, found bool, pendingEvents bool, err error) {
	rows, err := p.db.Query(`
		SELECT CASE WHEN ae.id IS NULL THEN c.description END, ae.id IS NOT NULL
		FROM communities_communities c
		LEFT JOIN communities_events ae ON c.id = ae.id
		WHERE c.id = ?`, id)
	if err != nil {
		return nil, false, false, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, false, false, rows.Err()
	}

	// Valid until the rows are closed: the description is not copied.
	var wrapped sql.RawBytes
	if err := rows.Scan(&wrapped, &pendingEvents); err != nil {
		return nil, true, false, err
	}
	if pendingEvents {
		return nil, true, true, nil
	}

	identity := &protobuf.ChatIdentity{}
	scanned, ok := scanWrappedDescription(wrapped)
	if ok {
		ok = proto.Unmarshal(scanned.identity, identity) == nil
	}
	if !ok {
		description, err := decodeStoredDescription(wrapped)
		if err != nil {
			return nil, true, false, err
		}
		identity = description.GetIdentity()
	}

	if identity.GetImages() == nil {
		return map[string]*protobuf.IdentityImage{}, true, false, nil
	}
	return identity.Images, true, false, nil
}

func decodeStoredDescription(wrapped []byte) (*protobuf.CommunityDescription, error) {
	description, err := decodeWrappedCommunityDescription(wrapped)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errMalformedDescription, err)
	}
	return description, nil
}

// scannedDescription holds what scanWrappedDescription reads.
type scannedDescription struct {
	clock uint64
	// identity is the encoded ChatIdentity, empty when the description has
	// none. It points into the scanned bytes.
	identity []byte
}

// scanWrappedDescription reads the clock and the identity of a wrapped
// community description at wire level, walking the whole wrapper and the
// whole payload. It answers only for a description in which decoding leaves
// no choice: the payload, the clock and the identity are each present at
// most once and with their own wire type, the clock is set, and every field
// on the way is well-formed. For anything else ok is false and the
// description has to be decoded: protobuf keeps the last of a repeated
// scalar and merges a repeated message, and the wrapper is stored as it was
// received, so a description can carry more than the copy its signature
// covers.
//
// The scan does not look inside the fields it passes over. A description
// that is broken only inside such a field is rejected by its decode and
// still answered here, for the clock and for the identity alike. No write
// path stores one: each decodes the description first.
func scanWrappedDescription(wrapped []byte) (scanned scannedDescription, ok bool) {
	payload, count, ok := scanSingleField(wrapped, wrappedDescriptionPayloadField, protowire.BytesType)
	if !ok || count != 1 {
		return scannedDescription{}, false
	}
	payloadContent, _ := protowire.ConsumeBytes(payload)

	clock, count, ok := scanSingleField(payloadContent, descriptionClockField, protowire.VarintType)
	if !ok || count != 1 {
		return scannedDescription{}, false
	}
	scanned.clock, _ = protowire.ConsumeVarint(clock)
	if scanned.clock == 0 {
		return scannedDescription{}, false
	}

	identity, count, ok := scanSingleField(payloadContent, descriptionIdentityField, protowire.BytesType)
	if !ok {
		return scannedDescription{}, false
	}
	if count == 1 {
		scanned.identity, _ = protowire.ConsumeBytes(identity)
	}

	return scanned, true
}

// scanSingleField walks every field of message and returns the encoded value
// of the field with the given number, and how many times the field occurs.
// ok is false when the field occurs more than once or with another wire
// type, and when message holds anything but well-formed varint, fixed and
// length-delimited fields with a valid field number.
func scanSingleField(message []byte, field protowire.Number, fieldType protowire.Type) (value []byte, count int, ok bool) {
	for len(message) > 0 {
		number, wireType, n := protowire.ConsumeTag(message)
		if n < 0 || number > protowire.MaxValidNumber {
			return nil, 0, false
		}
		message = message[n:]

		switch wireType {
		case protowire.VarintType, protowire.Fixed32Type, protowire.Fixed64Type, protowire.BytesType:
		default:
			return nil, 0, false
		}

		n = protowire.ConsumeFieldValue(number, wireType, message)
		if n < 0 {
			return nil, 0, false
		}
		if number == field {
			if wireType != fieldType || count > 0 {
				return nil, 0, false
			}
			value = message[:n]
			count++
		}
		message = message[n:]
	}

	return value, count, true
}

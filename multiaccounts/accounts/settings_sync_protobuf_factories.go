package accounts

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/status-im/status-go/protocol/protobuf"
)

func currencyProtobufFactory(value interface{}, clock time.Time) SyncSettingProtobuf {
	pb := new(protobuf.SyncSettingCurrency)
	pb.Value = value.(string)
	pb.Clock = timestamppb.New(clock)
	return pb
}

func gifRecentsProtobufFactory(value interface{}, clock time.Time) SyncSettingProtobuf {
	pb := new(protobuf.SyncSettingGifRecents)
	pb.Value = value.([]byte)
	pb.Clock = timestamppb.New(clock)
	return pb
}

func gifFavouritesProtobufFactory(value interface{}, clock time.Time) SyncSettingProtobuf {
	pb := new(protobuf.SyncSettingGifFavorites)
	pb.Value = value.([]byte)
	pb.Clock = timestamppb.New(clock)
	return pb
}

func messagesFromContactsOnlyProtobufFactory(value interface{}, clock time.Time) SyncSettingProtobuf {
	pb := new(protobuf.SyncSettingMessagesFromContactsOnly)
	pb.Value = value.(bool)
	pb.Clock = timestamppb.New(clock)
	return pb
}

func preferredNameProtobufFactory(value interface{}, clock time.Time) SyncSettingProtobuf {
	pb := new(protobuf.SyncSettingPreferredName)
	pb.Value = value.(string)
	pb.Clock = timestamppb.New(clock)
	return pb
}

func previewPrivacyProtobufFactory(value interface{}, clock time.Time) SyncSettingProtobuf {
	pb := new(protobuf.SyncSettingPreviewPrivacy)
	pb.Value = value.(bool)
	pb.Clock = timestamppb.New(clock)
	return pb
}

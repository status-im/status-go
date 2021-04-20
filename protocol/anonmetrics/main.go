package anonmetrics

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/status-im/status-go/appmetrics"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

// adaptProtoToModel is an adaptor helper function to convert a protobuf.AnonymousMetric into a appmetrics.AppMetric
func adaptProtoToModel(pbAnonMetric *protobuf.AnonymousMetric) *appmetrics.AppMetric {
	return &appmetrics.AppMetric{
		Event:      appmetrics.AppMetricEventType(pbAnonMetric.Event),
		Value:      pbAnonMetric.Value,
		AppVersion: pbAnonMetric.AppVersion,
		OS:         pbAnonMetric.Os,
	}
}

// adaptModelToProto is an adaptor helper function to convert a appmetrics.AppMetric into a protobuf.AnonymousMetric
func adaptModelToProto(modelAnonMetric *appmetrics.AppMetric, sendID *ecdsa.PublicKey) *protobuf.AnonymousMetric {
	id := generateProtoID(modelAnonMetric, sendID)

	return &protobuf.AnonymousMetric{
		Id:         id,
		Event:      string(modelAnonMetric.Event),
		Value:      modelAnonMetric.Value,
		AppVersion: modelAnonMetric.AppVersion,
		Os:         modelAnonMetric.OS,
		SessionId:  modelAnonMetric.SessionID,
		CreatedAt:  modelAnonMetric.CreatedAt,
	}
}

func adaptModelsToProtoBatch(modelAnonMetrics []*appmetrics.AppMetric, sendID *ecdsa.PublicKey) *protobuf.AnonymousMetricBatch {
	amb := new(protobuf.AnonymousMetricBatch)

	for _, m := range modelAnonMetrics {
		amb.Metrics = append(amb.Metrics, adaptModelToProto(m, sendID))
	}

	return amb
}

func adaptProtoBatchToModels(protoBatch *protobuf.AnonymousMetricBatch) []*appmetrics.AppMetric {
	var ams []*appmetrics.AppMetric

	for _, pm := range protoBatch.Metrics {
		ams = append(ams, adaptProtoToModel(pm))
	}

	return ams
}

func generateProtoID(modelAnonMetric *appmetrics.AppMetric, sendID *ecdsa.PublicKey) string {
	return types.EncodeHex(crypto.Keccak256([]byte(fmt.Sprintf(
		"%s%s%s%s%s%d",
		types.EncodeHex(crypto.FromECDSAPub(sendID)),
		modelAnonMetric.CreatedAt,
		modelAnonMetric.SessionID,
		modelAnonMetric.Event,
		modelAnonMetric.Value,
		modelAnonMetric.ID))))
}

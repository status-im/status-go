package anonmetrics

import (
	"github.com/status-im/status-go/appmetrics"
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
func adaptModelToProto(modelAnonMetric *appmetrics.AppMetric) *protobuf.AnonymousMetric {
	return &protobuf.AnonymousMetric{
		Event:                string(modelAnonMetric.Event),
		Value:                modelAnonMetric.Value,
		AppVersion:           modelAnonMetric.AppVersion,
		Os:                   modelAnonMetric.OS,
	}
}

func adaptModelsToProtoBatch(modelAnonMetrics []*appmetrics.AppMetric) *protobuf.AnonymousMetricBatch {
	amb :=  new(protobuf.AnonymousMetricBatch)

	for _, m := range modelAnonMetrics {
		amb.Metrics = append(amb.Metrics, adaptModelToProto(m))
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

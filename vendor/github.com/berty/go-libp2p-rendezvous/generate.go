//go:generate protoc --proto_path=pb/ --gofast_opt="Mrendezvous.proto=.;rendezvous_pb" --gofast_out=./pb ./pb/rendezvous.proto
package rendezvous

// +build nimbus

package node

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/syndtr/goleveldb/leveldb"

	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/params"
	nimbussvc "github.com/status-im/status-go/services/nimbus"
	"github.com/status-im/status-go/services/nodebridge"
	"github.com/status-im/status-go/services/shhext"
	"github.com/status-im/status-go/services/status"
	"github.com/status-im/status-go/timesource"
)

// Errors related to node and services creation.
var (
	// ErrNodeMakeFailureFormat                      = "error creating p2p node: %s"
	ErrWhisperServiceRegistrationFailure = errors.New("failed to register the Whisper service")
	// ErrLightEthRegistrationFailure                = errors.New("failed to register the LES service")
	ErrLightEthRegistrationFailureUpstreamEnabled = errors.New("failed to register the LES service, upstream is also configured")
	// ErrPersonalServiceRegistrationFailure         = errors.New("failed to register the personal api service")
	ErrStatusServiceRegistrationFailure = errors.New("failed to register the Status service")
	// ErrPeerServiceRegistrationFailure             = errors.New("failed to register the Peer service")
	// ErrIncentivisationServiceRegistrationFailure  = errors.New("failed to register the Incentivisation service")
)

func (n *NimbusStatusNode) activateServices(config *params.NodeConfig, db *leveldb.DB) error {
	// start Ethereum service if we are not expected to use an upstream server
	if !config.UpstreamConfig.Enabled {
	} else {
		if config.LightEthConfig.Enabled {
			return ErrLightEthRegistrationFailureUpstreamEnabled
		}

		n.log.Info("LES protocol is disabled")

		// `personal_sign` and `personal_ecRecover` methods are important to
		// keep DApps working.
		// Usually, they are provided by an ETH or a LES service, but when using
		// upstream, we don't start any of these, so we need to start our own
		// implementation.
		// if err := n.activatePersonalService(accs, config); err != nil {
		// 	return fmt.Errorf("%v: %v", ErrPersonalServiceRegistrationFailure, err)
		// }
	}

	if err := n.activateNodeServices(config, db); err != nil {
		return err
	}

	return nil
}

func (n *NimbusStatusNode) activateNodeServices(config *params.NodeConfig, db *leveldb.DB) error {
	// start Whisper service.
	if err := n.activateShhService(config, db); err != nil {
		return fmt.Errorf("%v: %v", ErrWhisperServiceRegistrationFailure, err)
	}

	// // start Waku service
	// if err := activateWakuService(stack, config, db); err != nil {
	// 	return fmt.Errorf("%v: %v", ErrWakuServiceRegistrationFailure, err)
	// }

	// start incentivisation service
	// if err := n.activateIncentivisationService(config); err != nil {
	// 	return fmt.Errorf("%v: %v", ErrIncentivisationServiceRegistrationFailure, err)
	// }

	// start status service.
	if err := n.activateStatusService(config); err != nil {
		return fmt.Errorf("%v: %v", ErrStatusServiceRegistrationFailure, err)
	}

	// start peer service
	// if err := activatePeerService(n); err != nil {
	// 	return fmt.Errorf("%v: %v", ErrPeerServiceRegistrationFailure, err)
	// }
	return nil
}

// // activateLightEthService configures and registers the eth.Ethereum service with a given node.
// func activateLightEthService(stack *node.Node, accs *accounts.Manager, config *params.NodeConfig) error {
// 	if !config.LightEthConfig.Enabled {
// 		logger.Info("LES protocol is disabled")
// 		return nil
// 	}

// 	genesis, err := calculateGenesis(config.NetworkID)
// 	if err != nil {
// 		return err
// 	}

// 	ethConf := eth.DefaultConfig
// 	ethConf.Genesis = genesis
// 	ethConf.SyncMode = downloader.LightSync
// 	ethConf.NetworkId = config.NetworkID
// 	ethConf.DatabaseCache = config.LightEthConfig.DatabaseCache
// 	return stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
// 		// NOTE(dshulyak) here we set our instance of the accounts manager.
// 		// without sharing same instance selected account won't be visible for personal_* methods.
// 		nctx := &node.ServiceContext{}
// 		*nctx = *ctx
// 		nctx.AccountManager = accs
// 		return les.New(nctx, &ethConf)
// 	})
// }

// func activatePersonalService(stack *node.Node, accs *accounts.Manager, config *params.NodeConfig) error {
// 	return stack.Register(func(*node.ServiceContext) (node.Service, error) {
// 		svc := personal.New(accs)
// 		return svc, nil
// 	})
// }

// func (n *NimbusStatusNode) activatePersonalService(accs *accounts.Manager, config *params.NodeConfig) error {
// 	return n.Register(func(*nimbussvc.ServiceContext) (nimbussvc.Service, error) {
// 		svc := personal.New(accs)
// 		return svc, nil
// 	})
// }

func (n *NimbusStatusNode) activateStatusService(config *params.NodeConfig) error {
	if !config.EnableStatusService {
		n.log.Info("Status service api is disabled")
		return nil
	}

	return n.Register(func(ctx *nimbussvc.ServiceContext) (nimbussvc.Service, error) {
		var service *nodebridge.WhisperService
		if err := ctx.Service(&service); err != nil {
			return nil, err
		}
		svc := status.New(service.Whisper)
		return svc, nil
	})
}

// func (n *NimbusStatusNode) activatePeerService() error {
// 	return n.Register(func(ctx *nimbussvc.ServiceContext) (nimbussvc.Service, error) {
// 		svc := peer.New()
// 		return svc, nil
// 	})
// }

// func registerWhisperMailServer(whisperService *whisper.Whisper, config *params.WhisperConfig) (err error) {
// 	var mailServer mailserver.WhisperMailServer
// 	whisperService.RegisterMailServer(&mailServer)

// 	return mailServer.Init(whisperService, config)
// }

// func registerWakuMailServer(wakuService *waku.Waku, config *params.WakuConfig) (err error) {
// 	var mailServer mailserver.WakuMailServer
// 	wakuService.RegisterMailServer(&mailServer)

// 	return mailServer.Init(wakuService, config)
// }

// activateShhService configures Whisper and adds it to the given node.
func (n *NimbusStatusNode) activateShhService(config *params.NodeConfig, db *leveldb.DB) (err error) {
	if !config.WhisperConfig.Enabled {
		n.log.Info("SHH protocol is disabled")
		return nil
	}
	if config.EnableNTPSync {
		if err = n.Register(func(*nimbussvc.ServiceContext) (nimbussvc.Service, error) {
			return timesource.Default(), nil
		}); err != nil {
			return
		}
	}

	// err = n.Register(func(ctx *nimbussvc.ServiceContext) (nimbussvc.Service, error) {
	// 	return n.createShhService(ctx, &config.WhisperConfig, &config.ClusterConfig)
	// })
	// if err != nil {
	// 	return
	// }

	// Register eth-node node bridge
	err = n.Register(func(ctx *nimbussvc.ServiceContext) (nimbussvc.Service, error) {
		return &nodebridge.NodeService{Node: n.node}, nil
	})
	if err != nil {
		return
	}

	// Register Whisper eth-node bridge
	err = n.Register(func(ctx *nimbussvc.ServiceContext) (nimbussvc.Service, error) {
		n.log.Info("Creating WhisperService")

		var ethnode *nodebridge.NodeService
		if err := ctx.Service(&ethnode); err != nil {
			return nil, err
		}

		w, err := ethnode.Node.GetWhisper(ctx)
		if err != nil {
			n.log.Error("GetWhisper returned error", "err", err)
			return nil, err
		}

		return &nodebridge.WhisperService{Whisper: w}, nil
	})
	if err != nil {
		return
	}

	// TODO(dshulyak) add a config option to enable it by default, but disable if app is started from statusd
	return n.Register(func(ctx *nimbussvc.ServiceContext) (nimbussvc.Service, error) {
		var ethnode *nodebridge.NodeService
		if err := ctx.Service(&ethnode); err != nil {
			return nil, err
		}
		return shhext.NewNimbus(ethnode.Node, ctx, "shhext", db, config.ShhextConfig), nil
	})
}

// activateWakuService configures Waku and adds it to the given node.
func (n *NimbusStatusNode) activateWakuService(config *params.NodeConfig, db *leveldb.DB) (err error) {
	if !config.WakuConfig.Enabled {
		n.log.Info("Waku protocol is disabled")
		return nil
	}

	panic("not implemented")
	// err = n.Register(func(ctx *nimbussvc.ServiceContext) (nimbussvc.Service, error) {
	// 	return createWakuService(ctx, &config.WakuConfig, &config.ClusterConfig)
	// })
	// if err != nil {
	// 	return
	// }

	// // TODO(dshulyak) add a config option to enable it by default, but disable if app is started from statusd
	// return n.Register(func(ctx *nimbussvc.ServiceContext) (nimbussvc.Service, error) {
	// 	var ethnode *nodebridge.NodeService
	// 	if err := ctx.Service(&ethnode); err != nil {
	// 		return nil, err
	// 	}
	// 	return shhext.New(ethnode.Node, ctx, "wakuext", shhext.EnvelopeSignalHandler{}, db, config.ShhextConfig), nil
	// })
}

// Register injects a new service into the node's stack. The service created by
// the passed constructor must be unique in its type with regard to sibling ones.
func (n *NimbusStatusNode) Register(constructor nimbussvc.ServiceConstructor) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.isRunning() {
		return ErrNodeRunning
	}
	n.serviceFuncs = append(n.serviceFuncs, constructor)
	return nil
}

func (n *NimbusStatusNode) startServices() error {
	services := make(map[reflect.Type]nimbussvc.Service)
	for _, constructor := range n.serviceFuncs {
		// Create a new context for the particular service
		ctxServices := make(map[reflect.Type]nimbussvc.Service)
		for kind, s := range services { // copy needed for threaded access
			ctxServices[kind] = s
		}
		ctx := nimbussvc.NewServiceContext(n.config, ctxServices)
		//EventMux:       n.eventmux,
		//AccountManager: n.accman,
		// Construct and save the service
		service, err := constructor(ctx)
		if err != nil {
			n.log.Info("Service constructor returned error", "err", err)
			return err
		}
		kind := reflect.TypeOf(service)
		if _, exists := services[kind]; exists {
			return &nimbussvc.DuplicateServiceError{Kind: kind}
		}
		services[kind] = service
	}
	// Start each of the services
	var started []reflect.Type
	for kind, service := range services {
		// Start the next service, stopping all previous upon failure
		if err := service.StartService(); err != nil {
			for _, kind := range started {
				services[kind].Stop()
			}

			return err
		}
		// Mark the service started for potential cleanup
		started = append(started, kind)
	}
	// Lastly start the configured RPC interfaces
	if err := n.startRPC(services); err != nil {
		for _, service := range services {
			service.Stop()
		}
		return err
	}
	// Finish initializing the startup
	n.services = services

	return nil
}

// startRPC is a helper method to start all the various RPC endpoint during node
// startup. It's not meant to be called at any time afterwards as it makes certain
// assumptions about the state of the node.
func (n *NimbusStatusNode) startRPC(services map[reflect.Type]nimbussvc.Service) error {
	// Gather all the possible APIs to surface
	apis := []gethrpc.API{}
	for _, service := range services {
		apis = append(apis, service.APIs()...)
	}

	// Start the various API endpoints, terminating all in case of errors
	if err := n.startInProc(apis); err != nil {
		return err
	}
	if err := n.startPublicInProc(apis, n.config.FormatAPIModules()); err != nil {
		n.stopInProc()
		return err
	}
	// All API endpoints started successfully
	n.rpcAPIs = apis
	return nil
}

// startInProc initializes an in-process RPC endpoint.
func (n *NimbusStatusNode) startInProc(apis []gethrpc.API) error {
	n.log.Debug("startInProc", "apis", apis)
	// Register all the APIs exposed by the services
	handler := gethrpc.NewServer()
	for _, api := range apis {
		n.log.Debug("Registering InProc", "namespace", api.Namespace)
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			return err
		}
		n.log.Debug("InProc registered", "namespace", api.Namespace)
	}
	n.inprocHandler = handler
	return nil
}

// stopInProc terminates the in-process RPC endpoint.
func (n *NimbusStatusNode) stopInProc() {
	if n.inprocHandler != nil {
		n.inprocHandler.Stop()
		n.inprocHandler = nil
	}
}

// startPublicInProc initializes an in-process RPC endpoint for public APIs.
func (n *NimbusStatusNode) startPublicInProc(apis []gethrpc.API, modules []string) error {
	n.log.Debug("startPublicInProc", "apis", apis, "modules", modules)
	// Generate the whitelist based on the allowed modules
	whitelist := make(map[string]bool)
	for _, module := range modules {
		whitelist[module] = true
	}

	// Register all the public APIs exposed by the services
	handler := gethrpc.NewServer()
	for _, api := range apis {
		if whitelist[api.Namespace] || (len(whitelist) == 0 && api.Public) {
			n.log.Debug("Registering InProc public", "service", api.Service, "namespace", api.Namespace)
			if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
				return err
			}
			n.log.Debug("InProc public registered", "service", api.Service, "namespace", api.Namespace)
		}
	}
	n.inprocPublicHandler = handler
	return nil
}

// stopPublicInProc terminates the in-process RPC endpoint for public APIs.
func (n *NimbusStatusNode) stopPublicInProc() {
	if n.inprocPublicHandler != nil {
		n.inprocPublicHandler.Stop()
		n.inprocPublicHandler = nil
	}
}

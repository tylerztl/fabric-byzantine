package server

import (
	"errors"
	"fabric-byzantine/server/helpers"
	"fmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/event"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/deliverclient/seek"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	contextAPI "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

var logger = helpers.GetLogger()
var appConf = helpers.GetAppConf().Conf

type OrgInstance struct {
	Config      *helpers.OrgInfo
	AdminClient *resmgmt.Client
	MspClient   *mspclient.Client
	Peers       []fab.Peer
}

type OrdererInstance struct {
	Config      *helpers.OrderderInfo
	AdminClient *resmgmt.Client
}

type FabSdkProvider struct {
	Sdk      *fabsdk.FabricSDK
	Orgs     []*OrgInstance
	Orderers []*OrdererInstance
}

func loadOrgPeers(org string, ctxProvider contextAPI.ClientProvider) ([]fab.Peer, error) {
	ctx, err := ctxProvider()
	if err != nil {
		return nil, err
	}

	orgPeers, ok := ctx.EndpointConfig().PeersConfig(org)
	if !ok {
		return nil, errors.New(fmt.Sprintf("Failed to load org peers for %s", org))
	}
	peers := make([]fab.Peer, len(orgPeers))
	for i, val := range orgPeers {
		if peer, err := ctx.InfraProvider().CreatePeerFromConfig(&fab.NetworkPeer{PeerConfig: val}); err != nil {
			return nil, err
		} else {
			peers[i] = peer
		}

	}
	return peers, nil
}

func NewFabSdkProvider() (*FabSdkProvider, error) {
	configOpt := config.FromFile(helpers.GetConfigPath("config.yaml"))
	sdk, err := fabsdk.New(configOpt)
	if err != nil {
		logger.Error("Failed to create new SDK: %s", err)
		return nil, err
	}

	provider := &FabSdkProvider{
		Sdk:      sdk,
		Orgs:     make([]*OrgInstance, len(appConf.OrgInfo)),
		Orderers: make([]*OrdererInstance, len(appConf.OrderderInfo)),
	}
	for i, org := range appConf.OrgInfo {
		//clientContext allows creation of transactions using the supplied identity as the credential.
		adminContext := sdk.Context(fabsdk.WithUser(org.Admin), fabsdk.WithOrg(org.Name))

		mspClient, err := mspclient.New(sdk.Context(), mspclient.WithOrg(org.Name))
		if err != nil {
			logger.Error("Failed to create mspClient for %s, err: %v", org.Name, err)
			return nil, err
		}
		// Resource management client is responsible for managing channels (create/update channel)
		// Supply user that has privileges to create channel (in this case orderer admin)
		adminClient, err := resmgmt.New(adminContext)
		if err != nil {
			logger.Error("Failed to new resource management client: %s", err)
			return nil, err
		}

		orgPeers, err := loadOrgPeers(org.Name, adminContext)
		if err != nil {
			logger.Error("Failed to load peers for %s, err: %v", org.Name, err)
			return nil, err
		}

		provider.Orgs[i] = &OrgInstance{org, adminClient, mspClient, orgPeers}
	}

	if len(provider.Orgs) == 0 {
		logger.Error("Not provider org config in conf/app.yaml", err)
		return nil, errors.New("not provider org config")
	}

	for i, orderer := range appConf.OrderderInfo {
		//clientContext allows creation of transactions using the supplied identity as the credential.
		adminContext := sdk.Context(fabsdk.WithUser(orderer.Admin), fabsdk.WithOrg(orderer.Name))

		// Resource management client is responsible for managing channels (create/update channel)
		// Supply user that has privileges to create channel (in this case orderer admin)
		adminClient, err := resmgmt.New(adminContext)
		if err != nil {
			logger.Error("Failed to new resource management client: %s", err)
			return nil, err
		}
		provider.Orderers[i] = &OrdererInstance{orderer, adminClient}
	}

	return provider, nil
}

func (f *FabSdkProvider) InvokeCC(channelID, ccID, function string, args [][]byte) ([]byte, helpers.TransactionID, error) {
	//ledger.WithTargets(orgTestPeer0, orgTestPeer1)
	orgInstance := f.Orgs[0]
	//prepare context
	userContext := f.Sdk.ChannelContext(channelID, fabsdk.WithUser(orgInstance.Config.User), fabsdk.WithOrg(orgInstance.Config.Name))
	//get channel client
	chClient, err := channel.New(userContext)
	if err != nil {
		logger.Error("Failed to create new channel client: %v", err)
		return nil, "", fmt.Errorf("Failed to create new channel client:  %s", orgInstance.Config.Name)
	}
	// Synchronous transaction
	response, err := chClient.Execute(
		channel.Request{
			ChaincodeID: ccID,
			Fcn:         function,
			Args:        args,
		},
		channel.WithRetry(retry.DefaultChannelOpts))
	if err != nil {
		logger.Error("Failed InvokeCC: %s", err)
		return nil, "", err
	}
	logger.Debug("Successfully invoke chaincode  ccName[%s] func[%v] txId[%v] payload[%v]",
		ccID, function, response.TransactionID, response.Payload)
	return response.Payload, helpers.TransactionID(response.TransactionID), nil
}

func (f *FabSdkProvider) QueryCC(channelID, ccID, function string, args [][]byte) ([]byte, error) {
	orgInstance := f.Orgs[0]

	//prepare context
	userContext := f.Sdk.ChannelContext(channelID, fabsdk.WithUser(orgInstance.Config.User), fabsdk.WithOrg(orgInstance.Config.Name))
	//get channel client
	chClient, err := channel.New(userContext)
	if err != nil {
		logger.Error("Failed to create new channel client: %v", err)
		return nil, fmt.Errorf("Failed to create new channel client:  %s", orgInstance.Config.Name)
	}

	response, err := chClient.Query(channel.Request{ChaincodeID: ccID, Fcn: function, Args: args},
		channel.WithRetry(retry.DefaultChannelOpts))
	if err != nil {
		logger.Error("Failed QueryCC: %s", err)
		return nil, err
	}

	logger.Debug("Successfully query chaincode  ccName[%s] func[%v] payload[%v]",
		ccID, function, response.Payload)
	return response.Payload, nil
}

func (f *FabSdkProvider) BlockListener(channelID string) {
	orgInstance := f.Orgs[0]
	//prepare context
	userContext := f.Sdk.ChannelContext(channelID, fabsdk.WithUser(orgInstance.Config.User), fabsdk.WithOrg(orgInstance.Config.Name))
	// create event client with block events
	eventClient, err := event.New(userContext, event.WithBlockEvents(), event.WithSeekType(seek.Newest))
	if err != nil {
		panic(fmt.Sprintf("Failed to create new events client with block events: %s", err))
	}
	registerBlockEvent(eventClient)
}

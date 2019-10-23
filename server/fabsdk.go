package server

import (
	"encoding/json"
	"errors"
	"fabric-byzantine/server/helpers"
	"fabric-byzantine/server/mysql"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/event"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	contextAPI "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/deliverclient/seek"
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

func (f *FabSdkProvider) InvokeCC(peer string, peerType int, index int, channelID, ccID, function string, args [][]byte) ([]byte, helpers.TransactionID, error) {
	//ledger.WithTargets(orgTestPeer0, orgTestPeer1)
	orgInstance := f.Orgs[index]
	//prepare context
	userContext := f.Sdk.ChannelContext(channelID, fabsdk.WithUser(orgInstance.Config.User), fabsdk.WithOrg(orgInstance.Config.Name))
	//get channel client
	chClient, err := channel.New(userContext)
	if err != nil {
		logger.Error("Failed to create new channel client: %v", err)
		return nil, "", fmt.Errorf("Failed to create new channel client:  %s", orgInstance.Config.Name)
	}

	result, _ := f.QueryCC(0, "mychannel1", "token",
		"getPeers", [][]byte{[]byte("fab")})
	peers := make(map[string]bool)
	json.Unmarshal(result, &peers)

	nPeers := []int{} // normal peer
	bPeers := []int{} // byzantine peer
	for k, v := range peers {
		index, _ := strconv.Atoi(k[9:10])
		if index == 1 {
			if k, err := strconv.Atoi(k[9:11]); err == nil {
				index = k
			}
		}
		index--
		if v {
			nPeers = append(nPeers, index)
		} else {
			bPeers = append(bPeers, index)
		}
	}
	var targets []fab.Peer
	if len(nPeers) >= 7 {
		targets = make([]fab.Peer, len(nPeers))
		for k, v := range nPeers {
			targets[k] = f.Orgs[v].Peers[0]
		}
	} else if len(bPeers) >= 7 {
		targets = make([]fab.Peer, len(bPeers))
		for k, v := range bPeers {
			targets[k] = f.Orgs[v].Peers[0]
		}
	} else {
		targets = make([]fab.Peer, len(f.Orgs))
		for k, v := range f.Orgs {
			targets[k] = v.Peers[0]
		}
	}
	fmt.Println("targets:", targets)

	// Synchronous transaction
	response, err := chClient.Execute(
		channel.Request{
			ChaincodeID: ccID,
			Fcn:         function,
			Args:        args,
		},
		channel.WithTargets(targets...))
	//channel.WithRetry(retry.DefaultChannelOpts))
	if err != nil {
		logger.Error("[%s] failed invokeCC: %s", peer, err)
		TxChans.Range(func(key, value interface{}) bool {
			datas, _ := json.Marshal(&TransactionInfo{
				Status:   500,
				TxId:     string(response.TransactionID),
				DateTime: time.Now(),
			})
			value.(chan []byte) <- datas
			return true
		})

		return nil, helpers.TransactionID(response.TransactionID), err
	}

	bFlage := false
	timeout := 0
	for {
		if bFlage || timeout > 10 {
			break
		}
		select {
		case <-time.After(time.Millisecond * time.Duration(500)):
			val, _ := mysql.QueryTransaction(string(response.TransactionID))
			count, _ := strconv.Atoi(string(val))
			bFlage = count == 1
			timeout++
		}
	}

	if bFlage {
		if err := mysql.UpdateTransaction(peer, string(response.TransactionID), peerType); err != nil {
			logger.Error("UpdateTransaction err:%s", err.Error())
		}
	}

	logger.Debug("Successfully invoke chaincode  ccName[%s] func[%v] txId[%v]",
		ccID, function, response.TransactionID)
	return response.Payload, helpers.TransactionID(response.TransactionID), nil
}

func (f *FabSdkProvider) QueryCC(index int, channelID, ccID, function string, args [][]byte) ([]byte, error) {
	orgInstance := f.Orgs[index]

	//prepare context
	userContext := f.Sdk.ChannelContext(channelID, fabsdk.WithUser(orgInstance.Config.User), fabsdk.WithOrg(orgInstance.Config.Name))
	//get channel client
	chClient, err := channel.New(userContext)
	if err != nil {
		logger.Error("Failed to create new channel client: %v", err)
		return nil, fmt.Errorf("Failed to create new channel client:  %s", orgInstance.Config.Name)
	}

	response, err := chClient.Query(channel.Request{ChaincodeID: ccID, Fcn: function, Args: args}, channel.WithTargets(orgInstance.Peers[0]),
		channel.WithRetry(retry.DefaultChannelOpts))
	if err != nil {
		logger.Error("Failed QueryCC: %s", err)
		return nil, err
	}

	logger.Debug("Successfully query chaincode  ccName[%s] func[%v] payload[%s]",
		ccID, function, string(response.Payload))
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
	ledgerClient, err := ledger.New(userContext)
	if err != nil {
		panic(fmt.Sprintf("Failed to create new ledger client: %s", err))
	}
	go syncBlock(ledgerClient, orgInstance.Peers[0])
	registerBlockEvent(eventClient)
}

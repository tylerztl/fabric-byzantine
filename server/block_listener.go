package server

import (
	"encoding/hex"
	"encoding/json"
	"fabric-byzantine/server/mysql"
	"fabric-byzantine/server/protoutil"
	"fmt"
	"sync"
	"time"

	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/event"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
)

var BlockChans = new(sync.Map)
var BlockNumberChans = new(sync.Map)
var TxChans = new(sync.Map)
var TxNumberChans = new(sync.Map)
var TxList = new(sync.Map)

type PeerTx struct {
	PeerName string `json:"name"`
	PeerType int    `json:"type"`
}

type BlockInfo struct {
	Number    uint64    `json:"number"`
	TxCount   int       `json:"tx_count"`
	BlockHash string    `json:"block_hash"`
	DateTime  time.Time `json:"datetime"`
	Alice     string    `json:"alice_balance"`
	Bob       string    `json:"bob_balance"`
	Type      int       `json:"type"`
}

type TransactionInfo struct {
	Status   int       `json:"status"`
	TxId     string    `json:"tx_id"`
	DateTime time.Time `json:"datetime"`
	Peer     string    `json:"peer"`
	TxType   int       `json:"tx_type"`
}

func registerBlockEvent(eventClient *event.Client) {
	reg, eventch, err := eventClient.RegisterBlockEvent()
	if err != nil {
		logger.Error("Error registering for block events: %s", err)
	}
	defer eventClient.Unregister(reg)

	flag := true
	for {
		select {
		case e, ok := <-eventch:
			if !ok {
				logger.Error("unexpected closed channel while waiting for block event")
			}
			logger.Info("Received block event: %#v", e)
			if e.Block == nil {
				logger.Error("Expecting block in block event but got nil")
			}
			if flag {
				flag = false
			} else {
				go updateBlock(e.Block)
			}
		}
	}
}

func updateBlock(block *cb.Block) {
	if block.Header.Number == 0 {
		return
	}

	begin, err := mysql.GetDB().Begin()
	if err != nil {
		panic(err.Error())
	}

	peerName := "peer0.org1.example.com"
	peerType := 0
	txLen := len(block.Data.Data)
	var txTime time.Time
	for i, envBytes := range block.Data.Data {
		envelope, err := protoutil.GetEnvelopeFromBlock(envBytes)
		if err != nil {
			logger.Error("Error GetEnvelopeFromBlock:", err)
			break
		}
		payload, err := protoutil.UnmarshalPayload(envelope.Payload)
		if err != nil {
			logger.Error("error extracting payload from block: %s", err)
			continue
		}
		channelHeader, _ := protoutil.UnmarshalChannelHeader(payload.Header.ChannelHeader)
		txTimestamp := channelHeader.Timestamp
		txTime = time.Unix(txTimestamp.GetSeconds(), int64(txTimestamp.GetNanos()))

		validationCode := int(block.Metadata.Metadata[cb.BlockMetadataIndex_TRANSACTIONS_FILTER][i])

		logger.Debug("Seek block number:%d", block.Header.Number)

		val, ok := TxList.Load(channelHeader.TxId)
		if !ok {
			logger.Error("UpdateBlock not found txId: %s", channelHeader.TxId)
		} else {
			peerName = val.(PeerTx).PeerName
			peerType = val.(PeerTx).PeerType
			TxList.Delete(channelHeader.TxId)
		}

		if peerType > 0 && peerType <= 3 {
			validationCode = 1
		}

		TxChans.Range(func(key, value interface{}) bool {
			datas, _ := json.Marshal(&TransactionInfo{
				Status:   validationCode,
				TxId:     channelHeader.TxId,
				DateTime: txTime,
				Peer:     peerName,
				TxType:   peerType,
			})
			value.(chan []byte) <- datas
			return true
		})

		_, err = begin.Stmt(mysql.GetStmtTx()).Exec(block.Header.Number*uint64(appConf.TxNumPerBlock)+uint64(i), block.Header.Number,
			channelHeader.TxId, peerName, peerType, validationCode, txTime)
		if err != nil {
			logger.Warn(err.Error()) // proper error handling instead of panic in your app
		}

		//_, err = stmTx.Exec(block.Header.Number*uint64(AppConf.TxNumPerBlock)+uint64(i), channelHeader.TxId, validationCode, txTime)
		//if err != nil {
		//	Logger.Warn(err.Error()) // proper error handling instead of panic in your app
		//}'
	}

	aliceBalance, _ := GetSdkProvider().QueryCC(0, "mychannel1", "token",
		"balance", [][]byte{[]byte("fab"), []byte("alice")})
	bobBalance, _ := GetSdkProvider().QueryCC(0, "mychannel1", "token",
		"balance", [][]byte{[]byte("fab"), []byte("bob")})

	if peerType > 0 {
		if peerType <= 3 {
			peerType = 0
		} else {
			peerType = 1
		}
	}

	_, err = begin.Stmt(mysql.GetStmtBlock()).Exec(block.Header.Number, hex.EncodeToString(block.Header.DataHash), txLen, txTime, aliceBalance, bobBalance, peerType)
	if err != nil {
		logger.Warn(err.Error()) // proper error handling instead of panic in your app
	}

	//_, err = stmtIns.Exec(block.Header.Number, hex.EncodeToString(block.Header.DataHash), txLen, txTime)
	//if err != nil {
	//	Logger.Warn(err.Error()) // proper error handling instead of panic in your app
	//}
	err = begin.Commit()
	if err != nil {
		logger.Warn(err.Error())
	}

	BlockNumberChans.Range(func(key, value interface{}) bool {
		value.(chan uint64) <- block.Header.Number
		return true
	})

	TxNumberChans.Range(func(key, value interface{}) bool {
		value.(chan uint64) <- mysql.TxNumber()
		return true
	})

	BlockChans.Range(func(key, value interface{}) bool {
		datas, _ := json.Marshal(&BlockInfo{
			Number:    block.Header.Number,
			TxCount:   txLen,
			BlockHash: hex.EncodeToString(block.Header.DataHash),
			DateTime:  txTime,
			Alice:     string(aliceBalance),
			Bob:       string(bobBalance),
			Type:      peerType,
		})
		value.(chan []byte) <- datas
		return true
	})
}

func syncBlock(ledgerClient *ledger.Client, targets fab.Peer) {
	height := mysql.GetBlockHeight()
	logger.Info("mysql block height: %d", height)

	ledgerInfoBefore, err := ledgerClient.QueryInfo(ledger.WithTargets(targets), ledger.WithMinTargets(1), ledger.WithMaxTargets(10))
	if err != nil {
		panic(fmt.Sprintf("QueryInfo return error: %s", err))
	}
	logger.Info("current block height: %d", ledgerInfoBefore.BCI.Height)

	if height > ledgerInfoBefore.BCI.Height-1 {
		panic(fmt.Sprintf("syncBlock invalid block height: %d, %d", height, ledgerInfoBefore.BCI.Height))
	} else if height < ledgerInfoBefore.BCI.Height-1 {
		for i := height; i < ledgerInfoBefore.BCI.Height; i++ {
			block, err := ledgerClient.QueryBlock(i)
			if err != nil {
				panic(err.Error())
			}
			go updateBlock(block)
		}
	}
}

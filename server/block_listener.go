package server

import (
	"encoding/hex"
	"fabric-byzantine/server/mysql"
	"fabric-byzantine/server/protoutil"
	"fmt"
	"strconv"
	"time"

	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/event"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
)

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

		metadata, err := protoutil.GetMetadataFromBlock(block, cb.BlockMetadataIndex(i))
		if err != nil {
			logger.Error("error get metadata from block: %s", err)
			continue
		}

		validationCode, _ := strconv.Atoi(string(metadata.Value))

		logger.Debug("Seek block number:%d", block.Header.Number)
		_, err = begin.Stmt(mysql.GetStmtTx()).Exec(block.Header.Number*uint64(appConf.TxNumPerBlock)+uint64(i), channelHeader.TxId, validationCode, txTime)
		if err != nil {
			logger.Warn(err.Error()) // proper error handling instead of panic in your app
		}

		//_, err = stmTx.Exec(block.Header.Number*uint64(AppConf.TxNumPerBlock)+uint64(i), channelHeader.TxId, validationCode, txTime)
		//if err != nil {
		//	Logger.Warn(err.Error()) // proper error handling instead of panic in your app
		//}
	}

	_, err = begin.Stmt(mysql.GetStmtBlock()).Exec(block.Header.Number, hex.EncodeToString(block.Header.DataHash), txLen, txTime)
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
}

func syncBlock(ledgerClient *ledger.Client, targets fab.Peer) {
	height := mysql.GetDBMgr().GetBlockHeight()
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

package server

import (
	"encoding/hex"
	"fabric-byzantine/server/mysql"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/event"
	cb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
)

func registerBlockEvent(eventClient *event.Client) {
	reg, eventch, err := eventClient.RegisterBlockEvent()
	if err != nil {
		logger.Error("Error registering for block events: %s", err)
	}
	defer eventClient.Unregister(reg)

	for {
		select {
		case e, ok := <-eventch:
			if !ok {
				logger.Error("unexpected closed channel while waiting for block event")
			}
			logger.Error("Received block event: %#v", e)
			if e.Block == nil {
				logger.Error("Expecting block in block event but got nil")
			}
			go updateBlock(e.Block)
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
		envelope, err := utils.GetEnvelopeFromBlock(envBytes)
		if err != nil {
			logger.Error("Error GetEnvelopeFromBlock:", err)
			break
		}
		payload, err := utils.GetPayload(envelope)
		if err != nil {
			logger.Error("Error GetPayload:", err)
			break
		}

		channelHeader, _ := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
		txTimestamp := channelHeader.Timestamp
		txTime = time.Unix(txTimestamp.GetSeconds(), int64(txTimestamp.GetNanos()))

		msg := cb.ConfigValue{}
		if err := proto.Unmarshal(payload.Data, &msg); err != nil {
			logger.Error("Error proto unmarshal", err)
			break
		}
		txId, err := strconv.ParseUint(string(msg.Value), 10, 64)
		if err != nil {
			logger.Error("Error ParseUint:", err)
			break
		}

		logger.Debug("Seek block number:%d, payload:%d", block.Header.Number, txId)
		_, err = begin.Stmt(mysql.GetStmtTx()).Exec(block.Header.Number*uint64(appConf.TxNumPerBlock)+uint64(i), channelHeader.TxId, "", "", "", 0, txTime)
		if err != nil {
			logger.Warn(err.Error()) // proper error handling instead of panic in your app
		}

		//_, err = stmTx.Exec(block.Header.Number*uint64(AppConf.TxNumPerBlock)+uint64(i), channelHeader.TxId, "", "", "", 0, txTime)
		//if err != nil {
		//	Logger.Warn(err.Error()) // proper error handling instead of panic in your app
		//}
	}

	_, err = begin.Stmt(mysql.GetStmtBlock()).Exec(block.Header.Number, hex.EncodeToString(block.Header.DataHash), txLen, 0, txTime)
	if err != nil {
		logger.Warn(err.Error()) // proper error handling instead of panic in your app
	}

	//_, err = stmtIns.Exec(block.Header.Number, hex.EncodeToString(block.Header.DataHash), txLen, 0, txTime)
	//if err != nil {
	//	Logger.Warn(err.Error()) // proper error handling instead of panic in your app
	//}
	err = begin.Commit()
	if err != nil {
		logger.Warn(err.Error())
	}
}

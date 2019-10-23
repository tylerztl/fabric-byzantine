package server

import (
	"encoding/json"
	"time"
)

type Statistics struct {
	ByzantinePeers  int8      `json:"byzantine_peers"`
	Datetime        time.Time `json:"datetime"`
	NormalTx        uint64    `json:"normal_tx"`
	NormalCommit    uint64    `json:"normal_commit"`
	ByzantineTx     uint64    `json:"byzantine_tx"`
	ByzantineCommit uint64    `json:"byzantine_commit"`
}

var StatisticsTable = make(map[int8]*Statistics)

func GetTable(num int8) *Statistics {
	table, ok := StatisticsTable[num]
	if ok {
		return table
	} else {
		return &Statistics{
			ByzantinePeers: num,
			Datetime:       time.Now(),
		}
	}
}

func ByzantineNum() {
	result, _ := GetSdkProvider().QueryCC(0, "mychannel1", "token",
		"getPeers", [][]byte{[]byte("fab")})
	peers := make(map[string]bool)
	json.Unmarshal(result, &peers)

	var num int8 = 0
	for _, v := range peers {
		if !v {
			num++
		}
	}
	GetTable(num)
}

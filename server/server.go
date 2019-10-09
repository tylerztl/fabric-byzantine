package server

import (
	. "fabric-byzantine/server/helpers"
)

type SdkProvider interface {
	InvokeCC(channelID, ccID, function string, args [][]byte) ([]byte, TransactionID, error)
	QueryCC(channelID, ccID, function string, args [][]byte) ([]byte, error)
}

type Handler struct {
	Provider SdkProvider
}

var hanlder = NewHandler()

func init() {
	provider, err := NewFabSdkProvider()
	if err != nil {
		panic(err)
	}
	hanlder.Provider = provider
}

func NewHandler() *Handler {
	return &Handler{}
}

func GetSdkProvider() SdkProvider {
	return hanlder.Provider
}

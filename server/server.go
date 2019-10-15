package server

type Handler struct {
	Provider *FabSdkProvider
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

func GetSdkProvider() *FabSdkProvider {
	return hanlder.Provider
}

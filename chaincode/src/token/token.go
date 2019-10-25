package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

type Token struct {
	Owner       string          `json:"Owner"`
	TotalSupply int             `json:"TotalSupply"`
	TokenName   string          `json:"TokenName"`
	BalanceOf   map[string]int  `json:"BalanceOf"`
	Peers       map[string]bool `json:"Peers"` // true: normal peer false: byzantine peer
	Flag        bool            `json:"Flag"`
}

func (token *Token) transfer(_from string, _to string, _value int) {
	if token.BalanceOf[_from] >= _value {
		token.BalanceOf[_from] -= _value
		token.BalanceOf[_to] += _value
	}
}

func (token *Token) balance(_from string) int {
	return token.BalanceOf[_from]
}

// Define the Smart Contract structure
type TokenContract struct {
}

func (s *TokenContract) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

func (s *TokenContract) issue(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 4 {
		return shim.Error("Incorrect number of arguments. Expecting 4")
	}

	owner := args[0]
	name := args[1]
	supply, _ := strconv.Atoi(args[2])
	byzantineNum, _ := strconv.Atoi(args[3])

	peers := make(map[string]bool)
	peers["peer0.org1.example.com"] = !(byzantineNum > 9)
	peers["peer0.org2.example.com"] = !(byzantineNum > 8)
	peers["peer0.org3.example.com"] = !(byzantineNum > 7)
	peers["peer0.org4.example.com"] = !(byzantineNum > 6)
	peers["peer0.org5.example.com"] = !(byzantineNum > 5)
	peers["peer0.org6.example.com"] = !(byzantineNum > 4)
	peers["peer0.org7.example.com"] = !(byzantineNum > 3)
	peers["peer0.org8.example.com"] = !(byzantineNum > 2)
	peers["peer0.org9.example.com"] = !(byzantineNum > 1)
	peers["peer0.org10.example.com"] = !(byzantineNum > 0)

	token := &Token{
		Owner:       owner,
		TotalSupply: supply,
		TokenName:   name,
		BalanceOf:   map[string]int{},
		Peers:       peers,
		Flag:        false, // true: byzantine peer attack
	}

	token.BalanceOf[token.Owner] = token.TotalSupply

	tokenAsBytes, _ := json.Marshal(token)
	err := stub.PutState(name, tokenAsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}
	fmt.Printf("Init %s \n", string(tokenAsBytes))

	return shim.Success(tokenAsBytes)
}

func (s *TokenContract) transfer(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) != 5 {
		return shim.Error("Incorrect number of arguments. Expecting 5")
	}
	_from := args[1]
	_to := args[2]
	_amount, _ := strconv.Atoi(args[3])
	if _amount <= 0 {
		return shim.Error("Incorrect number of amount")
	}

	tokenAsBytes, err := stub.GetState(args[0])
	if err != nil {
		return shim.Error(err.Error())
	}
	fmt.Printf("transferToken - begin %s \n", string(tokenAsBytes))

	flag, _ := strconv.ParseBool(args[4])
	token := Token{
		Owner:       "",
		TotalSupply: 0,
		TokenName:   "",
		BalanceOf:   make(map[string]int),
		Peers:       make(map[string]bool),
		Flag:        false,
	}

	err = json.Unmarshal(tokenAsBytes, &token)
	if err != nil {
		return shim.Error(err.Error())
	}
	token.Flag = flag
	token.transfer(_from, _to, _amount)

	tokenAsBytes, err = json.Marshal(token)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutState(args[0], tokenAsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}
	fmt.Printf("transferToken - end %s \n", string(tokenAsBytes))

	return shim.Success(tokenAsBytes)
}

func (s *TokenContract) setPeer(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}

	tokenAsBytes, err := stub.GetState(args[0])
	if err != nil {
		return shim.Error(err.Error())
	}
	fmt.Printf("setPeer - begin %s \n", string(tokenAsBytes))

	token := Token{
		Owner:       "",
		TotalSupply: 0,
		TokenName:   "",
		BalanceOf:   make(map[string]int),
		Peers:       make(map[string]bool),
		Flag:        false,
	}

	err = json.Unmarshal(tokenAsBytes, &token)
	if err != nil {
		return shim.Error(err.Error())
	}

	var peers map[string]string
	if err = json.Unmarshal([]byte(args[1]), &peers); err != nil {
		return shim.Error(err.Error())
	}

	for k, v := range peers {
		if _, ok := token.Peers[k]; ok {
			token.Peers[k], _ = strconv.ParseBool(v)
		}else {
			return shim.Error("invaild peer params")
		}
	}
	
	flag, _ := strconv.ParseBool(args[2])
	token.Flag = flag

	tokenAsBytes, err = json.Marshal(token)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutState(args[0], tokenAsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}
	fmt.Printf("setPeer - end %s \n", string(tokenAsBytes))

	return shim.Success(tokenAsBytes)
}

func (s *TokenContract) getPeers(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	tokenAsBytes, err := stub.GetState(args[0])
	if err != nil {
		return shim.Error(err.Error())
	}

	token := Token{
		Owner:       "",
		TotalSupply: 0,
		TokenName:   "",
		BalanceOf:   make(map[string]int),
		Peers:       make(map[string]bool),
		Flag:        false,
	}

	err = json.Unmarshal(tokenAsBytes, &token)
	if err != nil {
		return shim.Error(err.Error())
	}

	peers, err := json.Marshal(token.Peers)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(peers)
}

func (s *TokenContract) balance(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	tokenAsBytes, err := stub.GetState(args[0])
	if err != nil {
		return shim.Error(err.Error())
	}
	fmt.Println("get State:", string(tokenAsBytes))
	token := &Token{}
	err = json.Unmarshal(tokenAsBytes, token)
	if err != nil {
		fmt.Println("json Unmarshal err:", err)
		return shim.Error(err.Error())
	}
	fmt.Println("json Unmarshal succeed:", token)
	amount := token.balance(args[1])
	value := strconv.Itoa(amount)

	fmt.Printf("%s balance is %s \n", args[1], value)

	return shim.Success([]byte(value))
}

func (s *TokenContract) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	// Retrieve the requested Smart Contract function and arguments
	function, args := stub.GetFunctionAndParameters()
	// Route to the appropriate handler function to interact with the ledger appropriately
	if function == "issue" {
		return s.issue(stub, args)
	} else if function == "transfer" {
		return s.transfer(stub, args)
	} else if function == "balance" {
		return s.balance(stub, args)
	} else if function == "setPeer" {
		return s.setPeer(stub, args)
	} else if function == "getPeers" {
		return s.getPeers(stub, args)
	}

	return shim.Error(fmt.Sprintf("Invalid Token Contract function name:%s", function))
}

// The main function is only relevant in unit test mode. Only included here for completeness.
func main() {
	// Create a new Smart Contract
	err := shim.Start(new(TokenContract))
	if err != nil {
		fmt.Printf("Error creating new Token Contract: %s", err)
	}
}

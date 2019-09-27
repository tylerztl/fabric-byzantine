package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

type Token struct {
	Owner       string         `json:"Owner"`
	TotalSupply int            `json:"TotalSupply"`
	TokenName   string         `json:"TokenName"`
	TokenSymbol string         `json:"TokenSymbol"`
	BalanceOf   map[string]int `json:"BalanceOf"`
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

func (token *Token) mint(_value int) {
	token.BalanceOf[token.Owner] += _value
	token.TotalSupply += _value
}

type Account struct {
	Owner       string `json:"Owner"`
	TokenName   string `json:"TokenName"`
	TokenSymbol string `json:"TokenSymbol"`
	Balance     int    `json:"BalanceOf"`
}

// Define the Smart Contract structure
type TokenContract struct {
}

func (s *TokenContract) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

func (s *TokenContract) issue(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 4 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	owner := args[0]
	symbol := args[1]
	name := args[2]
	supply, _ := strconv.Atoi(args[3])

	token := &Token{
		Owner:       owner,
		TotalSupply: supply,
		TokenName:   name,
		TokenSymbol: symbol,
		BalanceOf:   map[string]int{},
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

	if len(args) != 4 {
		return shim.Error("Incorrect number of arguments. Expecting 4")
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

	token := Token{
		Owner:       "",
		TotalSupply: 0,
		TokenName:   "",
		TokenSymbol: "",
		BalanceOf:   make(map[string]int),
	}

	err = json.Unmarshal(tokenAsBytes, &token)
	if err != nil {
		return shim.Error(err.Error())
	}
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

	return shim.Success(nil)
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

	account := Account{
		Owner:       token.Owner,
		TokenName:   token.TokenName,
		TokenSymbol: token.TokenSymbol,
		Balance:     amount,
	}
	value := strconv.Itoa(amount)
	tokenAsBytes, _ = json.Marshal(account)
	fmt.Printf("%s balance is %s \n", args[1], value)

	return shim.Success(tokenAsBytes)
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

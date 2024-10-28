package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	lru "github.com/hashicorp/golang-lru"
)

var (
	infraURL   = "https://sepolia.infura.io/v3/8aaddcea52c24faeac2b2f6830528e93"
	client     *ethclient.Client
	nftAddress = common.HexToAddress("0x0e075058f9d07a328c5b84ad1e1e18d159fee1e8")
	abiJSON    = `[{"inputs":[{"internalType":"address","name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"player","type":"address"},{"internalType":"string","name":"tokenURI","type":"string"}],"name":"awardItem","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"nonpayable","type":"function"}]`
	chainID    = big.NewInt(11155111)
	key        *ecdsa.PrivateKey
	cache      *lru.Cache
)

type JSONRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	ID      int         `json:"id"`
}

func init() {
	var err error
	client, err = ethclient.Dial(infraURL)
	if err != nil {
		log.Fatalf("Error connecting to Ethereum client: %v", err)
	}
	fmt.Println("Connected to Ethereum client")

	cache, err = lru.New(100)
	if err != nil {
		log.Fatalf("Error creating cache: %v", err)
	}

	keyFile := "wallet/UTC--2024-10-23T17-43-44.459150000Z--a7001d3e6ed777bb28bd8246c5192bf5ab8d0151.json"
	password := "password"
	keyData, err := ioutil.ReadFile(keyFile)

	if err != nil {
		log.Fatalf("Failed to read key file: %v", err)
	}

	keyBytes, err := keystore.DecryptKey(keyData, password)
	if err != nil {
		log.Fatalf("Failed to decrypt key: %v", err)
	}
	key = keyBytes.PrivateKey
}

func handleJSONRPC(w http.ResponseWriter, r *http.Request) {
	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, req.ID, "Invalid JSON-RPC request")
		return
	}

	switch req.Method {
	case "mintNFT":
		if len(req.Params) < 2 {
			sendError(w, req.ID, "Invalid parameters")
			return
		}
		playerAddress := req.Params[0].(string)
		tokenURI := req.Params[1].(string)
		txHash, err := awardItem(playerAddress, tokenURI)
		if err != nil {
			sendError(w, req.ID, err.Error())
			return
		}
		sendResult(w, req.ID, map[string]string{"tx_hash": txHash})
	case "balanceOf":
		if len(req.Params) < 1 {
			sendError(w, req.ID, "Invalid parameters")
			return
		}
		playerAddress := req.Params[0].(string)
		balance, err := getBalanceOf(playerAddress)
		if err != nil {
			sendError(w, req.ID, err.Error())
			return
		}
		sendResult(w, req.ID, map[string]string{"balance": balance.String()})
	case "sendEther":
		if len(req.Params) < 3 {
			sendError(w, req.ID, "Invalid parameters")
			return
		}
		senderAddress := req.Params[0].(string)
		receiverAddress := req.Params[1].(string)
		amount := new(big.Int).SetUint64(uint64(req.Params[2].(float64)))

		txHash, err := sendEtherWithAuth(senderAddress, receiverAddress, amount)
		if err != nil {
			sendError(w, req.ID, err.Error())
			return
		}
		sendResult(w, req.ID, map[string]string{"tx_hash": txHash})

	default:
		sendError(w, req.ID, "Method not found")
	}
}

func awardItem(playerAddress, tokenURI string) (string, error) {
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return "", fmt.Errorf("failed to parse ABI: %v", err)
	}
	address := common.HexToAddress(playerAddress)

	txData, err := parsedABI.Pack("awardItem", address, tokenURI)
	if err != nil {
		return "", fmt.Errorf("failed to pack awardItem function: %v", err)
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to suggest gas price: %v", err)
	}

	tx := types.NewTransaction(15, nftAddress, big.NewInt(0), 100000, gasPrice, txData)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), key)
	if err != nil {
		return "", fmt.Errorf("transaction signing failed: %v", err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %v", err)
	}

	return signedTx.Hash().Hex(), nil
}

func getBalanceOf(playerAddress string) (*big.Int, error) {
	if val, ok := cache.Get(playerAddress); ok {
		return val.(*big.Int), nil
	}

	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %v", err)
	}
	address := common.HexToAddress(playerAddress)

	callData, err := parsedABI.Pack("balanceOf", address)
	if err != nil {
		return nil, fmt.Errorf("failed to pack balanceOf function: %v", err)
	}

	msg := ethereum.CallMsg{
		To:   &nftAddress,
		Data: callData,
	}

	output, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call balanceOf: %v", err)
	}

	var balance *big.Int
	err = parsedABI.UnpackIntoInterface(&balance, "balanceOf", output)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack balanceOf output: %v", err)
	}

	cache.Add(playerAddress, balance)
	return balance, nil
}

func sendEtherWithAuth(senderAddress, receiverAddress string, amount *big.Int) (string, error) {
	balance, err := getBalanceOf(senderAddress)
	if err != nil {
		return "", err
	}
	if balance.Cmp(big.NewInt(0)) <= 0 {
		return "", fmt.Errorf("user does not own an NFT")
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to suggest gas price: %v", err)
	}

	// nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(senderAddress)``)
	// if err != nil {
	// 	return "", fmt.Errorf("failed to fetch nonce: %v", err)
	// }

	tx := types.NewTransaction(19, common.HexToAddress(receiverAddress), amount, 21000, gasPrice, nil)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), key)
	if err != nil {
		return "", fmt.Errorf("transaction signing failed: %v", err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %v", err)
	}

	return signedTx.Hash().Hex(), nil
}

func sendResult(w http.ResponseWriter, id int, result interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func sendError(w http.ResponseWriter, id int, errMsg string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Error:   map[string]string{"message": errMsg},
		ID:      id,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	http.HandleFunc("/rpc", handleJSONRPC)
	fmt.Println("JSON-RPC server is running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

```markdown
# Blockchain JSON-RPC Example in Golang

This project implements a JSON-RPC server that interacts with an Ethereum smart contract for minting NFTs and checking their balances. It utilizes the Go programming language and the `go-ethereum` package for Ethereum interactions.

## Features

- **Mint NFTs**: The server allows users to mint new NFTs using the `mintNFT` method.
- **Check NFT Ownership**: Users can check their NFT balance with the `balanceOf` method.
- **Send Ether**: Users can send Ether between addresses, but this action is restricted to users who own an NFT.
- **LRU Caching**: Implements an LRU (Least Recently Used) caching strategy for optimized balance retrieval.
- **Ethereum Client Integration**: Connects to the Ethereum network via Infura.

## Prerequisites

- Go (version 1.17 or later)
- Access to an Infura account for Ethereum network connectivity
- A wallet file in JSON format for authentication (keystore)

## Installation

1. Clone the repository:
   ```bash
   git clone [https://github.com/yourusername/blockchainjsonrpc.git](https://github.com/branislavstojkovic70/NFT-Gate-Golang)
   cd project
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Update the `infraURL` and `keyFile` in `main.go` with your Infura project ID and the path to your wallet file.

## Running the Server

To start the JSON-RPC server, run the following command:
```bash
go run main.go
```
The server will start on port 8080.

## JSON-RPC Methods

### 1. `mintNFT`

- **Description**: Mints a new NFT for the specified player address.
- **Parameters**:
  - `playerAddress` (string): The Ethereum address of the player.
  - `tokenURI` (string): The metadata URI for the NFT.
- **Example Request**:
```json
{
    "jsonrpc": "2.0",
    "method": "mintNFT",
    "params": ["0xBe5EEEabbC51C16bf1c801F5ebe0A1b44E89009D", "https://link-to-tokenURI"],
    "id": 1
}
```

### 2. `balanceOf`

- **Description**: Checks the NFT balance of a specified address.
- **Parameters**:
  - `playerAddress` (string): The Ethereum address to check.
- **Example Request**:
```json
{
    "jsonrpc": "2.0",
    "method": "balanceOf",
    "params": ["0xBe5EEEabbC51C16bf1c801F5ebe0A1b44E89009D"],
    "id": 1
}
```

### 3. `sendEther`

- **Description**: Sends Ether from one address to another. This action is restricted to users who own an NFT.
- **Parameters**:
  - `senderAddress` (string): The Ethereum address of the sender.
  - `receiverAddress` (string): The Ethereum address of the receiver.
  - `amount` (float64): The amount of Ether to send.
- **Example Request**:
```json
{
    "jsonrpc": "2.0",
    "method": "sendEther",
    "params": ["0x0F9f5Feb075985Cf015C626bBaCFC0B37eE0130D", "0x0F9f5Feb075985Cf015C626bBaCFC0B37eE0130D", 0.01],
    "id": 1
}
```

## Important Note
The `sendEther` method will only succeed if the sender address owns at least one NFT. If the sender does not own an NFT, the transaction will be rejected.



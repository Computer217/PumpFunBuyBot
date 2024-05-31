package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/Computer217/SolanaBotV2/data"
	"github.com/Computer217/SolanaBotV2/test"
	"github.com/Computer217/SolanaBotV2/token"
	"github.com/Computer217/SolanaBotV2/transaction"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

func main() {
	// Parse command line arguments.
	var runTest bool
	var HeliusMainNet, WebHook, walletPath string
	var solanaAmount float64
	flag.BoolVar(&runTest, "test", false, "run test")
	flag.StringVar(&HeliusMainNet, "helius-api-key", "", "mainnet url")
	flag.StringVar(&WebHook, "webhook", "", "mainnet url")
	flag.StringVar(&walletPath, "wallet-path", "./wallet/botwallet.json", "wallet path")
	flag.Float64Var(&solanaAmount, "solana-amount", 0, "solana amount")
	flag.Parse()

	// If test flag is set, run test.
	if runTest {
		test.TestBuyOnDevNet(walletPath)
	}

	// wallet to perform transactions.
	var wallet solana.Wallet
	pk, err := solana.PrivateKeyFromSolanaKeygenFile(walletPath)
	if err != nil {
		log.Fatal("Failed to read wallet file:", err)
	}
	wallet.PrivateKey = pk

	// init context.
	ctx, cancel := context.WithCancel(context.Background())

	// Connect to the WebSocket server
	wsClient, err := ws.Connect(ctx, WebHook)
	if err != nil {
		log.Fatal("Failed to connect to WebSocket server:", err)
	}
	defer wsClient.Close()

	// Handle WebSocket close signal.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Subscribe to account changes
	wsSub, err := wsClient.LogsSubscribeMentions(solana.MustPublicKeyFromBase58(transaction.PumpFunCreateContract), rpc.CommitmentFinalized)
	if err != nil {
		log.Fatal("Failed to subscribe to account changes:", err)
	}

	// Create a new RPC client
	rpcClient := rpc.New(HeliusMainNet)
	defer rpcClient.Close()

	// Create transaction handler
	h := transaction.NewTransactionHandler(HeliusMainNet, rpcClient, wsClient, &wallet)
	h.SetSolPurchaseAmount(solanaAmount)

	// initialize channel for mint data.
	mintChan := make(chan *data.MintData)
	go token.FetchCreateTokens(ctx, wsSub, h, mintChan)

	// Go routine for sniping tokens.
	go transaction.SnipeTokens(ctx, mintChan, h)

	// Wait for interrupt signal to close the connection
	<-interrupt
	log.Println("interrupt signal received, closing connection")
	wsClient.Close()
	cancel()
}

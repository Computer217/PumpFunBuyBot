package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/Computer217/SolanaBotV2/data"
	"github.com/Computer217/SolanaBotV2/token"
	"github.com/Computer217/SolanaBotV2/transaction"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

func main() {
	// Parse command line arguments.
	dryRun := flag.Bool("test", false, "run test")
	HeliusMainNet := flag.String("helius-rpc-api-key", "", "mainnet url api key")
	WebHook := flag.String("helius-websocket-api-key", "", "websocket url api key")
	walletPath := flag.String("wallet-path", "./wallet/botwallet.json", "wallet path")
	defaultSolanaAmount := flag.Float64("solana-amount", 0, "solana amount to purchase")
	flag.Parse()

	// extract wallet to perform transactions.
	var wallet solana.Wallet
	pk, err := solana.PrivateKeyFromSolanaKeygenFile(*walletPath)
	if err != nil {
		log.Fatal("Failed to read wallet file:", err)
	}
	wallet.PrivateKey = pk

	// init context.
	ctx, cancel := context.WithCancel(context.Background())

	// Connect to the WebSocket server.
	wsClient, err := ws.Connect(ctx, *WebHook)
	if err != nil {
		log.Fatal("Failed to connect to WebSocket server:", err)
	}
	defer wsClient.Close()

	// Handle WebSocket close signal.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Subscribe to account changes.
	wsSub, err := wsClient.LogsSubscribeMentions(solana.MustPublicKeyFromBase58(transaction.PumpFunCreateTokenContract), rpc.CommitmentFinalized)
	if err != nil {
		log.Fatal("Failed to subscribe to account changes:", err)
	}

	// Create a new RPC client.
	rpcClient := rpc.New(*HeliusMainNet)
	defer rpcClient.Close()

	// Create transaction handler.
	h := transaction.NewTransactionHandler(*HeliusMainNet, rpcClient, wsClient, &wallet, *dryRun)

	// Set the amount of sol used to purchase tokens.
	h.SetPurchaseAmount(*defaultSolanaAmount)

	// initialize channel for mint data.
	mintChan := make(chan *data.MintData)

	// listen for pump fun contract mint activity.
	// for each mint, fetch the mint data and pass to the channel.
	go token.FetchCreatedTokenMintData(ctx, wsSub, h, mintChan)

	// Go routine for sniping tokens.
	go transaction.SnipeTokens(ctx, mintChan, h)

	// Wait for interrupt signal to close the connection
	<-interrupt
	log.Println("interrupt signal received, closing connection")
	wsClient.Close()
	cancel()
}

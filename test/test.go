package test

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/Computer217/SolanaBotV2/transaction"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

const localTestNetRPC = "http://127.0.0.1:8899"
const localTestNetWS = "ws://127.0.0.1:8900"

// Airdrop to this wallet:
// solana airdrop 10 ~/my-solana-wallet.json

// To spin up dev enviorment:
// solana-test-validator --reset \
//   --clone 4wTV1YmiEkRvAtNtsSGPtUrqRYQMe5SKy2uB4Jjaxnjf \
//   --clone CebN5WGQ4jvEPvsVU4EoHEpgzq1VV7AbicfhtW4xC9iM \
//   --clone EcLRGW3cUpc1yump1sY9NDXKzjkJmaYAzpZcF7Le7Mka \
//   --clone BQonLHNiagXd29RQSgLtJVkPWKyp6MgnUE6D9A8hRhjE \
//   --clone 11111111111111111111111111111111 \
//   --clone TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA \
//   --clone SysvarRent111111111111111111111111111111111 \
//   --clone ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL \
//   --clone ComputeBudget111111111111111111111111111111 \
//   --clone 6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P \
//   --url http://api.mainnet-beta.solana.com

// In a seperate terminal run:  solana config set --url http://127.0.0.1:8899

// TODO: Figure out what other data needs to be imported.

func TestBuyOnDevNet(walletPath string) {
	// Initialize Test Here:
	// init context.
	ctx, cancel := context.WithCancel(context.Background())

	// Connect to the devnet WebSocket server
	wsClient, err := ws.Connect(ctx, localTestNetWS) // TODO: change to helius mainnet webhook.
	if err != nil {
		log.Fatal("Failed to connect to WebSocket server:", err)
	}
	defer wsClient.Close()

	// Create a new devnet RPC client
	rpcClient := rpc.New(localTestNetRPC)
	defer rpcClient.Close()

	// Handle WebSocket close signal.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Create a new account.
	wallet, err := createFundedDevNetWallet(ctx, rpcClient, walletPath)
	if err != nil {
		log.Fatalf("failed to create a new account: %v", err)
	}

	recentBlockhash, err := rpcClient.GetRecentBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		log.Fatalf("failed to get recent blockhash: %v", err)
	}

	// Buy this token https://solscan.io/token/EcLRGW3cUpc1yump1sY9NDXKzjkJmaYAzpZcF7Le7Mka
	tokenBuyParams := transaction.BuyParams{
		TokenAmount:            5645191266917,
		ComputeLimit:           100000,
		ComputeUnit:            100000,
		Wallet:                 wallet,
		Mint:                   solana.MustPublicKeyFromBase58("EcLRGW3cUpc1yump1sY9NDXKzjkJmaYAzpZcF7Le7Mka"),
		BondingCurve:           solana.MustPublicKeyFromBase58("BQonLHNiagXd29RQSgLtJVkPWKyp6MgnUE6D9A8hRhjE"),
		AssociatedBondingCurve: solana.MustPublicKeyFromBase58("3XQ93roQjHvPvbuwZDDpYeiiPedVnyo8wsHkiuv1YRuF"),
		MaxLamportCost:         1785000000,
	}

	// Build transaction with instructions to buy tokens.
	tx, err := tokenBuyParams.BuildBuyTransaction()
	if err != nil {
		log.Fatalf("failed to build buy transaction: %v", err)
	}
	tx.Message.RecentBlockhash = recentBlockhash.Value.Blockhash

	h := transaction.NewTransactionHandler(localTestNetRPC, rpcClient, wsClient, wallet)
	// Sign and send the transaction.
	if err := h.SignAndSendTransaction(ctx, tx, wallet); err != nil {
		log.Fatalf("failed to sign and send transaction: %v", err)
	}

	// Wait for interrupt signal to close the connection
	<-interrupt
	log.Println("interrupt signal received, closing connection")
	wsClient.Close()
	cancel()

}

func createFundedDevNetWallet(ctx context.Context, rpcClient *rpc.Client, walletPath string) (*solana.Wallet, error) {
	// Create a new account:
	account := solana.NewWallet()

	if walletPath != "" {
		// wallet to perform transactions.
		pk, err := solana.PrivateKeyFromSolanaKeygenFile(walletPath)
		if err != nil {
			log.Fatal("Failed to read wallet file:", err)
		}
		account.PrivateKey = pk
		return account, nil
	}

	// Airdrop 1 SOL to the new account:
	out, err := rpcClient.RequestAirdrop(
		ctx,
		account.PublicKey(),
		solana.LAMPORTS_PER_SOL*1,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to airdrop to account: %v", err)
	}
	fmt.Println("airdrop transaction signature:", out)
	time.Sleep(5 * time.Second)

	return account, nil
}

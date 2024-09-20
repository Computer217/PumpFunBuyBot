package transaction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"

	"github.com/Computer217/SolanaBotV2/data"
	"github.com/Computer217/SolanaBotV2/generated/pump"
	"github.com/davecgh/go-spew/spew"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/gagliardetto/solana-go/text"

	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	confirm "github.com/gagliardetto/solana-go/rpc/sendAndConfirmTransaction"
)

const JitoTip5 = "DfXygSm4jCyNCybVYYK6DwvWqjKee8pbDmJGcLWNDXjh"
const PumpFunBuyContract = "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P"
const PumpFunCreateTokenContract = "TSLvdd1pWpHVjahSpsvCXUbgwsL3JAcvokwaKt1eokM"
const PumpFun = "CebN5WGQ4jvEPvsVU4EoHEpgzq1VV7AbicfhtW4xC9iM"
const Global = "4wTV1YmiEkRvAtNtsSGPtUrqRYQMe5SKy2uB4Jjaxnjf"
const BlockRouteContract = "HWEoBxYs7ssKuudEjzjmpfJVX7Dvi7wescFsVx2L5yoY"
const EventAuthority = "Ce6TQqeHC9p8KetsN6JsjHK7UTZk7nasjjnr7XxXp9F1"

// createComputeLimitInstruction creates an instruction that sets the compute unit limit.
func createComputeLimitInstruction(computeLimit uint32) (*computebudget.Instruction, error) {
	return computebudget.NewSetComputeUnitLimitInstructionBuilder().SetUnits(computeLimit).ValidateAndBuild()
}

// createJitoTip5Instruction creates an instruction that transfers an amount to the jitoTip5 initiative.
func createJitoTip5Instruction(wallet solana.PublicKey, tip uint64) (*system.Instruction, error) {
	transf := system.NewTransferInstructionBuilder()
	transf = transf.SetLamports(tip)
	transf = transf.SetFundingAccount(wallet)

	// convert string address to PublicKey
	pubk, err := solana.PublicKeyFromBase58(JitoTip5)
	if err != nil {
		return nil, err
	}
	transf = transf.SetRecipientAccount(pubk)
	return transf.ValidateAndBuild()
}

// createComputeUnitPriceInstruction creates an instruction that sets the compute unit price.
func createComputeUnitPriceInstruction(computeUnit uint64) (*computebudget.Instruction, error) {
	return computebudget.NewSetComputeUnitPriceInstructionBuilder().SetMicroLamports(computeUnit).ValidateAndBuild()
}

// createAssociateTokenAccountInstruction creates an instruction that creates a token account.
// Input:
// - Wallet: the account that will be the owner of the new token account (main wallet).
// - Mint: the mint of the token that will be associated with the new token account.
func createAssociateTokenAccountInstruction(Wallet solana.PublicKey, mint solana.PublicKey) (*associatedtokenaccount.Instruction, error) {
	return associatedtokenaccount.NewCreateInstruction(Wallet, Wallet, mint).ValidateAndBuild()
}

// createPumpFunInstruction creates the instruction for buying tokens on pump.fun.
func createPumpFunInstruction(wallet, mint, bondingCurve, associatedBondingCurve, associatedUser solana.PublicKey, maxLamportCost, tokenAmount uint64) (*pump.Instruction, error) {
	pumpInstruction := pump.NewBuyInstruction(
		tokenAmount,
		maxLamportCost,
		solana.MustPublicKeyFromBase58(Global),
		solana.MustPublicKeyFromBase58(PumpFun),
		mint,
		bondingCurve,
		associatedBondingCurve,
		associatedUser,
		wallet,
		solana.SystemProgramID,
		solana.TokenProgramID,
		solana.SysVarRentPubkey,
		solana.MustPublicKeyFromBase58(EventAuthority),
		solana.MustPublicKeyFromBase58(PumpFunBuyContract))

	return pumpInstruction.ValidateAndBuild()
}

type BuyParams struct {
	ComputeLimit           uint32 // Default: 100000
	ComputeUnit            uint64 // Default: 100000
	Wallet                 *solana.Wallet
	Mint                   solana.PublicKey
	BondingCurve           solana.PublicKey
	AssociatedBondingCurve solana.PublicKey
	AssociatedUser         solana.PublicKey
	TokenAmount            uint64
	MaxLamportCost         uint64 // Price including slippage tolerance.
}

// BuildBuyInstructions creates the instructions for buying tokens on pump.fun.
func (b *BuyParams) BuildBuyTransaction() (*solana.Transaction, error) {
	// create compute limit instruction
	computeLimitInstruction, err := createComputeLimitInstruction(b.ComputeLimit)
	if err != nil {
		return nil, err
	}

	// create jito tip instruction
	jitoTipInstruction, err := createJitoTip5Instruction(b.Wallet.PublicKey(),
		10000000)
	if err != nil {
		return nil, err
	}

	// create compute unit price instruction.
	computeUnitPriceInstruction, err := createComputeUnitPriceInstruction(b.ComputeUnit)
	if err != nil {
		return nil, err
	}

	// create associate token account instruction.
	associateTokenAccountInstruction, err := createAssociateTokenAccountInstruction(b.Wallet.PublicKey(),
		b.Mint)
	if err != nil {
		return nil, err
	}

	// create pump fun instruction.
	pumpFunInstruction, err := createPumpFunInstruction(b.Wallet.PublicKey(),
		b.Mint,
		b.BondingCurve,
		b.AssociatedBondingCurve,
		b.AssociatedUser,
		b.MaxLamportCost,
		b.TokenAmount)
	if err != nil {
		return nil, err
	}

	return solana.NewTransaction([]solana.Instruction{
		computeLimitInstruction,
		jitoTipInstruction,
		computeUnitPriceInstruction,
		associateTokenAccountInstruction,
		pumpFunInstruction},
		solana.Hash{},
		solana.TransactionPayer(b.Wallet.PublicKey()),
	)
}

type TransactionHandler struct {
	UrlEndpoint   string
	RpcClient     *rpc.Client
	WsClient      *ws.Client
	Wallet        *solana.Wallet
	TokensHandled map[solana.PublicKey]bool
	LampsToBuy    uint64
	DryRun        bool
}

// NewTransactionHandler creates a handler to send transactions to interact with pump.fun contract.
func NewTransactionHandler(urlEndpoint string, rpcClient *rpc.Client, wsClient *ws.Client, wallet *solana.Wallet, dryRun bool) *TransactionHandler {
	return &TransactionHandler{
		UrlEndpoint:   urlEndpoint,
		RpcClient:     rpcClient,
		WsClient:      wsClient,
		Wallet:        wallet,
		TokensHandled: make(map[solana.PublicKey]bool),
		DryRun:        dryRun,
	}
}

// set the amount of tokens to buy in sol.
func (t *TransactionHandler) SetPurchaseAmount(sol float64) {
	// instruction requires the amount of tokens to buy in lamports.
	t.LampsToBuy = SolToLamp(sol)
}

func (t *TransactionHandler) SignAndSendTransaction(ctx context.Context, tx *solana.Transaction, wallet *solana.Wallet) error {
	// Sign the transaction:
	_, err := tx.Sign(
		func(key solana.PublicKey) *solana.PrivateKey {
			if wallet.PublicKey().Equals(key) {
				return &wallet.PrivateKey
			}
			return nil
		},
	)
	if err != nil {
		return fmt.Errorf("tx.Sign failure: %v", err)
	}
	spew.Dump(tx)
	tx.EncodeToTree(text.NewTreeEncoder(os.Stdout, "Buying from Pump.fun"))

	// If dry run is enabled, simulate the buy.
	if t.DryRun {
		log.Println("Simulating buy...")
		resp, err := t.RpcClient.SimulateTransaction(ctx, tx)
		if err != nil {
			log.Fatal("Failed to simulate transaction:", err)
		}
		spew.Dump(resp)
		return nil
	}

	// Send transaction, and wait for confirmation:
	sig, err := confirm.SendAndConfirmTransaction(
		ctx,
		t.RpcClient,
		t.WsClient,
		tx,
	)
	if err != nil {
		return fmt.Errorf("SendAndConfirmTransaction failed: %v", err)
	}
	spew.Dump(sig)
	return nil
}

func (t *TransactionHandler) buyFromPumpfun(ctx context.Context, mintData *data.MintData) {
	// Calculate the amount of tokens to buy.
	mintData.TokenAmount = uint64(math.Floor(LampToSol(int64(t.LampsToBuy)) / mintData.TokenPriceInSol))
	fmt.Printf("\033[1;33mPurchase Amount (SOL):\033[0m \033[1;36m%.12f\033[0m\n", LampToSol(int64(t.LampsToBuy)))
	fmt.Printf("\033[1;33mToken Price:\033[0m \033[1;36m%.12f\033[0m\n", mintData.TokenPriceInSol)
	fmt.Printf("\033[1;33mToken Amount:\033[0m \033[1;36m%v\033[0m\n", mintData.TokenAmount)
	fmt.Printf("\033[1;33mMarket Cap (SOL):\033[0m \033[1;36m%v\033[0m\n", mintData.Info.MarketCapInSol)
	fmt.Printf("\033[1;33mMax Tx Cost (SOL):\033[0m \033[1;36m%.12f\033[0m\n", LampToSol(int64(maxLamportCost(t.LampsToBuy))))

	// derive associated token account (ie. associateUser value from IDL).
	ata, _, err := solana.FindAssociatedTokenAddress(
		t.Wallet.PublicKey(),
		mintData.Info.MintAddress,
	)
	if err != nil {
		log.Fatalf("failed to derive associated token account: %v", err)
	}

	// Derive bonding curve address.
	// define the seeds used to derive the PDA
	// getProgramDerivedAddress equivalent.
	seeds := [][]byte{
		[]byte("bonding-curve"),
		mintData.Info.MintAddress.Bytes(),
	}

	bondingCurve, _, err := solana.FindProgramAddress(seeds, solana.MustPublicKeyFromBase58(PumpFunBuyContract))
	if err != nil {
		log.Fatalf("failed to derive bonding curve address: %v", err)
	}

	// Derive associated bonding curve address.
	associatedBondingCurve, _, err := solana.FindAssociatedTokenAddress(
		bondingCurve,
		mintData.Info.MintAddress,
	)
	if err != nil {
		log.Fatalf("failed to derive associated bonding curve address: %v", err)
	}

	// set all parameters for buying tokens.
	tokenBuyParams := BuyParams{
		TokenAmount:            mintData.TokenAmount * 1000000, // shift by 6 since pumpfun specifies 6 decimal places.
		ComputeLimit:           100000,
		ComputeUnit:            100000,
		Wallet:                 t.Wallet,
		Mint:                   mintData.Info.MintAddress,
		BondingCurve:           bondingCurve,
		AssociatedBondingCurve: associatedBondingCurve,
		AssociatedUser:         ata,
		MaxLamportCost:         maxLamportCost(t.LampsToBuy),
	}

	// Build transaction with instructions to buy tokens.
	tx, err := tokenBuyParams.BuildBuyTransaction()
	if err != nil {
		log.Fatalf("failed to build buy transaction: %v", err)
	}

	// Fetch recent blockhash and set in transaction.
	recentBlockhash, err := t.RpcClient.GetRecentBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		log.Fatalf("Failed to fetch recent blockhash: %v", err)
	}
	tx.Message.RecentBlockhash = recentBlockhash.Value.Blockhash

	// Sign and send the transaction.
	if err := t.SignAndSendTransaction(ctx, tx, t.Wallet); err != nil {
		log.Fatalf("failed to sign and send transaction: %v", err)
	}

	log.Println("Listening for contract activity again...")
}

func (t *TransactionHandler) handleTransaction(ctx context.Context, mintData *data.MintData) {
	// handle duplicate token creation messages.
	if found := t.TokensHandled[mintData.Info.MintAddress]; found {
		return
	} else {
		t.TokensHandled[mintData.Info.MintAddress] = true
	}

	// Print to stdout when a avalid token is found.
	fmt.Println("\033[1;32m|============================|")
	fmt.Println("|        Token Found         |")
	fmt.Println("|============================|\033[0m")
	fmt.Printf("\033[1;33mToken Creation Hash:\033[0m \033[1;32m%v\033[0m\n", mintData.CreationHash)
	fmt.Printf("\033[1;33mMint Address:\033[0m \033[1;32m%v\033[0m\n", mintData.Info.MintAddress)
	fmt.Printf("\033[1;33mTotal Supply:\033[0m \033[1;32m%v\033[0m\n", mintData.Info.TotalSupply)
	fmt.Printf("\033[1;33mAmount of Tokens Purchased by Dev:\033[0m \033[1;32m%f\033[0m\n", mintData.DevSupply)
	fmt.Printf("\033[1;33mPrice of Token paid by dev in Sol:\033[0m \033[1;32m%.12f\033[0m\n", mintData.TokenPriceInSol)
	fmt.Printf("\033[1;33murl:\033[0m \033[1;32mpump.fun/%v\033[0m\n", mintData.Info.MintAddress)

	fmt.Println()
	t.buyFromPumpfun(ctx, mintData)
}

func (t *TransactionHandler) GetParsedTransactionFromSignature(hash string) (*data.GetParsedTransactionFromSignatureResponse, error) {
	// JSON-RPC request body
	requestBody, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getTransaction",
		"params": []interface{}{
			hash,
			map[string]interface{}{
				"maxSupportedTransactionVersion": 0,
				"encoding":                       "jsonParsed",
			},
		},
	})
	if err != nil {
		log.Println("Error marshaling JSON:", err)
		return nil, err
	}

	// Send the HTTP request
	response, err := http.Post(t.UrlEndpoint, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Println("Error sending request:", err)
		return nil, err
	}
	defer response.Body.Close()

	// response.Body is a stream that can only be read once.
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
		return nil, err
	}
	nullSkip, err := skipNull(bodyBytes)
	if err != nil {
		return nil, err
	}
	if nullSkip {
		return nil, nil
	}

	// Skip legacy version transactions.
	legacySkip, err := skipLegacy(bodyBytes)
	if err != nil {
		return nil, err
	}
	if legacySkip {
		return nil, nil
	}

	// Read the response body into structured Data.
	result := new(data.GetParsedTransactionFromSignatureResponse)
	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("error decoding JSON: %v", err)
	}
	return result, nil
}

// maxLamportCost calculates the maximum lamport cost for a given amount of tokens to buy including the slipage.
// for no slippage, set the slippage tolerance to -1.
func maxLamportCost(LampsToBuy uint64) uint64 {
	solValue := LampToSol(int64(LampsToBuy))
	totalAmountWithSlippage := solValue * 1.25 // %25 slippage tolerance.
	return SolToLamp(totalAmountWithSlippage)
}

// skipNull skips null body responses when fetching for a transaction hash.
func skipNull(bodyBytes []byte) (bool, error) {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, bodyBytes, "", "    ")
	if err != nil {
		log.Println("Error generating pretty JSON:", err)
		return false, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		log.Println("Error decoding JSON:", err)
		return false, err
	}

	// Check if the specific field is nil
	if result["result"] == nil {
		// log.Printf("result is nil: %v", prettyJSON.String())
		return true, nil
	}
	return false, nil
}

// skipLegacy skips legacy version transactions.
func skipLegacy(bodyBytes []byte) (bool, error) {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, bodyBytes, "", "    ")
	if err != nil {
		log.Println("Error generating pretty JSON:", err)
		return false, err
	}

	// Skip legacy version transactions.
	// Unmarshal the JSON into a map
	var legacyResult struct {
		Result struct {
			Version interface{} `json:"version"`
		} `json:"result"`
	}
	err = json.Unmarshal(bodyBytes, &legacyResult)
	if err != nil {
		log.Println("Error decoding vanilla JSON:", err)
		return false, err
	}

	// Check the type of the version field
	switch v := legacyResult.Result.Version.(type) {
	case string:
		if v == "legacy" {
			return true, nil
		}
	case float64:
		if v == 0 {
			return false, nil
		}
	default:

		return false, fmt.Errorf("unexpected type for version field: %T, transaction: %v", v, prettyJSON.String())
	}
	return false, nil
}

func LampToSol(lamports int64) float64 {
	return float64(lamports) / 1000000000.0
}

func SolToLamp(sol float64) uint64 {
	return uint64(sol * 1000000000)
}

func SnipeTokens(ctx context.Context, mintChan chan *data.MintData, t *TransactionHandler) {
	for {
		select {
		case <-ctx.Done():
			// Context has been cancelled, stop the goroutine
			return
		case mintData := <-mintChan:
			// Handle transaction.
			t.handleTransaction(ctx, mintData)
		}
	}
}

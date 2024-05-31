package transaction

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Computer217/SolanaBotV2/data"
	"github.com/davecgh/go-spew/spew"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/gagliardetto/solana-go/text"
	"github.com/near/borsh-go"

	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	confirm "github.com/gagliardetto/solana-go/rpc/sendAndConfirmTransaction"
)

const JitoTip5 = "DfXygSm4jCyNCybVYYK6DwvWqjKee8pbDmJGcLWNDXjh"
const PumpFunBuyContract = "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P"
const PumpFunCreateContract = "TSLvdd1pWpHVjahSpsvCXUbgwsL3JAcvokwaKt1eokM"
const PumpFun = "CebN5WGQ4jvEPvsVU4EoHEpgzq1VV7AbicfhtW4xC9iM"
const Global = "4wTV1YmiEkRvAtNtsSGPtUrqRYQMe5SKy2uB4Jjaxnjf"
const BlockRouteContract = "HWEoBxYs7ssKuudEjzjmpfJVX7Dvi7wescFsVx2L5yoY"

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
func createPumpFunInstruction(wallet, mint, bondingCurve, associatedBondingCurve solana.PublicKey, maxLamportCost, tokenAmount uint64) (*solana.GenericInstruction, error) {
	// Define the program ID for Pump.Fun
	programID := solana.MustPublicKeyFromBase58(PumpFunBuyContract)

	// Get the associated token account for the receiver
	associatedDestinationTokenAddr, _, err := solana.FindAssociatedTokenAddress(wallet, mint)
	if err != nil {
		return nil, err
	}

	// Define input accounts
	accounts := []*solana.AccountMeta{
		{PublicKey: solana.MustPublicKeyFromBase58(Global), IsSigner: false, IsWritable: false},                                 // Global.
		{PublicKey: solana.MustPublicKeyFromBase58(PumpFun), IsSigner: false, IsWritable: true},                                 // Fee Recipient Pump.fun.
		{PublicKey: solana.MustPublicKeyFromBase58(mint.String()), IsSigner: false, IsWritable: false},                          // Mint Address.
		{PublicKey: solana.MustPublicKeyFromBase58(bondingCurve.String()), IsSigner: false, IsWritable: true},                   // Bonding Curve.
		{PublicKey: solana.MustPublicKeyFromBase58(associatedBondingCurve.String()), IsSigner: false, IsWritable: true},         // Associated Bonding Curve.
		{PublicKey: solana.MustPublicKeyFromBase58(associatedDestinationTokenAddr.String()), IsSigner: false, IsWritable: true}, // Associated User.
		{PublicKey: solana.MustPublicKeyFromBase58(wallet.String()), IsSigner: true, IsWritable: true},                          // User.
		{PublicKey: solana.SystemProgramID, IsSigner: false, IsWritable: false},                                                 // System Program.
		{PublicKey: solana.TokenProgramID, IsSigner: false, IsWritable: false},                                                  // Token Program.
		{PublicKey: solana.SysVarRentPubkey, IsSigner: false, IsWritable: false},                                                // Rent Program.
		// {PublicKey: solana.BPFLoaderUpgradeableProgramID, IsSigner: false, IsWritable: false},                                   // NEED TO FIGURE OUT.
		// {PublicKey: solana.MustPublicKeyFromBase58(programID.String()), IsSigner: false, IsWritable: false},                     // NEED TO FIGURE OUT.
	}

	// serialize the instruction inputs.
	data, err := borsh.Serialize(struct {
		Cmd  uint8
		Arg1 uint64
		Arg2 uint64
	}{
		Cmd:  3, // The index of the function you want to call
		Arg1: tokenAmount,
		Arg2: maxLamportCost,
	})
	if err != nil {
		return nil, err
	}
	// Create the instruction.
	return solana.NewInstruction(programID, accounts, data), nil
}

type BuyParams struct {
	ComputeLimit           uint32 // Default: 100000
	ComputeUnit            uint64 // Default: 100000
	Wallet                 *solana.Wallet
	Mint                   solana.PublicKey
	BondingCurve           solana.PublicKey
	AssociatedBondingCurve solana.PublicKey
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
	TokensHandled map[string]bool
	SolToBuy      uint64
}

func NewTransactionHandler(urlEndpoint string, rpcClient *rpc.Client, wsClient *ws.Client, wallet *solana.Wallet) *TransactionHandler {
	return &TransactionHandler{
		UrlEndpoint:   urlEndpoint,
		RpcClient:     rpcClient,
		WsClient:      wsClient,
		Wallet:        wallet,
		TokensHandled: make(map[string]bool),
	}
}

func (t *TransactionHandler) SetSolPurchaseAmount(sol float64) {
	t.SolToBuy = SolToLamp(sol)
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
		return err
	}
	spew.Dump(tx)
	tx.EncodeToTree(text.NewTreeEncoder(os.Stdout, "Buying from Pump.fun"))

	// Send transaction, and wait for confirmation:
	sig, err := confirm.SendAndConfirmTransaction(
		ctx,
		t.RpcClient,
		t.WsClient,
		tx,
	)
	if err != nil {
		return err
	}
	spew.Dump(sig)
	return nil
}

func (t *TransactionHandler) sendTransaction(ctx context.Context, mintData *data.MintData) {
	// Calculate Slippage 10%
	mintData.TokenAmount = uint64(math.Floor(float64(t.SolToBuy) / mintData.TokenPriceInSol))

	slippage := float64(t.SolToBuy) + (float64(t.SolToBuy) * 0.1)

	tokenBuyParams := BuyParams{
		TokenAmount:            mintData.TokenAmount,
		ComputeLimit:           100000,
		ComputeUnit:            100000,
		Wallet:                 t.Wallet,
		Mint:                   solana.MustPublicKeyFromBase58(mintData.Info.Mint),
		BondingCurve:           solana.MustPublicKeyFromBase58(mintData.Info.BondingCurve),
		AssociatedBondingCurve: solana.MustPublicKeyFromBase58(mintData.Info.AssociateBondingCurve),
		MaxLamportCost:         uint64(math.RoundToEven(slippage)),
	}

	// Build transaction with instructions to buy tokens.
	tx, err := tokenBuyParams.BuildBuyTransaction()
	if err != nil {
		log.Fatalf("failed to build buy transaction: %v", err)
	}

	// Fetch recent blockhash
	recentBlockhash, err := t.RpcClient.GetRecentBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		log.Fatalf("Failed to fetch recent blockhash: %v", err)
	}

	tx.Message.RecentBlockhash = recentBlockhash.Value.Blockhash

	// Sign and send the transaction.
	if err := t.SignAndSendTransaction(ctx, tx, t.Wallet); err != nil {
		log.Fatalf("failed to sign and send transaction: %v", err)
	}
}

func (t *TransactionHandler) handleTransaction(ctx context.Context, mintData *data.MintData) {
	if found := t.TokensHandled[mintData.Info.Mint]; found {
		return
	} else {
		t.TokensHandled[mintData.Info.Mint] = true

	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("|============================|")
	fmt.Println("|        Token Found         |")
	fmt.Println("|============================|")
	// fmt.Printf("Token Name: %v\n", l.TransactionLog) Print Symbol and Name     url := fmt.Sprintf("https://api.solanabeach.io/v1/token/%s", mintAddress)
	fmt.Printf("Mint Address: %v\n", mintData.Info.Mint)
	fmt.Printf("Total Supply: %v\n", mintData.Info.TotalSupply)
	fmt.Printf("Amount of Tokens Purchased by Dev %f\n", mintData.DevSupply)
	fmt.Printf("Price of Token paid by dev in Sol: %.12f\n", mintData.TokenPriceInSol)
	fmt.Printf("url: pump.fun/%v\n", mintData.Info.Mint)
	// Print amount purchased by the dev
	fmt.Printf("Purchase Parameters{\n    token_amount: %v\n    sol_amount: %v\n    token_info: %+v\n}\n", mintData.TokenAmount, mintData.SolAmount, mintData.Info)
	fmt.Println()
	fmt.Printf("Do you want to BUY this token? [y/n]: ")
	input, _ := reader.ReadString('\n')

	if strings.Contains(strings.ToLower(input), "y") {
		go t.sendTransaction(ctx, mintData)
	} else {
		fmt.Println("|============================|")
		fmt.Println("|Transaction skipped by user.|")
		fmt.Println("|============================|")
		fmt.Println()
	}
	log.Println("Listening for contract activity again...")
	fmt.Println()
}

func (t *TransactionHandler) FilterTransactionForMintData(signature solana.Signature) (*data.MintData, error) {
	// fetch granular tx data.
	data, err := t.getParsedTransactionFromSignature(signature.String())
	if err != nil {
		return nil, err
	}
	if data != nil {
		// Parse the transaction for mint data.
		mint, err := parseTransaction(data)
		if err != nil {
			log.Println("Error parsing transaction:", err)
			return nil, err
		}
		return mint, nil
	}
	return nil, nil
}

func (t *TransactionHandler) getParsedTransactionFromSignature(hash string) (*data.GetParsedTransactionFromSignatureResponse, error) {
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

func parseTransaction(d *data.GetParsedTransactionFromSignatureResponse) (*data.MintData, error) {
	// if transaction results in error skip.
	if d.Result.Meta.Err != (data.Err{}) {
		return nil, nil
	}

	// Check if the transaction has as a destination the pump.fun contract.
	calledPumpFun := calledPumpfunContract(d)

	// fetch mint data and check if it's a mint transaction.
	mintData, foundMint := fetchMintInstruction(d)

	// if calledPumpFun || foundMint {
	// 	log.Println("Transaction signature: ", d.Result.Transaction.Signatures)
	// 	log.Println("calledPumpFun: ", calledPumpFun)
	// 	log.Println("foundMint: ", foundMint)
	// }

	if foundMint && calledPumpFun {
		// fetch market cap.
		fetchMarketCap(d, mintData)
		fetchBondingCurve(d, mintData)
		// fetch Bonding Curve
		return mintData, nil
	}
	return nil, nil
}

func fetchMarketCap(data *data.GetParsedTransactionFromSignatureResponse, mintData *data.MintData) {
	var splTokenTransfer, solanaTokenTransfer struct {
		Accounts    []string "json:\"accounts,omitempty\""
		Data        string   "json:\"data,omitempty\""
		ProgramID   string   "json:\"programId\""
		StackHeight int      "json:\"stackHeight\""
		Parsed      struct {
			Info struct {
				Amount        string "json:\"amount\""
				Authority     string "json:\"authority\""
				Destination   string "json:\"destination\""
				Source        string "json:\"source\""
				Lamports      int64  "json:\"lamports,omitempty\""
				Mint          string "json:\"mint,omitempty\""
				MintAuthority string "json:\"mintAuthority,omitempty\""
				Account       string "json:\"account,omitempty\""
			} "json:\"info\""
			Type string "json:\"type\""
		} "json:\"parsed,omitempty\""
		Program string "json:\"program,omitempty\""
	}

	for _, innerInstruction := range data.Result.Meta.InnerInstructions {
		for _, instruction := range innerInstruction.Instructions {
			if instruction.Parsed.Type == "transfer" {
				i, _ := strconv.Atoi(instruction.Parsed.Info.Amount)
				if instruction.Program == "spl-token" && i > 0 {
					splTokenTransfer = instruction
				}
				if instruction.Program == "system" && instruction.ProgramID == "11111111111111111111111111111111" && instruction.Parsed.Info.Destination != PumpFun && instruction.Parsed.Info.Destination != BlockRouteContract && instruction.Parsed.Info.Lamports != 0 {
					solanaTokenTransfer = instruction
				}
			}
		}
	}

	// calculate market cap.
	// Fetch the dev amount and the remaining circulating supply.
	var remainingCirculatingSupply, devSupply float64
	if splTokenTransfer.Parsed.Info.Amount == data.Result.Meta.PostTokenBalances[1].UITokenAmount.Amount {
		devSupply = data.Result.Meta.PostTokenBalances[1].UITokenAmount.UIAmount
		remainingCirculatingSupply = data.Result.Meta.PostTokenBalances[0].UITokenAmount.UIAmount
	} else {
		devSupply = data.Result.Meta.PostTokenBalances[0].UITokenAmount.UIAmount
		remainingCirculatingSupply = data.Result.Meta.PostTokenBalances[1].UITokenAmount.UIAmount
	}
	mintData.DevSupply = devSupply

	// calculate how much dev paid per token.
	amountPaidForToken := float64(LampToSol(solanaTokenTransfer.Parsed.Info.Lamports))
	priceInSol := amountPaidForToken / devSupply
	mintData.TokenPriceInSol = priceInSol

	// Calculate market cap.
	mintData.Info.MarketCapInSol = fmt.Sprintf("%f", priceInSol*remainingCirculatingSupply)
}

func fetchBondingCurve(d *data.GetParsedTransactionFromSignatureResponse, mintData *data.MintData) {
	for _, innerInstruction := range d.Result.Meta.InnerInstructions {
		for _, instruction := range innerInstruction.Instructions {
			if instruction.Parsed.Type == "transfer" && instruction.ProgramID == "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA" && instruction.Program == "spl-token" {
				mintData.Info.BondingCurve = instruction.Parsed.Info.Destination
			}
		}
	}
}

func fetchMintInstruction(d *data.GetParsedTransactionFromSignatureResponse) (*data.MintData, bool) {
	for _, innerInstruction := range d.Result.Meta.InnerInstructions {
		for _, instruction := range innerInstruction.Instructions {
			if instruction.Parsed.Type == "mintTo" && instruction.ProgramID == "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA" {
				return &data.MintData{
					Info: &data.MintInfo{
						TotalSupply:           instruction.Parsed.Info.Amount,
						Mint:                  instruction.Parsed.Info.Mint,
						AssociateBondingCurve: instruction.Parsed.Info.Account,
						MintAuthority:         instruction.Parsed.Info.MintAuthority,
					},
					Type: instruction.Parsed.Type,
				}, true
			}
		}
	}
	return nil, false
}

func calledPumpfunContract(data *data.GetParsedTransactionFromSignatureResponse) bool {
	// underTwoSolPurchased denotes whether the dev bought less than 2 sol at the time of token creation.
	// pumpFunCalled denotes whether the dev interacted with the pump.fun contract at the time of token creation.
	var overOneSolPurchased, pumpFunCalled bool

	// if there is a preTokenBalance then it is not a mintTransaction.
	if len(data.Result.Meta.PreTokenBalances) > 0 {
		return false
	}

	// fetch the dev address. This is the source address for the transaction interacting with the pumpFun contract address.
	var devAddress string
	for _, innerInstruction := range data.Result.Meta.InnerInstructions {
		for _, instruction := range innerInstruction.Instructions {
			if instruction.Parsed.Type == "transfer" {
				if instruction.Parsed.Info.Destination == PumpFun {
					devAddress = instruction.Parsed.Info.Source
				}
			}
		}
	}

	// theAccountOwnerOfNewTokenOne and accountOwnerOfNewTokenTwo are either:
	// 1. The address for the dev token account.
	// 2. The address for the account associated with the mint of the new token.

	if len(data.Result.Meta.PostTokenBalances) < 2 {
		log.Println("PostTokenBalances is less than 2")
		log.Println(data.Result.Transaction.Signatures)
	}

	accountOwnerOfNewTokenOne := data.Result.Meta.PostTokenBalances[0].Owner
	accountOwnerOfNewTokenTwo := data.Result.Meta.PostTokenBalances[1].Owner

	// check if the dev purchased less than 2 sol.
	for _, innerInstruction := range data.Result.Meta.InnerInstructions {
		for _, instruction := range innerInstruction.Instructions {
			if instruction.Parsed.Type == "transfer" {
				// check the following 3 conditions:
				// 1. The source/sender address is the dev address.
				// 2. The destination address is either the dev token account or the contract holding account associated with the mint of the new token.
				// 3. The amount of lamports transferred is less than 2 sol.
				if instruction.Parsed.Info.Source == devAddress &&
					(instruction.Parsed.Info.Destination == accountOwnerOfNewTokenOne || instruction.Parsed.Info.Destination == accountOwnerOfNewTokenTwo) &&
					LampToSol(instruction.Parsed.Info.Lamports) <= 2.0 {
					overOneSolPurchased = true
				}
			}
		}
	}

	// check if we interacted with the pump.fun contract.
	for _, innerInstruction := range data.Result.Meta.InnerInstructions {
		for _, instruction := range innerInstruction.Instructions {
			if instruction.ProgramID == "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P" {
				pumpFunCalled = true
			}
		}
	}
	return overOneSolPurchased && pumpFunCalled
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

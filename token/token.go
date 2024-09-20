package token

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/Computer217/SolanaBotV2/data"
	"github.com/Computer217/SolanaBotV2/transaction"
	"github.com/avast/retry-go"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

func FetchCreatedTokenMintData(ctx context.Context, wsSub *ws.LogSubscription, t *transaction.TransactionHandler, mintChan chan *data.MintData) {
	log.Println("Listening for contract activity...")
	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, cleaning up go routine FetchCreateTokens...")
			return
		default:
			// Receive account updates.
			msg, err := wsSub.Recv()
			if err != nil {
				log.Fatalf("Failed to receive message %v: %v", msg, err)
			}
			if msg.Value.Err != nil {
				continue
			}
			// Fetch transaction details.
			var data *data.GetParsedTransactionFromSignatureResponse
			err = retry.Do(
				func() error {
					data, err = t.GetParsedTransactionFromSignature(msg.Value.Signature.String())
					return err
				},
				retry.Attempts(3),        // Set the number of retries
				retry.Delay(time.Second), // Set the delay between retries
			)
			if err != nil {
				log.Printf("Retry budget Exhausted for fetching transaction %v details: %v\n", msg.Value.Signature, err)
				continue
			}
			if data.Result.Meta.Err.Message != "" {
				log.Printf("Fetching Signature: %v Received GetTransaction() error: %+v\n", msg.Value.Signature, data.Result.Meta.Err)
				continue
			}

			// Filter transaction for mint data.
			mintData, err := filterTransactionForMintData(data)
			if err != nil {
				log.Println("Error parsing transaction:", err)
				return
			}
			if mintData != nil {
				mintData.CreationHash = msg.Value.Signature.String()
				mintChan <- mintData
			}
		}
	}
}

func filterTransactionForMintData(data *data.GetParsedTransactionFromSignatureResponse) (*data.MintData, error) {
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

func parseTransaction(d *data.GetParsedTransactionFromSignatureResponse) (*data.MintData, error) {
	// if transaction results in error skip.
	if d.Result.Meta.Err != (data.Err{}) {
		return nil, nil
	}

	// fetch mint data and check if it's a mint transaction.
	mintData, foundMint := fetchMintInstruction(d)

	// Check mint transaction values associated to pumpfun contract.
	overOneSolPurchased := assertAmountPurchasedByDev(d)

	if foundMint && overOneSolPurchased {
		// fetch market cap.
		fetchMarketCap(d, mintData)
		return mintData, nil
	}
	return nil, nil
}

func assertAmountPurchasedByDev(data *data.GetParsedTransactionFromSignatureResponse) bool {
	// overOneSolPurchased denotes whether the dev bought more than 1 sol at the time of token creation.
	// pumpFunCalled denotes whether the dev interacted with the pump.fun contract at the time of token creation.
	var overOneSolPurchased bool

	// fetch the dev address. This is the source address for the transaction interacting with the pumpFun contract address.
	var devAddress string
	for _, innerInstruction := range data.Result.Meta.InnerInstructions {
		for _, instruction := range innerInstruction.Instructions {
			if instruction.Parsed.Type == "transfer" {
				if instruction.Parsed.Info.Destination == transaction.PumpFun {
					devAddress = instruction.Parsed.Info.Source
				}
			}
		}
	}

	// theAccountOwnerOfNewTokenOne and accountOwnerOfNewTokenTwo are either:
	// 1. The address for the dev token account.
	// 2. The address for the account associated with the mint of the new token.
	// I saw these values flipped in practice. What we are trying to determine here is whether the dev purchased more than 1 sol.
	accountOwnerOfNewTokenOne := data.Result.Meta.PostTokenBalances[0].Owner
	accountOwnerOfNewTokenTwo := data.Result.Meta.PostTokenBalances[1].Owner

	for _, innerInstruction := range data.Result.Meta.InnerInstructions {
		for _, instruction := range innerInstruction.Instructions {
			if instruction.Parsed.Type == "transfer" {
				// check the following 3 conditions:
				// 1. The source/sender address is the dev address.
				// 2. The destination address is either the dev token account or the contract holding account associated with the mint of the new token.
				// 3. The amount of lamports transferred is more than 1 sol.
				if instruction.Parsed.Info.Source == devAddress &&
					(instruction.Parsed.Info.Destination == accountOwnerOfNewTokenOne || instruction.Parsed.Info.Destination == accountOwnerOfNewTokenTwo) &&
					transaction.LampToSol(instruction.Parsed.Info.Lamports) >= 1.0 {
					overOneSolPurchased = true
				}
			}
		}
	}

	return overOneSolPurchased
}

func fetchMintInstruction(d *data.GetParsedTransactionFromSignatureResponse) (*data.MintData, bool) {
	for _, innerInstruction := range d.Result.Meta.InnerInstructions {
		for _, instruction := range innerInstruction.Instructions {
			if instruction.Parsed.Type == "mintTo" && instruction.ProgramID == "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA" {
				return &data.MintData{
					Info: &data.MintInfo{
						TotalSupply:   instruction.Parsed.Info.Amount,
						MintAddress:   solana.MustPublicKeyFromBase58(instruction.Parsed.Info.Mint),
						MintAuthority: instruction.Parsed.Info.MintAuthority,
					},
					Type: instruction.Parsed.Type,
				}, true
			}
		}
	}
	return nil, false
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
				if instruction.Program == "system" && instruction.ProgramID == "11111111111111111111111111111111" && instruction.Parsed.Info.Destination != transaction.PumpFun && instruction.Parsed.Info.Destination != transaction.BlockRouteContract && instruction.Parsed.Info.Lamports != 0 {
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
	amountPaidForToken := float64(transaction.LampToSol(solanaTokenTransfer.Parsed.Info.Lamports))
	priceInSol := amountPaidForToken / devSupply
	mintData.TokenPriceInSol = priceInSol

	// Calculate market cap.
	mintData.Info.MarketCapInSol = fmt.Sprintf("%f", priceInSol*remainingCirculatingSupply)
}

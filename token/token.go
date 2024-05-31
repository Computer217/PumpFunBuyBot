package token

import (
	"context"
	"log"
	"time"

	"github.com/Computer217/SolanaBotV2/data"
	"github.com/Computer217/SolanaBotV2/transaction"
	"github.com/avast/retry-go"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

var maxSupportedTransactionVersion uint64 = 0

func FetchCreateTokens(ctx context.Context, wsSub *ws.LogSubscription, t *transaction.TransactionHandler, mintChan chan *data.MintData) {
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
				log.Fatal("Failed to receive message:", err)
			}
			if msg.Value.Err != nil {
				continue
			}
			// Fetch transaction details.
			var tx *rpc.GetTransactionResult
			err = retry.Do(
				func() error {
					tx, err = t.RpcClient.GetTransaction(ctx, msg.Value.Signature, &rpc.GetTransactionOpts{Encoding: solana.EncodingBase58, Commitment: rpc.CommitmentFinalized, MaxSupportedTransactionVersion: &maxSupportedTransactionVersion})
					return err
				},
				retry.Attempts(3),        // Set the number of retries
				retry.Delay(time.Second), // Set the delay between retries
			)
			if err != nil {
				log.Printf("Retry budget Exhausted for fetching transaction details: %v\n", err)
				continue
			}
			if tx.Meta.Err != nil {
				log.Printf("Fetching Signature: %v Received GetTransaction() error: %+v\n", msg.Value.Signature, tx.Meta.Err)
				continue
			}

			// Filter transaction for mint data.
			mintData, err := t.FilterTransactionForMintData(msg.Value.Signature)
			if err != nil {
				log.Println("Error parsing transaction:", err)
				return
			}
			if mintData != nil {
				mintChan <- mintData
			}
		}
	}
}

This code builds and sends a buy transtation to the pump.fun contract.

Parameters:
* --helius-rpc-api-key="" // REQUIRED(string): helius rpc key or solana free rpc endpoint
* --helius-websocket-api-key="" // REQUIRED(string): helius websocket key or solana free websocket endpoint
* --wallet-path="" // REQUIRED(string): relative path to wallet private key in JSON format
* --Solana-amount="" // REQUIRED(string): Solana amount to be used for buying token
* --test="" // OPTIONAL(bool): test denotes whether the transaction is sent to the mainnet or simulated. 

Example call:

> go run all --helius-rpc-api-key="https://api.mainnet-beta.solana.com" --helius-websocket-api-key="ws://api.mainnet-beta.solana.com" --wallet-path="./path/to/wallet/private/key/wallet.json" --solana-amount=0.001 --test=true 

General Logic:

1. subscribe tp the pump.fun mint contract
2. parse mint events to pump.fun
3. Extract mint data
4. Build pump.fun buy transaction
5. Send pump.fun buy transaction

 

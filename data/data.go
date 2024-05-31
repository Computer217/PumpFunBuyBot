package data

type Err struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type GetParsedTransactionFromSignatureResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		BlockTime int `json:"blockTime"`
		Meta      struct {
			ComputeUnitsConsumed int `json:"computeUnitsConsumed"`
			Err                  Err `json:"err,omitempty"`
			Fee                  int `json:"fee"`
			InnerInstructions    []struct {
				Index        int `json:"index"`
				Instructions []struct {
					Accounts    []string `json:"accounts,omitempty"`
					Data        string   `json:"data,omitempty"`
					ProgramID   string   `json:"programId"`
					StackHeight int      `json:"stackHeight"`
					Parsed      struct {
						Info struct {
							Amount        string `json:"amount"`
							Authority     string `json:"authority"`
							Destination   string `json:"destination"`
							Source        string `json:"source"`
							Lamports      int64  `json:"lamports,omitempty"`
							Mint          string `json:"mint,omitempty"`
							MintAuthority string `json:"mintAuthority,omitempty"`
							Account       string `json:"account,omitempty"`
						} `json:"info"`
						Type string `json:"type"`
					} `json:"parsed,omitempty"`
					Program string `json:"program,omitempty"`
				} `json:"instructions"`
			} `json:"innerInstructions"`
			LogMessages       []string      `json:"logMessages"`
			PostBalances      []interface{} `json:"postBalances"`
			PostTokenBalances []struct {
				AccountIndex  int    `json:"accountIndex"`
				Mint          string `json:"mint"`
				Owner         string `json:"owner"`
				ProgramID     string `json:"programId"`
				UITokenAmount struct {
					Amount         string  `json:"amount"`
					Decimals       int     `json:"decimals"`
					UIAmount       float64 `json:"uiAmount"`
					UIAmountString string  `json:"uiAmountString"`
				} `json:"uiTokenAmount"`
			} `json:"postTokenBalances"`
			PreBalances      []interface{} `json:"preBalances"`
			PreTokenBalances []struct {
				AccountIndex  int    `json:"accountIndex"`
				Mint          string `json:"mint"`
				Owner         string `json:"owner"`
				ProgramID     string `json:"programId"`
				UITokenAmount struct {
					Amount         string  `json:"amount"`
					Decimals       int     `json:"decimals"`
					UIAmount       float64 `json:"uiAmount"`
					UIAmountString string  `json:"uiAmountString"`
				} `json:"uiTokenAmount"`
			} `json:"preTokenBalances"`
			Rewards []interface{} `json:"rewards"`
			Status  struct {
				Ok interface{} `json:"Ok"`
			} `json:"status"`
		} `json:"meta"`
		Slot        int `json:"slot"`
		Transaction struct {
			Message struct {
				AccountKeys []struct {
					Pubkey   string `json:"pubkey"`
					Signer   bool   `json:"signer"`
					Source   string `json:"source"`
					Writable bool   `json:"writable"`
				} `json:"accountKeys"`
				AddressTableLookups []struct {
					AccountKey      string `json:"accountKey"`
					ReadonlyIndexes []int  `json:"readonlyIndexes"`
					WritableIndexes []int  `json:"writableIndexes"`
				} `json:"addressTableLookups"`
				Instructions []struct {
					Accounts    []interface{} `json:"accounts"`
					Data        string        `json:"data"`
					ProgramID   string        `json:"programId"`
					StackHeight interface{}   `json:"stackHeight"`
				} `json:"instructions"`
				RecentBlockhash string `json:"recentBlockhash"`
			} `json:"message"`
			Signatures []string `json:"signatures"`
		} `json:"transaction"`
		Version int `json:"version"`
	} `json:"result"`
	ID int `json:"id"`
}

type LogsNotificationResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  struct {
		Result struct {
			Context struct {
				Slot int `json:"slot"`
			} `json:"context"`
			Value struct {
				Signature string      `json:"signature"`
				Err       interface{} `json:"err"`
				Logs      []string    `json:"logs"`
			} `json:"value"`
		} `json:"result"`
		Subscription int `json:"subscription"`
	} `json:"params"`
}

type MintData struct {
	Info            *MintInfo
	Type            string
	TokenAmount     uint64
	SolAmount       uint64
	DevSupply       float64
	TokenPriceInSol float64
}

type MintInfo struct {
	Mint                  string
	MintAuthority         string
	BondingCurve          string
	AssociateBondingCurve string // Associated Bonding Curve / Account.
	TotalSupply           string
	MarketCapInSol        string
}

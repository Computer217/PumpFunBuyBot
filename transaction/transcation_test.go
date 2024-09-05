package transaction

import (
	"testing"

	"github.com/Computer217/SolanaBotV2/data"
)

func TestFetchBondingCurveFunction(t *testing.T) {
	// Define your test cases
	testCases := []struct {
		name string
		json string
		want *data.MintData
	}{
		// TODO: Add test cases.
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// TODO:
			// tx := &data.GetParsedTransactionFromSignatureResponse{}
			// err := json.Unmarshal([]byte(tc.json), tx)
			// if err != nil {
			// 	t.Fatalf("json.Unmarshal returned an error: %v", err)
			// }
			// // Call the function with the test data
			// got := &data.MintData{Info: &data.MintInfo{}}
			// fetchBondingCurve(tx, got)

			// // Check if the result matches what's expected
			// if !reflect.DeepEqual(got, tc.want) {
			// 	t.Errorf("fetchBondingCurveFunction() = %+v,\nwant %+v", got.Info, tc.want.Info)
			// }
		})
	}
}

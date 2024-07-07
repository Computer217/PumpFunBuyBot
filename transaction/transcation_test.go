package transaction

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Computer217/SolanaBotV2/data"
)

func TestFetchBondingCurveAndAssociatedUserFunction(t *testing.T) {
	// Define your test cases
	testCases := []struct {
		name string
		json string
		want *data.MintData
	}{
		{
			name: "success",
			json: data.GetBondingCurveData,
			want: &data.MintData{
				Info: &data.MintInfo{
					BondingCurve:   "GSJ7gix2Q7k81quBYfH9ceD7Vz2bC8Bfim9JJBiKWtKE",
					AssociatedUser: "ARK76xJyuS9kEJqVQtxPBtXct1D9wnufoJjwf5onyaQh",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tx := &data.GetParsedTransactionFromSignatureResponse{}
			err := json.Unmarshal([]byte(tc.json), tx)
			if err != nil {
				t.Fatalf("json.Unmarshal returned an error: %v", err)
			}
			// Call the function with the test data
			got := &data.MintData{Info: &data.MintInfo{}}
			fetchBondingCurveAndAssociatedUser(tx, got)

			// Check if the result matches what's expected
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("fetchBondingCurveFunction() = %+v,\nwant %+v", got.Info, tc.want.Info)
			}
		})
	}
}

package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"

	"github.com/gagliardetto/solana-go"
)

type TokenListMeta struct {
	Address solana.PublicKey `json:"address"`
	Symbol  string           `json:"symbol"`
	Name    string           `json:"name"`
}

type TokenList struct {
	Tokens []TokenListMeta `json:"tokens"`
}

func GetTokenWallet(wallet solana.PublicKey, mint solana.PublicKey) (solana.PublicKey, error) {
	addr, _, err := solana.FindProgramAddress(
		[][]byte{
			wallet.Bytes(),
			solana.TokenProgramID.Bytes(),
			mint.Bytes(),
		},
		solana.SPLAssociatedTokenAccountProgramID,
	)
	return addr, err
}

func FetchTokenMeta() []TokenListMeta {
	var tokenList TokenList
	tokenListUrl := fmt.Sprint(CDN + "/solana.tokenlist.json")
	res, err := http.DefaultClient.Get(tokenListUrl)
	if err != nil {
		return tokenList.Tokens
	}
	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return tokenList.Tokens
	}

	err = json.Unmarshal(data, &tokenList)
	if err != nil {
		return tokenList.Tokens
	}

	return tokenList.Tokens

}

func ConvertUiAmountToAmount(uiAmount float64, decimals uint8) uint64 {
	return uint64(uiAmount * math.Pow10(int(decimals)))
}

func ConvertAmountToUiAmount(amount uint64, decimals uint8) float64 {
	return float64(amount) / math.Pow10(int(decimals))
}

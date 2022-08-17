package utils

import (
	"context"
	"encoding/json"
	"log"

	token_metadata "github.com/gagliardetto/metaplex-go/clients/token-metadata"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func GetTokenMintOwner(rpcClient *rpc.Client, mint solana.PublicKey) *solana.PublicKey {

	holders, err := rpcClient.GetTokenLargestAccounts(context.TODO(), mint, rpc.CommitmentConfirmed)
	if err != nil {
		log.Println(err)
		return nil
	}

	if len(holders.Value) == 0 {
		log.Println("WARN", "Token lacks substance")
		return nil
	}
	ownerTokenAccount := holders.Value[0].Address
	ownerTokenAccountInfo, err := rpcClient.GetAccountInfoWithOpts(context.TODO(), ownerTokenAccount, &rpc.GetAccountInfoOpts{Encoding: "jsonParsed"})
	if err != nil {
		log.Println(err)
		return nil
	}

	type tokenDataParsed struct {
		Parsed struct {
			Info struct {
				Owner solana.PublicKey `json:"owner"`
			} `json:"info"`
		} `json:"parsed"`
	}

	var tokenMeta tokenDataParsed
	err = json.Unmarshal(ownerTokenAccountInfo.Value.Data.GetRawJSON(), &tokenMeta)
	if err != nil {
		log.Println(err)
		return nil
	}

	return tokenMeta.Parsed.Info.Owner.ToPointer()
}

func GetMetadata(mint solana.PublicKey) (solana.PublicKey, error) {
	addr, _, err := solana.FindProgramAddress(
		[][]byte{
			[]byte("metadata"),
			token_metadata.ProgramID.Bytes(),
			mint.Bytes(),
		},
		token_metadata.ProgramID,
	)
	return addr, err
}

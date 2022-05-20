package utils

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/text"

	"github.com/gagliardetto/solana-go/rpc"
)

func SendTx(
	doc string,
	instructions []solana.Instruction,
	signers []solana.PrivateKey,
	feePayer solana.PublicKey,
) *solana.Signature {
	rpcClient := rpc.New(NETWORK)

	recent, err := rpcClient.GetRecentBlockhash(context.TODO(), rpc.CommitmentFinalized)
	if err != nil {
		log.Println("PANIC!!!", fmt.Errorf("unable to fetch recent blockhash - %w", err))
		return nil
	}

	tx, err := solana.NewTransaction(
		instructions,
		recent.Value.Blockhash,
		solana.TransactionPayer(feePayer),
	)
	if err != nil {
		log.Println("PANIC!!!", fmt.Errorf("unable to create transaction"))
		return nil
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		for _, candidate := range signers {
			if candidate.PublicKey().Equals(key) {
				return &candidate
			}
		}
		return nil
	})
	if err != nil {
		log.Println("PANIC!!!", fmt.Errorf("unable to sign transaction: %w", err))
		return nil
	}

	tx.EncodeTree(text.NewTreeEncoder(os.Stdout, doc))
	sig, err := rpcClient.SendEncodedTransaction(context.TODO(), tx.MustToBase64())
	if err != nil {
		fmt.Println(err)
		return nil
	}

	return &sig
}

func VerifyTransactionSignature(txSignature string) bool {
	rpcClient := rpc.New(NETWORK)

	transaction, err := rpcClient.GetTransaction(context.TODO(), solana.MustSignatureFromBase58(txSignature), nil)
	if err != nil {
		return false
	}
	if transaction == nil {
		return false
	}
	if &err == nil {
		return false
	}

	return true
}

package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/gagliardetto/solana-go"
	atok "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/programs/token"
	"triptych.labs/airdrop/v2/src/utils"
)

var MINT solana.PublicKey = solana.MustPublicKeyFromBase58("9bqobQxWDpx14dGob5jxXRmSsChtwh29KaxqAU5fDyDK")

func main() {
	op := os.Args[1]
	switch op {
	case "init":
		{
			initState()
		}
	case "sync_spreadsheet":
		{
			syncSpreadsheet()
		}
	case "airdrop":
		{
			airdrop()
		}
	case "verify":
		{
			verify()
		}
	default:
		{
			panic(errors.New("invalid operation"))
		}
	}
}

func syncSpreadsheet() {
	state := make(map[string]string)
	spreadsheet, err := os.Open("./spreadsheet.csv")
	if err != nil {
		panic(err)
	}
	csvReader := csv.NewReader(spreadsheet)
	data, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	airdropTxMapData, err := ioutil.ReadFile("./report.json")
	if err != nil {
		panic(err)
	}
	json.Unmarshal(airdropTxMapData, &state)

	for _, record := range data[1:] {
		for colIndex, cell := range record {
			if colIndex == 0 {
				if val, ok := state[cell]; !ok {
					if val != "" {
						state[cell] = ""
					}
				}
			}
		}
	}

	stateJSON, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		panic(err)
	}
	err = os.WriteFile("./report.json", stateJSON, 0644)
	if err != nil {
		panic(err)
	}
}

func initState() {
	state := make(map[string]string)
	spreadsheet, err := os.Open("./spreadsheet.csv")
	if err != nil {
		panic(err)
	}
	csvReader := csv.NewReader(spreadsheet)
	data, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	for _, record := range data[1:] {
		for colIndex, cell := range record {
			if colIndex == 0 {
				state[cell] = ""
			}
		}
	}

	stateJSON, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		panic(err)
	}
	err = os.WriteFile("./report.json", stateJSON, 0644)
	if err != nil {
		panic(err)
	}
}

func verify() {
	state := make(map[string]string)
	airdropTxMapData, err := ioutil.ReadFile("./report.json")
	if err != nil {
		panic(err)
	}
	json.Unmarshal(airdropTxMapData, &state)

	stateLockJSON, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		panic(err)
	}
	err = os.WriteFile("./report.lock", stateLockJSON, 0644)
	if err != nil {
		panic(err)
	}

	for address, transaction := range state {
		valid := utils.VerifyTransactionSignature(transaction)
		if !valid {
			state[address] = ""
		}
	}

	stateJSON, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		panic(err)
	}
	err = os.WriteFile("./report.json", stateJSON, 0644)
	if err != nil {
		panic(err)
	}

}

func airdrop() {
	oracle, err := solana.PrivateKeyFromSolanaKeygenFile("./oracle.key")
	if err != nil {
		panic(err)
	}

	state := make(map[string]string)
	airdropTxMapData, err := ioutil.ReadFile("./report.json")
	if err != nil {
		panic(err)
	}

	json.Unmarshal(airdropTxMapData, &state)

	for address, signature := range state {
		if signature != "" {
			continue
		}
		user := solana.MustPublicKeyFromBase58(address)
		userTokenAccountAddress, err := utils.GetTokenWallet(user, MINT)
		if err != nil {
			panic(err)
		}
		var instructions []solana.Instruction
		instructions = append(instructions,
			atok.NewCreateInstructionBuilder().
				SetPayer(oracle.PublicKey()).
				SetWallet(user).
				SetMint(MINT).
				Build(),

			token.NewMintToInstructionBuilder().
				SetMintAccount(MINT).
				SetDestinationAccount(userTokenAccountAddress).
				SetAuthorityAccount(oracle.PublicKey()).
				SetAmount(1).
				Build(),
		)

		txSignature := utils.SendTx(
			"airdrop",
			instructions,
			append(make([]solana.PrivateKey, 0), oracle),
			oracle.PublicKey(),
		)
		fmt.Println(txSignature)
		if txSignature != nil {
			state[user.String()] = txSignature.String()
		} else {
			state[user.String()] = ""
		}
	}

	stateJSON, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		panic(err)
	}
	err = os.WriteFile("./report.json", stateJSON, 0644)
	if err != nil {
		panic(err)
	}
}

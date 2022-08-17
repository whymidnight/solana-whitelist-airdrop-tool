package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	token_metadata "github.com/gagliardetto/metaplex-go/clients/token-metadata"
	"github.com/gagliardetto/solana-go"
	atok "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"triptych.labs/airdrop/v2/src/utils"
)

var MINT solana.PublicKey = solana.MustPublicKeyFromBase58("3nNeqjQ724xe4E8kKVGP5t8NFt4HSLksvN5NwMFphCSM")

type Ticket struct {
	Signature string `json:"Signature"`
	Amount    int    `json:"Amount"`
}

type State map[string]Ticket

func main() {
	op := os.Args[1]
	switch op {
	case "init":
		{
			initState()
		}
	case "register_token_meta":
		{
			register_token_meta()
		}
	case "sync_spreadsheet":
		{
			syncSpreadsheet()
		}
	case "sync_hashlist":
		{
			initStateUsingHashList(string(os.Args[2]))
		}
	case "airdrop":
		{
			airdrop()
		}
	case "airdrop_to":
		{
			airdropTo()
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

func register_token_meta() {
	oracle, err := solana.PrivateKeyFromSolanaKeygenFile("./airdrop.key")
	if err != nil {
		panic(err)
	}

	metadata := token_metadata.CreateMetadataAccountArgsV2{Data: token_metadata.DataV2{Name: "NBA Gen2 Whitelist", Symbol: "NBAG2WL", Uri: "", SellerFeeBasisPoints: 0, Creators: nil, Collection: nil, Uses: nil}, IsMutable: true}
	metadataPda, _ := utils.GetMetadata(MINT)

	ix := token_metadata.NewCreateMetadataAccountV2Instruction(metadata, metadataPda, MINT, oracle.PublicKey(), oracle.PublicKey(), oracle.PublicKey(), solana.SystemProgramID, solana.SysVarRentPubkey).Build()
	txSignature := utils.SendTx(
		"airdrop",
		append(make([]solana.Instruction, 0), ix),
		append(make([]solana.PrivateKey, 0), oracle),
		oracle.PublicKey(),
	)
	log.Println(txSignature)
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
	state := make(State)
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
				state[cell] = Ticket{
					Signature: "",
					Amount:    1,
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

func initStateUsingHashList(hashListFilePath string) {
	state := make(State)

	// fetch owners via token ownership
	var hashList []solana.PublicKey
	hashListBytes, err := os.ReadFile(hashListFilePath)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(hashListBytes, &hashList)
	if err != nil {
		panic(err)
	}

	owners := make([]solana.PublicKey, len(hashList))

	var wg sync.WaitGroup
	wg.Add(len(hashList))
	rpcClient := rpc.New(utils.NETWORK)
	for i, mint := range hashList {
		log.Println(i, len(hashList))
		if i%60 == 0 && i != 0 {
			time.Sleep(1 * time.Second)
		}
		go func(_i int, _mint solana.PublicKey) {
			owner := utils.GetTokenMintOwner(rpcClient, _mint)
			if owner != nil {
				owners[_i] = *owner
			}
			wg.Done()
		}(i, mint)
	}
	wg.Wait()

	type OwnerState map[solana.PublicKey]int
	ownerState := make(OwnerState)
	for _, owner := range owners {
		if _, ok := ownerState[owner]; !ok {
			ownerState[owner] = 0
		}
		ownerState[owner] += 1
	}

	for owner, mints := range ownerState {
		state[owner.String()] = Ticket{
			Signature: "",
			Amount:    mints,
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
	state := make(State)
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

	i := 0
	rpcClient := rpc.New(utils.NETWORK)
	for address, transaction := range state {
		if i%60 == 0 && i != 0 {
			log.Println(i, "2404")
			time.Sleep(1 * time.Second)
		}
		ticket := transaction
		valid := utils.VerifyTransactionSignature(rpcClient, ticket.Signature)
		if !valid {
			log.Println("not valid", address)
			ticket.Signature = ""
		}
		state[address] = ticket
		i++
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
	oracle, err := solana.PrivateKeyFromSolanaKeygenFile("./airdrop.key")
	if err != nil {
		panic(err)
	}

	state := make(State)
	airdropTxMapData, err := ioutil.ReadFile("./report.json")
	if err != nil {
		panic(err)
	}

	json.Unmarshal(airdropTxMapData, &state)

	i := 0
	for address, ticket := range state {
		if i%60 == 0 && i != 0 {
			log.Println(i, "2404")
			time.Sleep(1 * time.Second)
		}

		if ticket.Signature != "" {
			i++
			// log.Println(address, "already recvd")
			continue
		}
		user, err := solana.PublicKeyFromBase58(address)
		if err != nil {
			fmt.Println("bad address", address, "!!!")
		}
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
				SetAmount(uint64(ticket.Amount)).
				Build(),
		)

		txSignature := utils.SendTx(
			"airdrop",
			instructions,
			append(make([]solana.PrivateKey, 0), oracle),
			oracle.PublicKey(),
		)
		if txSignature != nil {
			log.Println(i, len(state), txSignature.String())
			state[user.String()] = Ticket{
				Signature: txSignature.String(),
				Amount:    ticket.Amount,
			}
		} else {
			state[user.String()] = Ticket{
				Signature: "",
				Amount:    ticket.Amount,
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
		i++
	}

}

func airdropTo() {
	oracle, err := solana.PrivateKeyFromSolanaKeygenFile("./airdrop.key")
	if err != nil {
		panic(err)
	}


		user, err := solana.PublicKeyFromBase58(solana.MustPublicKeyFromBase58("CatnSF39Xvg3w7vt7W8LrR7n8BJfzcvf1LCC4FLwqaR6"))
		if err != nil {
			fmt.Println("bad address", address, "!!!")
		}
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

      N := 50
			token.NewMintToInstructionBuilder().
				SetMintAccount(MINT).
				SetDestinationAccount(userTokenAccountAddress).
				SetAuthorityAccount(oracle.PublicKey()).
				SetAmount(uint64(N)).
				Build(),
		)

		txSignature := utils.SendTx(
			"airdrop",
			instructions,
			append(make([]solana.PrivateKey, 0), oracle),
			oracle.PublicKey(),
		)

}

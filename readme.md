# Solana Airdrop Tool v2

# Features

* Mints tokens as airdropped.
* Uses a CSV and JSON file for managing the state of the airdrop.

# Usage

## Required
```
go mod vendor
```

## Init
```
go run main.go init
```
Writes empty airdrop state file from `spreadsheet.csv` to `report.json`.

## Airdrop (restartable)
```
go run main.go airdrop
```
An idemptotent command to sync airdrops amongst the `address`es - declared in `spreadsheet.csv` and initialized in `report.json`.

## Verify Airdrops to Holders
```
go run main.go verify
```
Fetches transaction state from Chain for transaction signatures of `success` in `report.json`.

## Sync CSV additions to State
```
go run main.go sync_spreadsheet
```
Appends the (new) `address`es from `spreadsheet.csv` to `report.json`.

import pandas as pd
import json

if __name__ == "__main__":
    records = []
    data = {}
    with open("report.json", "r") as report:
        data = json.load(report)

    for address in data.keys():
        records.append([address, data[address]["Amount"]])

    df = pd.DataFrame().from_records(records, columns=["address", "amount"])
    print(len(df["address"].unique()))

    df.to_csv("report.csv")


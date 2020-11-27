package main

import (
	"encoding/json"
	"fmt"
)

type tm struct {
	Symbol  string
	Name    string
	Logo    string
	Icon    string
	Website string
}

func main() {

	reg := map[string]*tm{}
	if err := json.Unmarshal([]byte(j), &reg); err != nil {
		panic(fmt.Errorf("sol: %w", err))
	}

	for k, x := range reg {

		n := x.Name
		s := x.Symbol
		l := x.Logo
		w := x.Website

		fmt.Printf("slnc -u http://localhost:8899 token-registry register %s %q %q %q %q\n", k, n, s, l, w)
	}
}

var j = `
{
  "29PEpZeuqWf9tS2gwCjpeXNdXLkaZSMR2s1ibkvGsfnP": {
    "Symbol": "Need for Speed",
    "Name": "Need for Speed",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "2FPyTwcZLUg1MDrwsyoP4D6s1tM7hAkHYRjkNb5w6Pxk": {
    "Symbol": "ETH",
    "Name": "Wrapped Ethereum",
    "Logo": "QmQFLaETa526ZQEE8CbWeEeVZJETqtkQEUXFbVGFTWrrP4",
    "Icon": "/tokens/ethereum.svg",
    "Website": ""
  },
  "2gn1PJdMAU92SU5inLSp4Xp16ZC5iLF6ScEi7UBvp8ZD": {
    "Symbol": "Satoshi Closeup",
    "Name": "Satoshi Closeup",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "3JSf5tPeuscJGtaCp5giEiDhv51gQ4v3zWg8DGgyLfAB": {
    "Symbol": "YFI",
    "Name": "Wrapped YFI",
    "Logo": "QmaRRAKpxRBXUcxMC7SQMe7YrXji2iHbbCgztHu51tyCPQ",
    "Icon": "/tokens/yfi.svg",
    "Website": ""
  },
  "5Fu5UUgbjpUvdBveb3a1JTNirL8rXtiYeSMWvKjtUNQv": {
    "Symbol": "CREAM",
    "Name": "Wrapped Cream Finance",
    "Logo": "QmTfZzMXWvwcNwYJkBQhVprsnefkrud83CP7W89id6moGG",
    "Icon": "/tokens/cream.svg",
    "Website": ""
  },
  "6WNVCuxCGJzNjmMZoKyhZJwvJ5tYpsLyAtagzYASqBoF": {
    "Symbol": "AKRO",
    "Name": "AKRO",
    "Logo": "",
    "Icon": "",
    "Website": ""
  },
  "7RpFk44cMTAUt9CcjEMWnZMypE9bYQsjBiSNLn5qBvhP": {
    "Symbol": "Charles Hoskinson",
    "Name": "Charles Hoskinson",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "7TRzvCqXN8KSXggbSyeEG2Z9YBBhEFmbtmv6FLbd4mmd": {
    "Symbol": "SRM tee-shirt",
    "Name": "SRM tee-shirt",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "7mhZHtPL4GFkquQR4Y6h34Q8hNkQvGc1FaNtyE43NvUR": {
    "Symbol": "Satoshi GB",
    "Name": "Satoshi GB",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "873KLxCbz7s9Kc4ZzgYRtNmhfkQrhfyWGZJBmyCbC3ei": {
    "Symbol": "UBXT",
    "Name": "Wrapped Upbots",
    "Logo": "",
    "Icon": "",
    "Website": ""
  },
  "8RoKfLx5RCscbtVh8kYb81TF7ngFJ38RPomXtUREKsT2": {
    "Symbol": "Satoshi OG",
    "Name": "Satoshi OG",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "8T4vXgwZUWwsbCDiptHFHjdfexvLG9UP8oy1psJWEQdS": {
    "Symbol": "Uni Christmas",
    "Name": "Uni Christmas",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "91fSFQsPzMLat9DHwLdQacW3i3EGnWds5tA5mt7yLiT9": {
    "Symbol": "Unlimited Energy",
    "Name": "Unlimited Energy",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "9CmQwpvVXRyixjiE3LrbSyyopPZohNDN1RZiTk8rnXsQ": {
    "Symbol": "DeceFi",
    "Name": "DeceFi",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "9S4t2NEAiJVMvPdRYKVrfJpBafPBLtvbvyS3DecojQHw": {
    "Symbol": "FRONT",
    "Name": "Wrapped FRONT",
    "Logo": "QmT4KeUcrfwEZ6zzSHBMDLDVHqBQ7wFUX7hX6QKJ5ow6nh",
    "Icon": "/tokens/front.svg",
    "Website": "https://frontier.xyz/"
  },
  "9Vvre2DxBB9onibwYDHeMsY1cj6BDKtEDccBPWRN215E": {
    "Symbol": "Satoshi Nakamoto",
    "Name": "Satoshi Nakamoto",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "9n4nbM75f5Ui33ZbPYXn59EwSgE8CGsHtAeTH5YFeJ9E": {
    "Symbol": "BTC",
    "Name": "Wrapped Bitcoin",
    "Logo": "QmUZtg1nYAx9GwVyg9fMvWf7pdsTgofYvueWxTRAVuWM8M",
    "Icon": "/tokens/bitcoin.svg",
    "Website": ""
  },
  "9rw5hyDngBQ3yDsCRHqgzGHERpU2zaLh1BXBUjree48J": {
    "Symbol": "Satoshi BTC",
    "Name": "Satoshi BTC",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "AGFEad2et2ZJif9jaGpdMixQqvW5i81aBdvKe7PHNfz3": {
    "Symbol": "FTT",
    "Name": "Wrapped FTT",
    "Logo": "QmZNaGF3DzghDKxHtq75Ybrx59EaAbmh5CUo341hfUuzZe",
    "Icon": "/tokens/ftt.svg",
    "Website": ""
  },
  "AR1Mtgh7zAtxuxGd2XPovXPVjcSdY3i4rQYisNadjfKy": {
    "Symbol": "SUSHI",
    "Name": "Wrapped Sushi",
    "Logo": "",
    "Icon": "",
    "Website": ""
  },
  "AcstFzGGawvvdVhYV9bftr7fmBHbePUjhv53YK1W3dZo": {
    "Symbol": "LSD",
    "Name": "LSD",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "AgdBQN2Sy2abiZ2KToWeUsQ9PHdCv95wt6kVWRf5zDkx": {
    "Symbol": "Bitcoin Tram",
    "Name": "Bitcoin Tram",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "AiD7J6D5Hny5DJB1MrYBc2ePQqy2Yh4NoxWwYfR7PzxH": {
    "Symbol": "Satoshi GB",
    "Name": "Satoshi GB",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "BQcdHdAQW1hczDbBi9hiegXAR7A98Q9jx3X3iBBBDiq4": {
    "Symbol": "USDT",
    "Name": "Wrapped USDT",
    "Logo": "QmVhrTZpkM1TeHk4WZqYmrNqHiwK3Xr22v5tyDHdmH1NE5",
    "Icon": "/tokens/usdt.svg",
    "Website": ""
  },
  "BXXkv6z8ykpG1yuvUDPgh732wzVHB69RnB9YgSYh3itW": {
    "Symbol": "WUSDC",
    "Name": "Wrapped USDC",
    "Logo": "",
    "Icon": "",
    "Website": ""
  },
  "BrUKFwAABkExb1xzYU4NkRWzjBihVQdZ3PBz4m5S8if3": {
    "Symbol": "Tesla",
    "Name": "Tesla",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "BtZQfWqDGbk9Wf2rXEiWyQBdBY1etnUUn6zEphvVS7yN": {
    "Symbol": "HGET",
    "Name": "Wrapped Hedget",
    "Logo": "",
    "Icon": "",
    "Website": ""
  },
  "CWE8jPTUYhdCTZYWPTe1o5DFqfdjzWKc9WKz6rSjQUdG": {
    "Symbol": "LINK",
    "Name": "Wrapped Chainlink",
    "Logo": "QmTSGr6FGV3BfUgghqBTgLviTDHTTjejJd86Upg6v6NEiD",
    "Icon": "/tokens/link.svg",
    "Website": ""
  },
  "CsZ5LZkDS7h9TDKjrbL7VAwQZ9nsRu8vJLhRYfmGaN8K": {
    "Symbol": "ALEPH",
    "Name": "Wrapped Aleph",
    "Logo": "",
    "Icon": "",
    "Website": ""
  },
  "DEhAasscXF4kEGxFgJ3bq4PpVGp5wyUxMRvn6TzGVHaw": {
    "Symbol": "UNI",
    "Name": "UNI",
    "Logo": "",
    "Icon": "",
    "Website": ""
  },
  "DJafV9qemGp7mLMEn5wrfqaFwxsbLgUsGVS16zKRk9kc": {
    "Symbol": "HXRO",
    "Name": "HXRO",
    "Logo": "",
    "Icon": "",
    "Website": ""
  },
  "EDP8TpLJ77M3KiDgFkZW4v4mhmKJHZi9gehYXenfFZuL": {
    "Symbol": "CMS - Rare",
    "Name": "CMS - Rare",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v": {
    "Symbol": "USDC",
    "Name": "USD Coin",
    "Logo": "QmPnURDGTKphc81GbFRCRiWFPK6P6dNhMXEso42b5oAqUK",
    "Icon": "/tokens/usdc.svg",
    "Website": "https://www.centre.io/"
  },
  "EjFGGJSyp9UDS8aqafET5LX49nsG326MeNezYzpiwgpQ": {
    "Symbol": "BNB",
    "Name": "BNB",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "EqWCKXfs3x47uVosDpTRgFniThL9Y8iCztJaapxbEaVX": {
    "Symbol": "LUA",
    "Name": "LUA",
    "Logo": "",
    "Icon": "",
    "Website": ""
  },
  "F6ST1wWkx2PeH45sKmRxo1boyuzzWCfpnvyKL4BGeLxF": {
    "Symbol": "Power User",
    "Name": "Power User",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "FkmkTr4en8CXkfo9jAwEMov6PVNLpYMzWr3Udqf9so8Z": {
    "Symbol": "Seldom",
    "Name": "Seldom",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "GUohe4DJUA5FKPWo3joiPgsB7yzer7LpDmt1Vhzy3Zht": {
    "Symbol": "KEEP",
    "Name": "KEEP",
    "Logo": "",
    "Icon": "",
    "Website": ""
  },
  "GXMvfY2jpQctDqZ9RoU3oWPhufKiCcFEfchvYumtX7jd": {
    "Symbol": "TOMO",
    "Name": "TOMO",
    "Logo": "",
    "Icon": "",
    "Website": ""
  },
  "Ga2AXHpfAF6mv2ekZwcsJFqu7wB4NV331qNH7fW9Nst8": {
    "Symbol": "XRP",
    "Name": "Wrapped XRP",
    "Logo": "QmZEVjP8Jd2CTX9Tau8LEHdgv42gCekkRW1EqAE7xujQmk",
    "Icon": "/tokens/xrp.svg",
    "Website": ""
  },
  "GeDS162t9yGJuLEHPWXXGrb1zwkzinCgRwnT8vHYjKza": {
    "Symbol": "MATH",
    "Name": "MATH",
    "Logo": "",
    "Icon": "",
    "Website": ""
  },
  "GoC24kpj6TkvjzspXrjSJC2CVb5zMWhLyRcHJh9yKjRF": {
    "Symbol": "Satoshi Closeup",
    "Name": "Satoshi Closeup",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "GyRkPAxpd9XrMHcBF6fYHVRSZQvQBwAGKAGQeBPSKzMq": {
    "Symbol": "SBF",
    "Name": "SBF",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "HqB7uswoVg4suaQiDP3wjxob1G5WdZ144zhdStwMCq7e": {
    "Symbol": "HNT",
    "Name": "Wrapped Helium",
    "Logo": "",
    "Icon": "",
    "Website": ""
  },
  "HsY8PNar8VExU335ZRYzg89fX7qa4upYu6vPMPFyCDdK": {
    "Symbol": "ADOR OPENS",
    "Name": "ADOR OPENS",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "MSRMcoVyrFxnSgo5uXwone5SKcGhT1KEJMFEkMEWf9L": {
    "Symbol": "MSRM",
    "Name": "MegaSerum",
    "Logo": "QmZrGLw9GecLxrwCv67rXb3YPEfYHz6uTCnzQzneszv8e3",
    "Icon": "/tokens/serum-32.png",
    "Website": "https://projectserum.com"
  },
  "SF3oTvfWzEP3DTwGSvUXRrGTvr75pdZNnBLAH9bzMuX": {
    "Symbol": "SXP",
    "Name": "Wrapped Swipe",
    "Logo": "/tokens/sxp.svg",
    "Icon": "/tokens/sxp.svg",
    "Website": ""
  },
  "SRMuApVNdxXokk5GT7XD5cUUgXMBCoAz2LHeuAoKWRt": {
    "Symbol": "SRM",
    "Name": "Serum",
    "Logo": "QmZrGLw9GecLxrwCv67rXb3YPEfYHz6uTCnzQzneszv8e3",
    "Icon": "/tokens/serum-32.png",
    "Website": "https://projectserum.com"
  },
  "So11111111111111111111111111111111111111112": {
    "Symbol": "SOL",
    "Name": "Wrapped SOL",
    "Logo": "",
    "Icon": "",
    "Website": ""
  },
  "bxiA13fpU1utDmYuUvxvyMT8odew5FEm96MRv7ij3eb": {
    "Symbol": "Satoshi",
    "Name": "Satoshi",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "dZytJ7iPDcCu9mKe3srL7bpUeaR3zzkcVqbtqsmxtXZ": {
    "Symbol": "VIP Member",
    "Name": "VIP Member",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.com"
  },
  "oCUduD44ETuZ65bpWdPzPDSnAdreg1sJrugfwyFZVHV": {
    "Symbol": "Satoshi BTC",
    "Name": "Satoshi BTC",
    "Logo": "",
    "Icon": "",
    "Website": "https://solible.fire"
  }
}
`

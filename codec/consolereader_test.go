// Copyright 2019 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package codec

import (
	"encoding/hex"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"io"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/mr-tron/base58"
	pbcodec "github.com/streamingfast/sf-solana/pb/sf/solana/codec/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_readFromFile(t *testing.T) {
	testPath := "testdata/syncer_20210211"
	cleanup, testdir, err := copyTestDir(testPath, "syncer_20210211")
	require.NoError(t, err)
	defer func() {
		cleanup()
	}()

	cr := testFileConsoleReader(t, fmt.Sprintf("%s/test.dmlog", testPath), testdir)
	s, err := cr.Read()
	require.NoError(t, err)

	block := s.(*pbcodec.Block)
	spew.Dump(block)

	// TODO: add more testing

	//assert.Equal(t, &pbcodec.Block{
	//	Version: 1,
	//	Id:                   "FIXEDSLOTIDBECAUSEWEDONTNEEDITANDCODECHANGED",
	//	Number:               4,
	//	PreviousId:           "7Qjov8K99CSYu29eL7nrSzvmHSVvJfXCy4Vs91qQFQAt",
	//	PreviousBlock:    3,
	//	GenesisUnixTimestamp: 1635424624,
	//	ClockUnixTimestamp:   1635424623,
	//	RootNum:              0,
	//}, block)
	assert.Equal(t, "5XCVxTsM4u6i3AWz9duzzhWH7vwHukUeXrrbEskhQa9U", base58.Encode(block.Id))
	assert.Equal(t, uint64(1), block.Number)
	assert.Equal(t, "D9i2oNmbRpC3crs3JHw1bWXeRaairC1Ko2QeTYgG2Fte", base58.Encode(block.PreviousId))
	assert.Equal(t, uint32(1), block.Version)
	assert.Equal(t, uint32(1), block.TransactionCount)
	transaction := block.Transactions[0]
	assert.Equal(t, "4ctVmqXREqTutvFETgjfsZdXW3Q2kzHbmA9jecfP9z1FGy19VjNQXVqqc9HquimieXYFFWrmEKxTrYcww8ZjySwd", base58.Encode(transaction.Id))
	assert.Equal(t, 1, len(transaction.Instructions))

	s, err = cr.Read()
	require.NoError(t, err)
	block = s.(*pbcodec.Block)
	spew.Dump(block)

	s, err = cr.Read()
	require.NoError(t, err)
	block = s.(*pbcodec.Block)
	spew.Dump(block)

}

func Test_processBatchAggregation(t *testing.T) {
	b := &bank{
		transactionIDs: trxIDs(t, "11", "aa", "cc", "bb", "dd", "ee"),
		blk: &pbcodec.Block{
			Id:     blockId(t, "A"),
			Number: 1,
		},
		batchAggregator: [][]*pbcodec.Transaction{
			{
				{Id: trxID(t, "dd")},
			},
			{
				{Id: trxID(t, "ee")},
			},
			{
				{Id: trxID(t, "bb")},
			},
			{
				{Id: trxID(t, "aa")},
				{Id: trxID(t, "cc")},
			},
			{
				{Id: trxID(t, "11")},
			},
		},
	}
	err := b.processBatchAggregation()
	require.NoError(t, err)
	assert.Equal(t, trxSlice(t, []string{"11", "aa", "cc", "bb", "dd", "ee"}), b.blk.Transactions)
}

func MustHexDecode(s string) (out []byte) {
	out, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return
}

func Test_readBlockWork(t *testing.T) {
	parseCtx := &parseCtx{
		banks: map[uint64]*bank{},
	}
	parseCtx.readBlockWork("BLOCK_WORK 108227405 108227406 partial 4TbxQJpq7MT843rFzLwR3yLKcsRiLi7E5TQi3TD8LR5A 97455401 59 89 56 0 0 0 4TbxQJpq7MT843rFzLwR3yLKcsRiLi7E5TQi3TD8LR5A 0 T;668tfTrSGexUzimCftbizyuYqSeoEQPuE6QRmz6XCddtCxhBZm36Eaeid7EaDGCeMHXDenBHFKVRr7Djgzvk82hf;4xdihXZj7c9xTCTSS18PJZuSZ6GzzsxfZBqR2xiYU1XzSjx9wYxPoN4QQoqDUckSDVLVh5PzD7WUoxHfjnUqQ44r;5ZqiuFuyszM2G535noYpjHN9k6GQ2SuwVzAkQdXYx5uW9bFB5ujByaVTpKqDixbQmpxiRC4EuBBphKByKrYhFx9i;2gzbmbhUPSvV3H4EmG5E8MgKbiNRfcRoTVX7n7yqiB1Evhgpw5xHvk2KasRJ4hDfE3dhdzA2CPWcqLkyKmqxxH8d;5ZofEU5Hx5L7yTiRNAqgHo2e6R15BHvg1nXjnP9jyYJHL42BPw9ZwaAfmif4WwPxzQnix62JhXHABZYqLkeeNcr5;63zmRVeVt6ooveRpixkMphj2rMAMuZujMccSHwNe3iaJRDogtsZdpv5SFiaJnV8mkZzowHsGgLCunkcUnZ6KPPDE;2uxnJAJ8cxb3FcLXH92ezHfh6T2Row7QYtTF8Y5E2m4YpcUZDriYRnHixtDYdC8bvkf51zccRjCsv18RFdiVhUtL;5BwmHtuPcivoRK4bX91rmf9N7uNAnwu1WTki4zxezrb7NhLWPBywLRsAbp845TXZs2jPsjehWx7MGLPaFLK6mtta;37rnA3CEpdtJXR8YndYMAYVBwBw9UXAdRZQtd8Z3zzMwbNoofLLMetRTJUTLGWBGx6vhSw5WQyuesgzAaHdaRH5L;wJ6fDFUrQ1gDrpqpykBdswq7etvfaT7Y93ja9v3szuCAKsSDNqV8UCjk4zdhiqzqotqLUKwSqGPUnkXHUCbeWiX;MYvJEmKDyfbY6ZixpSAvXur3jJ3eA18K4U717CTWFVh45bs51r5myyUh1sbSNxRuEgLp8Z6iCzZggddtK8WCRDG;JWRSzMnHWFAUU8qbp2vMYvwEhv2dyxzEstXEAxFgC49WDr3kjD6MryvMzMnTyvyWrEzdwud2tnFewYrPAgAQNrk;2TvPWJ7RFiJUHYSEgHEMueDBd3Fj1gLpuFC5PF21wpo9uyyeuj3t57DUJsoAvSfP2PtynC6o8suppDhNc1D5KYH2;5zYKfMR8bLd9ZgFSLWDRiw2A69oEZtxYic6aeHkHSoJPi63YpyA9aWAqkozNyk8de3eEBtxPvAJjMMEqu9xThgwN;2SGyVco2nnr2Hz2uaV9bZYx1ckLHueKUDm1tcF7ssStvbcTaTbJjDrgEEzjudrS2Lcp1jpDAre82imUvkYYzdkKL;2PJHpZy2mnzFWGTf9QwiU8pciZVGjHun8jankMocY8DReFnMeLMLw6qX2JPWcDLc8fNrZMhiQQTMVaErdop2iPxX;3sngyaPw8GFM5Yhx76iEDt3bxNkn4Yn4vfzoxy8sr4mzbCZRtqtsybyrfPWFDjQCAmAwpv9ScLpZKrmBh5wNtgRL;3ecpQQUu3k236mbtKfUrBBVV6vaigzjBn8wYVhNkVSKRaYz9wErHoKMDk3LKzr3D6oh7TZxQx1FUGt8Ya7arGYCj;2k8d1bm1DWkWd9NWNkVUvRcccpWDmWm4pQLNyCoiSDPupMbUuYR7ddz9MCjF29GWnmpNYqPQKhDJkkXxfo2o4tsB;2LoPf3n8r2TBvxQUrQZNxqhAe5Dc2kV2fUXZd3dSxyX71zRg6GTQjzJmByLeWRDyZDwpoLJXhr6ZgEE66D3VFrBu;5eFxbm1trKtowgGTzoQzgmvKGWqTAviKgBeVoN8QYp88k56GfhvxzFtgdgrbvffvdkPTvgiW8DtU2ZQL7H2oc9Dv;2WXhithCc7pdnpZMEj5UFubkAzKFiT4PLHmtFLeEwDWJtyzXLs3dpQGvofWGLEsqQnWL4iJeeFmf9LXaMAd2dor8;5xaWH3WRMPEvC3Yhdghtyte8EgxBo8eD8tGKSNS4A8dsqydAB6cQBfiywvUZ1pjmyu6Ho6w8YLcoEvLb5v1wPUvk;4L3kzmzmJzd18x8H6x4AQWM2e8tnFDLUEaGupVzR55fspCbNASWoSVrHT3xPafXNfpgU7RqrztBNE9UMufB94ay1;PvR74QUmQk4MitsAEtutHjkf68E8dFYWq2nNcHyKisvCQKzJFdDFCMSriX2LtcfTVNtJB7c669UdU2FAi6s54m5;4hf8yVjf8yZ1ps31LEpwWcFUXoDb3aSpXkSnLRu7ZVg37w6s1kTM7mb1byJ1edsaNymGFkdoZYFfVxfGNuNThsGT;2iSUv5Rp6XzJCz7vDUtiULyv22wgjekYekKQtY6hcYuJ9c2eDV3jpUepJNQvoYZvg2xHSbcoWpdfx68KMcok9PRM;2GLZPb8scxqJSkHwC9nsaA1aZa968X7W5o9reZ5b8DYNiYgsUSXDwgxLARMYxrU3a6humpUiKK6X8wdHg5pMHP6v;GfbjFvP4Uvy2YVKpmaUaqa717CuieqXmyEbMP4TDW42oQQ3kpuEicCRrGVDhecuVqB7bZqXtZBLiTCti2gihDG2;a9yXmpJWnaSxJ3bR8NvhAc9yu1BTFYabgBAw723WoQhnCnPP1cPoMuvSXfW5ajvKAbz9PTVMdEfvJqGY1sz3TbD;7eXbdfsiEEEcX5EpwwExJiroHtumr7P53b1aVaEDQGzd2m33VNQJrYiWkdcKDYFKxd3zTykwACWFkdA8WzyNwnS;22o6WQSYmDaXRLoe8JdQyUhr4hv1FStoyiV1LDnRu4WaS1iVZduqiLaDZD8KS5T5Ccjd8wB4yP7NVkDAdASWtPi4;2XSazm3XNTfSH6Vtg3DGUvu3iTLa4zKpocLQ3MU4eydC2mN3qXh5gGM7xYcSYcfeDXSttMMa8bMPrjxEDBkGdUro;4K92S9ZJNvgAdkgwtuR8qUXSB9pLEozYGPPoiFJ1QVyXn7KCwp9usf5JrBRxh7BREZ8xgK8nDUFkb5ZMUHzv5xdG;5zXKFY9jW8VuzhS8RSLF7i9kusVAKZ9zrY5eEzhTpssUNHFpF5aCMnwFRgRpaEWp2TKwYFKBggvRYkQDyNuZ8tA1;4ySDmzYGCYProtCphszeuTK2PcjbVM9mZdYTjmUQ8sS3BN7VYECofDSv5sdmJdMFSmof34MdWimFxo1RJDBHZCpL;4rMZpXD6rfdMHKX79Z3LfKVZM23GS7QYNzYzHBxzL9qKGxfpX26Hp3mQBQNbtt3EKTbD7MAbKGM3qu6gowPdQHap;2Y7zNTwoGXQVTzkXNpwVR7LRGcpbfBDZHHqj5GEwBf8Q4zBTqHH25SoVGZJui18kkaYtLgwQXnuNTTXmXAkMdiUY;3pH69afRwXJF1kSMne7xbyoAmkibr3THLBavJaRUmMKwRWwvekPoLJLk5d2PuCJ2eW9ickfuCN2ytdrdnaPiMjkr;34M2hKQnZQgvzmak6rLNeHkGQbEp4WqbU9txhWcyMRqz6PBnuBTLjr1zJ4ee1uWLZAFmbhwb8mwcTNgdzs5JuHAN;57wbxKmjTa9DUiDHU77uwmEwXYDPG1nbDF14LM2q2bFXftV4LKTBSNGmFx5kacYjktkbNHFZwb2FQsDXxLd9jrZL;5rmSYiKYFoqYovrrgigtrhkkwz478yS2KTs9RePP8z2v9Jjek2JNcin2HdQAzt2RwAE6Ni2ATVdhCRHFod4zaE8C;22e8e3Gd9wzyGpK8oTe72jjHr5uN1eH6pgYa5ZkwEzA4dA1hVuBt6vYKb4MCQnVKYcSxyaJdWRTws1uWbZdnA1Kv;3TDJj3TNeBtmwCBmVZkAWnzMYCugEDgm9vAQhyz6aRM2HpeDqjCBUQ9VQN2qCuEZaMCg8FTjNpZ1JiNXvPJdUFTR;GKuJJUxGeSFA9xfhrJLkAaUgvGjxetsiLLCXaTTHpXutoKnHM15kA2h8B9YeyU7ZUw6pnFtAUTx6MtrdPAa1zpt;4psBf8sJMzAsPpKaDNfuaY1BRykSE7T9rArkQnHfq4cc8PTcJWpr3f2kzjgjzzZmmgjoKp9eYdcovzuyqjueCwLY;3idfGKuGvNsL4btjwZMxrQUs9guozqkWNW1Jfp1hQ49UJupudnQuC54SZLXr5ueBLEsQ4R5rLeLdo65TGM6Upqor;3s8UFvWwCTvVdp7JouFoximWTnVUKyZJRS7DWbDzANS4AsB3r1N8FHY4uYxHVj6q3RhTwTfDzaqmkbNDj2voJPXu;mojuR9vCpa8BTuVKw72tQzA2KSnkBtcLjBTSTwtpW4sAneDodAmBpDUodZHm7n19DKhCCyTxmX2SHTGMwZtrwYB;35VxMkqAe56EVpQXXvpK4BLnT6aAc3UH6Xx3jo4PVtoeZwveHXtQaV8KZkdtafpHVdvD9WBzX9Qs3TBwhxsAiSM9;uC9RuAqagQzb244bMej8sJYbmEkJP4Mqm1vaJBFsm1qGqmy2Ek3x8gmhaCyxZkwUKJsKSqpmghBUSwnfc5uocLR;2bzVNGBmAXDGMNUbeytzvAJY9EtxaxK8jDmssnev7EVuGk9LXuVh67uQV8rEcqL2ZfySCgPNQ76ULULWvYYheJSR;5z711SXSiP2RydvcGy9mvvDU6ceRJgecaEVbENC8PGeWuuTbGgDpXmJkMwDftnHttYJzpWgyxXsYpGyRGowmdcXt;4UWUzU3Cp3x5xCNWKLA3unMqcFvF2NnghzHu7CwnEEZVmgcThSQw4LjqDhjKx5WvBbks5VqNRq5E6ANwbMgrCard;Lve7sFokh224HDNEYZqf1TntagRA18JmdeNnF6uoNve1EnFTSHZ4DtWVo4sUjSXaaCFtCDU8QXu8jsdLuyXY1Sn;59aUNSMsLePvpxf7DwAJgUM7ZQ8UEp1FdVrxF8BicQFisQmH1x7zBTKReYtvAUyFmGUmfdqsP8631em5dk7H1bHZ;3Ns2fW5216CebGthy1TfNXeRYb3GwRVo3rdawe3nAxBx6rRxfKeBkv6iHDBJA64NcsbuqxgtdgGTqEyd85p9tcu1;5SWGqkYpJ3S9FbV3DKUt7faB1u6SutVYJZkZj68jCun4MVGEHhEH4J1wvLN1CF4tV6HzzgvGXFLP1S2mZCQbfTPw;2UckTkgm7JM4vimoRJasQpqfL6naNqcLGryaHxAFSfAtVug89uk7k5VYSjvCmh3MC9aWHRB2TKwkvzn82YYrxZ6a;4VSrUWuNJv9cTkAXiJnQQwSF7tpadPsLjx89wyGJnBU27rtxxKkakVDwUw6VuWTi6Ni4Vtz3SqAUGiWSjnQgE28y;38gbDWPHtNvQMm9VgpJoRDZpgHtQiYz8Po54cbuxj7eHHmPexasN6sfZ9eepieHmscVkGbbLgqjYC3iA99LQfTcs;2HXTPKGLqTxyYxxAvbHCMyuSA8XzNotMP4LAf9a1UabfkAohLGZG8GYNx5h8uh2Sv7oJNmtumsNRTXsERNzJxPG4;28xgBh1g2LPsnfassTEd2fEZHXrzRmvJhrT95eVXfx6QYu33KkgVt82nKQR3uWEyBfaZsUZsBPzPcpv5QspSobhB;4ZktRiC9eDRv7W9RduSL1tZRMyK6H1DG59a2n8Lgps6vfzM2hEsLLX8x28iqgAz4v8Bj3nC11CMxGJdFR1uY2hDS;5woi4YbMmgxyjTa4b2XsbHj9ByVAiuQ4y4rAAoTczhx2Wjti1FUUymD5GMaENGnpLYkyZZXmM98Bd3D7oYjmAAew;4Voct1cDCnwiKq5TLfrrTKFMHmdMmeujtGdufD9Kjyx3JwHrxd6u7u6toB5DbEysvisMX5GSpPD4vea5TcaQbmtm;39c23pbstSN1fwmkbD63iV3yLew18RDi2JQtaxbkcCynFYQKa1TsanuaEfCmwGf7iq9SugBpPoXabT6odBtZ6U1S;CrLowTHETYoWpLmnJAiyhSj2sbHjKfbY6xFkaR1YxGaUGmSwzMPGM9fZj8Qs7ypES1Q629NcHRbb1Hp8sR5CRxu;59htB9qCAKfiwPghH8YzpT31xAp2C81xiGD2tWFBGGnGccwH8JeTsbpJxDzBXhbjmYhPQp2Hx6SjE6iv4zREJjuT;3kMv1u77kvZMYKcbN2EMNd9zoQYFKMKrRm2FoupkaPFydRYt47g3CYZPyfyPDDXeWwxGZuSrdrqRqL8xUV5Fon3o;2WqUxH9ikrhidxGPjp7bpHcHknfFuYdTjEB4RR5uicicZ9ZJuhxV67vLNz2s1ZYspZ46JvdwLsPJxTEZhiXCfDMi;3PsKf9o6LE2TzNrGzZBbLtB3aVA8UpsJpLx8v1D4LcTVb3tQfaR2GWx2UFjYg4iunPYeK7qz3LYSFzxQ5eye6z45;2tWGEUB8YrPztJy41ekLmbYwxsgaKKcEYRUT2fWqFM6zJ129t9qyk6UNyECT5PJPgnJ4p1Ji65ceqhBjfj4z8e94;2J5GE1agjhnJ1sUaahDXTgA8V9omAF3bHr2sWBe6NLEg7CwECspG1SfCx1y9DfLDZXbAqCNPjBwauvMM2VPDM98n;4EiDavNDQY683YKnpDtfX3kvSnLHpVz7Fa913NcXNJy3LsoDd8CNHT44RiE1XWRfDqsDatru3uRp2meDfmtEfDCw;2ZUXGnJhDa9PVsWyNkDh4NzgTd2J4xewgwMEDgawXjfdo894ZEMdVFUih8HCCP7HB5P4qra2PwyZkdaoQo7f6A4m;2Lo9KGbEtahtSj3jLnzubUH831PFr68YqQjQyxQS5Mm5qmRu78qmEA4HjbGYGntpgydJp8QpjL4Entr2WXpL8Yqc;5LB3bKA9gBzu9SQEzPSo1WZRqZGbtonm5nsGuZuihnLigwKfR3S9bbrjStxpfCEHZZa6mdat5Lz2vbMzG952Y34h;Bp6PAntbMEaippQXFqeDzBq3RwWFYf1xmX56xti5g8DTGWGbhZqCADH2rGNQMFwEni3bkFGYiRUbuMEaUFPzPov;3mSKaXvj7YNuLiawdQZ2XNnCDqa5noMPju6m2Z2Thv171ad6omMprtsNBHAUFRn4ay5EMscWBfp3CRosys2UBHwq;4vPURw2Kd9g3rjuBTpv7t2T71d9HuLFxWwFCuUabwu8xu7GwQAv3UhWt1jVnHj4DPs19V9teve7rmMc8hHTTeFVP;2XbRwK5yV2wK1RDSafoxVUiyx1y1YCzGFSDr2ccixsUb8ds9RkUvx735PJHKgt5QMEyjacCYQz1UAvgrmpLa13PH;41yBCYD5n2CDtHpcMAzmefKfVNL3kBXRz1RLGUQmat2GzKVsjsDV7QtKeeUbzcdt68WhxVKLkbhKvavhfzeH9F7J;4Fs7WatksKuKLovvth4T5mirhCEpLwSLBuFABnQKBKhqfhU3jfZHhpuW1DW3Lsxa83wWPb76GipdS26XyV5RA2x5;avHQMQLAc2KUEQ2VYJfzHsk23zbHqjYLkcLM2EbmJpyBZkPsyRE6xfYzLnTJAEYJnB39AgU5fTvasxWH6VKy3mm;4TnP72QHzk5asFDq7n6K1sdUTnNseiqgypYoATBSGuLQaaFRHsRbEMJj9iHibPSRQMqGTjRsSDQAT7TN1cb1udSf;3pyxch7xmn4w89QCrqWPcb4gC5dciCmidSbNeymq5MfVRDebN1ctFLW1Uqd49us9Nz5MoYigFBq7bkdoTg4qD6bd;3C2ZtwF91WMQpR28e5ujVagePxUrz67NN5raAXBLY6mx8RYL7sitJRXR29S4PRWUP4gt4cqb2eWzax5PgMdmXdGh;X7ywH8mi7KRgpLszNgonoHWWTvUWgQbEp4nVu5AaFJqgrTJM9HtbmkL8VdWqi4QN6F1ogGfKK5QF3D5mZefff1z")
}

//func Test_readBlockWork(t *testing.T) {
//	t.Skip()
//	tests := []struct {
//		name       string
//		ctx        *parseCtx
//		line       string
//		expectCtx  *parseCtx
//		expecError bool
//	}{
//		{
//			name: "new full slot work",
//			ctx: &parseCtx{
//				banks: map[uint64]*bank{},
//			},
//			line: "BLOCK_WORK 55295939 55295941 full 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 932 814 526 0 0 0 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 0 T;aa;bb",
//			expectCtx: &parseCtx{
//				banks: map[uint64]*bank{
//					55295941: {
//						parentSlotNum:   55295939,
//						batchAggregator: [][]*pbcodec.Transaction{},
//						previousSlotID:  blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//						transactionIDs:  trxIDs(t, "aa", "bb"),
//						blk: &pbcodec.Block{
//							Version:       1,
//							Number:        55295941,
//							PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//							PreviousBlock: 55295939,
//						},
//					},
//				},
//				activeBank: &bank{
//					parentSlotNum:   55295939,
//					batchAggregator: [][]*pbcodec.Transaction{},
//					previousSlotID:  blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//					transactionIDs:  trxIDs(t, "aa", "bb"),
//					blk: &pbcodec.Block{
//						Version:       1,
//						Number:        55295941,
//						PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//						PreviousBlock: 55295939,
//					},
//				},
//			},
//		},
//		{
//			name: "new partial slot work",
//			ctx: &parseCtx{
//				banks: map[uint64]*bank{},
//			},
//			line: "BLOCK_WORK 55295939 55295941 partial 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 932 814 526 0 0 0 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 0 T;aa;bb",
//			expectCtx: &parseCtx{
//				banks: map[uint64]*bank{
//					55295941: {
//						parentSlotNum:   55295939,
//						previousSlotID:  blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//						batchAggregator: [][]*pbcodec.Transaction{},
//						transactionIDs:  trxIDs(t, "aa", "bb"),
//						blk: &pbcodec.Block{
//							Version:       1,
//							Number:        55295941,
//							PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//							PreviousBlock: 55295939,
//						},
//					},
//				},
//				activeBank: &bank{
//					parentSlotNum:   55295939,
//					previousSlotID:  blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//					batchAggregator: [][]*pbcodec.Transaction{},
//					transactionIDs:  trxIDs(t, "aa", "bb"),
//					blk: &pbcodec.Block{
//						Version:       1,
//						Number:        55295941,
//						PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//						PreviousBlock: 55295939,
//					},
//				},
//			},
//		},
//		{
//			name: "known partial slot work",
//			ctx: &parseCtx{
//				banks: map[uint64]*bank{
//					55295941: {
//						parentSlotNum:   55295939,
//						batchAggregator: [][]*pbcodec.Transaction{},
//						previousSlotID:  blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//						transactionIDs:  trxIDs(t, "aa"),
//						blk: &pbcodec.Block{
//							Number:        55295941,
//							PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//							PreviousBlock: 55295939,
//						},
//					},
//				},
//			},
//			line: "BLOCK_WORK 55295939 55295941 partial 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 423 814 526 0 0 0 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 0 T;bb",
//			expectCtx: &parseCtx{
//				banks: map[uint64]*bank{
//					55295941: {
//						parentSlotNum:   55295939,
//						batchAggregator: [][]*pbcodec.Transaction{},
//						previousSlotID:  blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//						transactionIDs:  trxIDs(t, "aa", "bb"),
//						blk: &pbcodec.Block{
//							Number:        55295941,
//							PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//							PreviousBlock: 55295939,
//						},
//					},
//				},
//				activeBank: &bank{
//					parentSlotNum:   55295939,
//					batchAggregator: [][]*pbcodec.Transaction{},
//					previousSlotID:  blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//					transactionIDs:  trxIDs(t, "aa", "bb"),
//					blk: &pbcodec.Block{
//						Number:        55295941,
//						PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//						PreviousBlock: 55295939,
//					},
//				},
//			},
//		},
//		{
//			name: "known full slot work",
//			ctx: &parseCtx{
//				banks: map[uint64]*bank{
//					55295941: {
//						parentSlotNum:   55295939,
//						batchAggregator: [][]*pbcodec.Transaction{},
//						previousSlotID:  blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//						transactionIDs:  trxIDs(t, "aa"),
//						blk: &pbcodec.Block{
//							Number:        55295941,
//							PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//							PreviousBlock: 55295939,
//						},
//					},
//				},
//			},
//			line: "BLOCK_WORK 55295939 55295941 full 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 423 814 526 0 0 0 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 0 T;bb",
//			expectCtx: &parseCtx{
//				banks: map[uint64]*bank{
//					55295941: {
//						parentSlotNum:   55295939,
//						batchAggregator: [][]*pbcodec.Transaction{},
//						previousSlotID:  blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//						transactionIDs:  trxIDs(t, "aa", "bb"),
//						blk: &pbcodec.Block{
//							Number:        55295941,
//							PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//							PreviousBlock: 55295939,
//						},
//					},
//				},
//				activeBank: &bank{
//					parentSlotNum:   55295939,
//					batchAggregator: [][]*pbcodec.Transaction{},
//					previousSlotID:  blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//					transactionIDs:  trxIDs(t, "aa", "bb"),
//					blk: &pbcodec.Block{
//						Number:        55295941,
//						PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//						PreviousBlock: 55295939,
//					},
//				},
//			},
//		},
//	}
//
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			err := test.ctx.readBlockWork(test.line)
//			if test.expecError {
//				require.Error(t, err)
//			} else {
//				require.NoError(t, err)
//				assert.Equal(t, test.expectCtx, test.ctx)
//			}
//		})
//	}
//}

func Test_readBlockEnd(t *testing.T) {
	tests := []struct {
		name        string
		ctx         *parseCtx
		line        string
		expectCtx   *parseCtx
		expectError bool
	}{
		{
			name: "end slot",
			ctx: &parseCtx{
				activeBank: &bank{
					transactionIDs: nil,
					blk: &pbcodec.Block{
						Number:        55295941,
						PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
						PreviousBlock: 55295939,
					},
				},
			},
			line: "BLOCK_END 55295941 3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz 1606487316 1606487316",
			expectCtx: &parseCtx{
				activeBank: nil,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.ctx.readBlockEnd(test.line)
			require.NoError(t, err)
			assert.Equal(t, test.expectCtx, test.ctx)
		})
	}
}

func Test_readBlockRoot(t *testing.T) {
	tests := []struct {
		name          string
		ctx           *parseCtx
		line          string
		expectedBlock *pbcodec.Block
		expectCtx     *parseCtx
		expectError   bool
	}{
		{
			name: "block root",
			ctx: &parseCtx{
				activeBank: &bank{
					previousSlotID: blockId(t, "5XcRYrCbLFGSACy43fRdG4zJ88tWxB3eSx36MePjy3Ae"),
					ended:          true,
					blk: &pbcodec.Block{
						Id:                   blockId(t, "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz"),
						Number:               55295941,
						Version:              1,
						PreviousId:           blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
						PreviousBlock:        55295939,
						GenesisUnixTimestamp: 1606487316,
						ClockUnixTimestamp:   1606487316,
						Transactions:         trxSlice(t, []string{"aa", "bb", "cc", "dd"}),
						TransactionCount:     4,
					},
				},
				banks: map[uint64]*bank{
					55295941: {
						parentSlotNum:  55295939,
						previousSlotID: blockId(t, "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz"),
						ended:          true,
						blk: &pbcodec.Block{
							Id:                   blockId(t, "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz"),
							Number:               55295941,
							Version:              1,
							PreviousId:           blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
							PreviousBlock:        55295939,
							GenesisUnixTimestamp: 1606487316,
							ClockUnixTimestamp:   1606487316,
							Transactions:         trxSlice(t, []string{"aa", "bb", "cc", "dd"}),
							TransactionCount:     4,
						},
					},
				},
				blockBuffer: make(chan *pbcodec.Block, 100),
			},
			line: "BANK_ROOT 55295921",
			expectedBlock: &pbcodec.Block{
				Id:                   blockId(t, "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz"),
				Number:               55295941,
				PreviousId:           blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
				PreviousBlock:        55295939,
				GenesisUnixTimestamp: 1606487316,
				ClockUnixTimestamp:   1606487316,
				Version:              1,
				RootNum:              55295921,
				Transactions:         trxSlice(t, []string{"aa", "bb", "cc", "dd"}),
				TransactionCount:     4,
			},
			expectCtx: &parseCtx{
				activeBank: nil,
				banks:      map[uint64]*bank{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.ctx.readBlockRoot(test.line)
			require.NoError(t, err)
			require.Equal(t, 1, len(test.ctx.blockBuffer))
			block := <-test.ctx.blockBuffer
			assert.Equal(t, test.expectedBlock, block)
		})
	}
}

func trxSlice(t *testing.T, trxIDs []string) (out []*pbcodec.Transaction) {
	for i, id := range trxIDs {
		out = append(out, &pbcodec.Transaction{Id: trxID(t, id), Index: uint64(i)})
	}
	return
}

func copyTestDir(testPath, testName string) (func(), string, error) {
	var err error
	var fds []os.FileInfo

	src := fmt.Sprintf("%s/dmlogs", testPath)
	dst, err := ioutil.TempDir("", testName)
	if err != nil {
		return func() {}, "", fmt.Errorf("unable to create test directory: %w", err)
	}

	cleanup := func() {
		os.RemoveAll(dst)
	}

	if fds, err = ioutil.ReadDir(src); err != nil {
		return cleanup, "", fmt.Errorf("unable to read test data")
	}

	for _, fd := range fds {
		srcfp := path.Join(src, fd.Name())
		dstfp := path.Join(dst, fd.Name())
		if !fd.IsDir() {
			if err = copyFile(srcfp, dstfp); err != nil {
				return cleanup, "", fmt.Errorf("unable to copy test file %q to tmp dir %q: %w", srcfp, dstfp, err)
			}
		}
	}
	return cleanup, dst, nil
}

func copyFile(src, dst string) error {
	var err error
	var srcfd *os.File
	var dstfd *os.File
	var srcinfo os.FileInfo

	if srcfd, err = os.Open(src); err != nil {
		return err
	}
	defer srcfd.Close()

	if dstfd, err = os.Create(dst); err != nil {
		return err
	}
	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		return err
	}
	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}
	return os.Chmod(dst, srcinfo.Mode())
}

func testFileConsoleReader(t *testing.T, dmlogFile, batchFilesPath string) *ConsoleReader {
	t.Helper()

	fl, err := os.Open(dmlogFile)
	require.NoError(t, err)

	cr := testReaderConsoleReader(t, make(chan string, 10000), func() { fl.Close() }, batchFilesPath)

	go cr.ProcessData(fl)

	return cr
}

func testReaderConsoleReader(t *testing.T, lines chan string, closer func(), batchFilesPath string) *ConsoleReader {
	t.Helper()

	l := &ConsoleReader{
		lines: lines,
		close: closer,
		ctx:   newParseCtx(batchFilesPath),
	}

	return l
}

func blockId(t *testing.T, input string) []byte {
	out, err := base58.Decode(input)
	require.NoError(t, err)

	return out
}

func trxIDs(t *testing.T, inputs ...string) [][]byte {
	out := make([][]byte, len(inputs))
	for i, input := range inputs {
		out[i] = trxID(t, input)
	}

	return out
}

func trxID(t *testing.T, input string) []byte {
	bytes, err := hex.DecodeString(input)
	require.NoError(t, err)

	return bytes
}

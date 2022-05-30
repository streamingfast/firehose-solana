package bt

// serde: deserialize,serialize
type StoredConfirmedBlock struct {
	PreviousBlockhash string
	Blockhash         string
	ParentSlot        uint64
	//Transactions []StoredConfirmedBlockTransaction
	//rewards: StoredConfirmedBlockRewards,
	//block_time: Option<UnixTimestamp>,

}

//// serde: deserialize,serialize
//type Example struct {
//	vint64   int64
//	vmap     map[int]int
//	varray   [2]int `serde:"skip"`
//	vslice   []int
//	vpointer *int
//}

//// A serialized `StoredConfirmedBlock` is stored in the `block` table
////
//// StoredConfirmedBlock holds the same contents as ConfirmedBlock, but is slightly compressed and avoids
//// some serde JSON directives that cause issues with bincode
////
//// Note: in order to continue to support old bincode-serialized bigtable entries, if new fields are
//// added to ConfirmedBlock, they must either be excluded or set to `default_on_eof` here
////
//#[derive(Serialize, Deserialize)]
//struct StoredConfirmedBlock {
//previous_blockhash: String,
//blockhash: String,
//parent_slot: Slot,
//transactions: Vec<StoredConfirmedBlockTransaction>,
//rewards: StoredConfirmedBlockRewards,
//block_time: Option<UnixTimestamp>,
//#[serde(deserialize_with = "default_on_eof")]
//block_height: Option<u64>,
//}

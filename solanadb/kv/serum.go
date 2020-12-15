package kv

import (
	"context"

	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
)

func (db *DB) processSerumSlot(ctx context.Context, slot *pbcodec.Slot) error {
	for _, transactionTrace := range slot.TransactionTraces {
		for _, instruction := range transactionTrace.InstructionTraces {
			for _, accountChange := range instruction.AccountChanges {

			}
		}
		//
		//signedTransaction, err := codec.ExtractEOSSignedTransactionFromReceipt(trxReceipt)
		//if err != nil {
		//	return fmt.Errorf("unable to extract EOS signed transaction from transaction receipt: %w", err)
		//}
		//
		//signedTrx := codec.SignedTransactionToDEOS(signedTransaction)
		//pubKeyProto := &pbcodec.PublicKeys{
		//	PublicKeys: codec.GetPublicKeysFromSignedTransaction(db.writerChainID, signedTransaction),
		//}
		//
		//trxRow := &pbtrxdb.TrxRow{
		//	Receipt:    trxReceipt,
		//	SignedTrx:  signedTrx,
		//	PublicKeys: pubKeyProto,
		//}
		//
		//key := Keys.PackTrxsKey(trxReceipt.Id, blk.Id)
		//// NOTE: This function is guarded by the parent with db.enableTrxWrite
		//err = db.writeStore.Put(ctx, key, db.enc.MustProto(trxRow))
		//
		//if err != nil {
		//	return fmt.Errorf("put trx: write to db: %w", err)
		//}
	}

	return nil
}

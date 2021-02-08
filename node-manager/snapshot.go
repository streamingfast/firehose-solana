package nodemanager

import "github.com/dfuse-io/dstore"

func (s *Superviser) TakeSnapshot(snapshotStore dstore.Store, numberOfSnapshotsToKeep int) error {
	// Simply check in the right directory if there are any snapshots that the node
	// has taken.

	// If there's a snapshot, we upload it to the snapshotStore,
	// and that's it!

	// If there are snapshot that are older than `numberOfSnapshotsToKeep`, then wipe them

	// Maybe we can upload the 3.5GB each 10 minutes.. would that work? That would be the window
	// from which we can bring it back.

	return nil
}

func (s *Superviser) RestoreSnapshot(snapshotName string, snapshotStore dstore.Store) error {
	if snapshotName == "latest" {
		// find latest
	}

	if snapshotName == "before-last-merged" {

		// 63090809
		// 63590809
		// 64090809
		//   merged at  64090800
		// 64092834
		//  merged at  64092800 ?
		//  merged at  64099900 ?

		// find the snapshot the CLOSEST before the LAST MERGED BLOCK FILES we can find.
	}

	// otherwise, use that as the block number ?!

	// take that snapshot, download it
	// remove the other snapshots from the local directory..
	return nil
}

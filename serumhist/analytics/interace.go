package analytics

type Analyzable interface {
	Get24hVolume() (float64, error)
}

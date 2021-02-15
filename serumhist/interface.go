package serumhist

type serumEvent interface {
}

type EventWriter interface {
	Write(e serumEvent)
}

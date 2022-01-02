package legohat

type Device struct {
	ID        int
	Name      string
	Listeners []chan []byte
}

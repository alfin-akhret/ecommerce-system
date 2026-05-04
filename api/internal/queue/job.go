package queue

type Job struct {
	Type    string
	Payload []byte
	Retry   int
}

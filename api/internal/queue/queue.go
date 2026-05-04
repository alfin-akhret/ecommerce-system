package queue

// the queue

type Queue struct {
	Jobs     chan Job
	Registry *Registry
}

// push job to queue
func (q *Queue) Enqueue(job Job) {
	q.Jobs <- job
}

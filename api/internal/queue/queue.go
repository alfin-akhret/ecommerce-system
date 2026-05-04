package queue

// the queue

type Queue struct {
	jobs     chan Job
	registry *Registry
}

// push job to queue
func (q *Queue) Enqueue(job Job) {
	q.jobs <- job
}

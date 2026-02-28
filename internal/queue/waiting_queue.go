package queue

type WaitingQueue struct {
	users []int64
}

func (w *WaitingQueue) IsEmpty() bool {
	return len(w.users) == 0
}

func (w *WaitingQueue) Dequeue() (int64, bool) {
	if len(w.users) == 0 {
		return 0, false
	}

	userID := w.users[0]
	w.users = w.users[1:]
	return userID, true
}

func (w *WaitingQueue) Enqueue(userID int64) {
	w.users = append(w.users, userID)
}

func (w *WaitingQueue) Remove(userID int64) {
	// TODO оптимизировать
	filtered := w.users[:0]

	for _, id := range w.users {
		if id != userID {
			filtered = append(filtered, id)
		}
	}

	w.users = filtered
}

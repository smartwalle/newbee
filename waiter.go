package newbee

type Waiter interface {
	Add(delta int)

	Done()

	Wait()
}

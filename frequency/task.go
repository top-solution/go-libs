package frequency

// import (
// 	"fmt"
// 	"time"
// )

// type TaskFn func() time.Time

// type AbortFn func()

// type Task struct {
// 	Frequency *Frequency
// 	ticker    *time.Ticker
// 	done      chan bool
// }

// func (d Frequency) PeriodicTask(fn TaskFn) *Task {
// 	t := &Task{
// 		done: make(chan bool),
// 	}
// 	if d.duration != 0 {
// 		t.ticker = time.NewTicker(d.duration)
// 	} else {
// 		t.ticker = time.NewTicker(time.Minute * 5)
// 	}
// 	go func() {
// 		for {
// 			select {
// 			case <-t.done:
// 				return
// 			case t := <-t.ticker.C:
// 				fmt.Println("Tick at", t)
// 			}
// 		}
// 	}()
// }

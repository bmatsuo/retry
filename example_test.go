package retry

import (
	"fmt"
	"time"
)

func Example_MaxTries() {
	withretry := func(maxTries int, minDelay, maxDelay, timeout time.Duration, dostuff func(chan<- error, <-chan struct{})) error {
		deadline := time.After(timeout)
		retry, maxtries := MaxTries(maxTries, Retry(0,
			Exponential(2),
			Bounded(minDelay, maxDelay)))
		ready := retry.Retry()
		killch := make(chan struct{})
		var stufferr chan error
		for {
			select {
			case <-deadline:
				close(killch)
				return fmt.Errorf("timeout")
			case <-ready:
				stufferr = make(chan error, 1)
				go dostuff(stufferr, killch)
			case <-maxtries:
				return fmt.Errorf("max tries")
			case err := <-stufferr:
				stufferr = nil
				if err != nil {
					ready = retry.Retry()
					continue
				}
				return nil
			}
		}
	}

	fmt.Println(withretry(4, time.Second, 5*time.Second, 4*time.Second, func(errch chan<- error, killed <-chan struct{}) {
		errch <- fmt.Errorf("dead")
		close(errch)
	}))

	fmt.Println(withretry(4, time.Second, 5*time.Second, 4*time.Second, func(errch chan<- error, killed <-chan struct{}) {
		defer close(errch)
		select {
		case <-killed:
			errch <- fmt.Errorf("killed")
		case <-time.After(time.Second):
			errch <- fmt.Errorf("dead")
		}
	}))

	flag := make(chan struct{}, 1)
	fmt.Println(withretry(4, time.Second, 5*time.Second, 4*time.Second, func(errch chan<- error, killed <-chan struct{}) {
		defer close(errch)
		select {
		case flag <- struct{}{}:
			errch <- fmt.Errorf("dead")
		default:
		}
	}))

	// Output:
	// max tries
	// timeout
	// <nil>
}

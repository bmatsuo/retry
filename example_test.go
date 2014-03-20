package retry

import (
	"fmt"
	"time"
)

// this constructs an exponential iterative delay function with an upper bound.
func ExampleDelay() {
	delay := Delay(
		Exponential(2),
		BoundedMax(10),
	)
	d := time.Duration(1)
	for i := 0; i < 6; i++ {
		fmt.Println(d)
		d = delay(d)
	}
	// Output:
	// 1ns
	// 2ns
	// 4ns
	// 8ns
	// 10ns
	// 10ns
}

// this example reverses the arguments to demonstrate the importance of order.
func ExampleDelay_reversed() {
	delay := Delay(
		BoundedMax(10),
		Exponential(2),
	)
	d := time.Duration(1)
	for i := 0; i < 6; i++ {
		fmt.Println(d)
		d = delay(d)
	}
	// Output:
	// 1ns
	// 2ns
	// 4ns
	// 8ns
	// 16ns
	// 20ns
}

// this simple example demonstrates the use of an exponential backoff and calls
// to Retry().
func ExampleRetry_simple() {
	retry := Retry(0,
		Exponential(2),
		Bounded(100*time.Millisecond, 5*time.Second))

	ready := retry.Retry() // approx. time.Now()
	fmt.Println(<-ready)

	ready = retry.Retry() // approx. time.Now() + 100*time.Millisecond
	fmt.Println(<-ready)

	ready = retry.Retry() // approx. time.Now() + 200*time.Millisecond
	fmt.Println(<-ready)
}

// making requests to an unreliable service
func ExampleRetry() {
	type Response struct {
		data string
		err  error
	}
	errServiceUnavailable := fmt.Errorf("kaboom")
	count := 0
	unreliableService := func() Response {
		count++
		if count > 3 {
			return Response{"ok", nil}
		}
		return Response{"", errServiceUnavailable}
	}
	retryService := func(kill <-chan struct{}) Response {
		// automatically retry if the service is temporarily unavailable
		done := make(chan Response)
		retry := Retry(0, Constant(time.Millisecond))
		ready := retry.Retry()
		for {
			select {
			case <-kill:
				return Response{"", fmt.Errorf("killed")}
			case resp := <-done:
				if resp.err == errServiceUnavailable {
					ready = retry.Retry()
					fmt.Println(resp.err)
					continue
				}
				return resp
			case <-ready:
				go func() {
					select {
					case done <- unreliableService():
					case <-kill:
					}
				}()
			}
		}
	}

	// attempt to contact the unreliableService for at most 5 ms
	stop := make(chan struct{})
	go func() {
		<-time.After(5 * time.Millisecond)
		close(stop)
	}()
	resp := retryService(stop)

	fmt.Println("got:", resp.data, resp.err)

	// Output:
	// kaboom
	// kaboom
	// kaboom
	// got: ok <nil>
}

// this example retries a concurrently executing function until it succeeds,
// a set number of tries has been exceeded, or a set amount of has time passed.
func ExampleMaxTries() {
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

	fmt.Println(withretry(4, time.Millisecond, 5*time.Millisecond, 9*time.Millisecond, func(errch chan<- error, killed <-chan struct{}) {
		errch <- fmt.Errorf("dead")
		close(errch)
	}))

	fmt.Println(withretry(4, time.Millisecond, 5*time.Millisecond, 9*time.Millisecond, func(errch chan<- error, killed <-chan struct{}) {
		defer close(errch)
		select {
		case <-killed:
			errch <- fmt.Errorf("killed")
		case <-time.After(time.Millisecond):
			errch <- fmt.Errorf("dead")
		}
	}))

	flag := make(chan struct{}, 1)
	fmt.Println(withretry(4, time.Millisecond, 5*time.Millisecond, 4*time.Millisecond, func(errch chan<- error, killed <-chan struct{}) {
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

/*
retries with expenential backoff.

this package is experimental and the api is subject to change.
*/
package retry

import (
	"math/rand"
	"time"
)

// the retry interface facilitates signaling the entrance of a state after
// an arbitrary amount of time.
type Interface interface {
	// returns a channel which, on receiving a value, notifies the caller
	// that an action should be retried.
	Retry() <-chan time.Time
}

// the returned interface sends after initial nanos on the first call to Retry().
// successed calls to the interface's Retry() method signal in delay(initial)
// nanos, delay(delay(initial)) nanos, ...
func Retry(initial time.Duration, delay ...DelayFunc) Interface {
	checkRollover(initial)
	if len(delay) == 0 {
		return retry{initial, nil}
	}
	if len(delay) == 1 {
		return retry{initial, delay[0]}
	}
	return retry{initial, Delay(delay...)}
}

// if n is less than 1 c is closed immediately.
// if r is nil it is equivalent to passing Retry(0).
func MaxTries(n int, r Interface) (Interface, <-chan struct{}) {
	c := make(chan struct{})
	_r := &maxretries{
		c: c,
		n: n,
		r: r,
	}
	if n < 1 {
		close(c)
	}
	return _r, c
}

// an iterative function for computing retry delays. delays are computed as
//	d1 = f(d0)
//	d2 = f(d1)
//	d3 = f(d2)
//	...
type DelayFunc func(time.Duration) time.Duration

// chain fns together to construct a new DelayFunc. Delay(f, g, h)(d) is equivalent to h(g(f(d))).
func Delay(fns ...DelayFunc) DelayFunc {
	return func(d time.Duration) time.Duration {
		for i := range fns {
			d = fns[i](d)
		}
		return d
	}
}

// a DelayFunc that returns the product of its input and a random value in
// the range
//	[1-fuzzfactor, 1+fuzzfactor)
func Randomize(fuzzfactor float64) DelayFunc {
	if fuzzfactor < 0 {
		panic("negative fuzz factor")
	}
	size := 2 * fuzzfactor
	return func(d time.Duration) time.Duration {
		scale := 1 + rand.Float64()*size - fuzzfactor
		return time.Duration(scale * float64(d))
	}
}

// the returned DelayFunc passes arguments through, but returns max if the
// argument is greater than max. panics if max is negative.
func BoundedMax(max time.Duration) DelayFunc {
	if max < 0 {
		panic("negative bound")
	}
	return func(d time.Duration) time.Duration {
		if d > max {
			return max
		}
		return d
	}
}

// the returned DelayFunc passes arguments through, but returns min if the
// argument is greater than max. panics if max is negative.
func BoundedMin(min time.Duration) DelayFunc {
	if min < 0 {
		panic("negative bound")
	}
	return func(d time.Duration) time.Duration {
		if d < min {
			return min
		}
		return d
	}
}

// panics if max is less than min. otherwise functionally equivalent to
//	Delay(BoundedMin(min), BoundedMax(max))
func Bounded(min, max time.Duration) DelayFunc {
	if max < min {
		panic("undefined bounds")
	}
	if max == min {
		return Constant(max)
	}
	return Delay(BoundedMin(min), BoundedMax(max))
}

// returns a DelayFunc that always produces delay.
func Constant(delay time.Duration) DelayFunc {
	return func(time.Duration) time.Duration { return delay }
}

// a DelayFunc that multiplies by scale on application.
// iterative application produces growth on the order of
//	scale**n
func Exponential(scale float64) DelayFunc {
	if scale < 0 {
		panic("negative scale")
	}
	if scale == 0 {
		return Constant(0)
	}
	if scale == 2 {
		return exponential2
	}
	return func(d time.Duration) time.Duration {
		d = time.Duration(scale * float64(d))
		checkRollover(d)
		return d
	}
}

func checkRollover(d time.Duration) {
	if d < 0 {
		panic("negative duration")
	}
}

func exponential2(d time.Duration) time.Duration {
	d = d << 1
	checkRollover(d)
	return d
}

type maxretries struct {
	c    chan<- struct{}
	i, n int
	r    Interface
}

func (r *maxretries) Retry() <-chan time.Time {
	if r.i == r.n {
		r.i++
		close(r.c)
		return nil
	}
	r.i++
	if r.r == nil {
		return retry{}.Retry()
	}
	return r.r.Retry()
}

type retry struct {
	d     time.Duration
	delay DelayFunc
}

func (r retry) Retry() <-chan time.Time {
	c := time.After(r.d)
	if r.delay != nil {
		r.d = r.delay(r.d)
	}
	return c
}

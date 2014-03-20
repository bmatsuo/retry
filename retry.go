// retries with expenential backoff.
package retry

import (
	"time"
)

// chain fns together to construct a new DelayFunc. Delay(f, g, h)(d) is equivalent to h(g(f(d))).
//	Delay(Exponential(2), BoundedMax(125*time.Millisecond, time.Minute))
func Delay(fns ...DelayFunc) DelayFunc {
	return func(d time.Duration) time.Duration {
		for i := range fns {
			d = fns[i](d)
		}
		return d
	}
}

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

func Bounded(min, max time.Duration) DelayFunc {
	if max < min {
		panic("undefined bounds")
	}
	if max == min {
		return Constant(max)
	}
	return Delay(BoundedMin(min), BoundedMax(max))
}

type DelayFunc func(time.Duration) time.Duration

func checkRollover(d time.Duration) {
	if d < 0 {
		panic("negative duration")
	}
}

func Constant(delay time.Duration) DelayFunc {
	return func(time.Duration) time.Duration { return delay }
}

func exponential2(d time.Duration) time.Duration {
	d = d << 1
	checkRollover(d)
	return d
}

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

type Interface interface {
	Retry() <-chan time.Time
}

// The Retry() on the returned interface receives a value after the initial duration.
// The initial duration seeds iterative calles to delay on successive calls to Retry()
//	Retry(0, Exponential(2), Bounded(100*time.Millisecond, 5*time.Second))
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

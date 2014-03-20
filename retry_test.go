package retry

import (
	y "github.com/bmatsuo/yup"
	yt "github.com/bmatsuo/yup/yuptype"

	"runtime"
	"testing"
	"time"
)

func callAndRecover(fn func()) (v interface{}) {
	defer func() { v = recover() }()
	fn()
	return
}

func TestExponential(t *testing.T) {
	yt.NotNil(t, callAndRecover(func() { Exponential(-1) }))
	yt.Nil(t, callAndRecover(func() { Exponential(0) }))
	yt.Nil(t, callAndRecover(func() { Exponential(0.5) }))
	yt.Nil(t, callAndRecover(func() { Exponential(2) }))
	yt.Nil(t, callAndRecover(func() { Exponential(2.5) }))
	yt.Equal(t, time.Duration(0), Exponential(0)(1))
	yt.Equal(t, time.Duration(2), Exponential(2)(1))
	yt.Equal(t, time.Duration(1), Exponential(1.5)(1))
	yt.Equal(t, time.Duration(3), Exponential(1.5)(2))
}

func TestBounds(t *testing.T) {
	yt.NotNil(t, callAndRecover(func() { BoundedMax(-1) }))
	yt.NotNil(t, callAndRecover(func() { BoundedMin(-1) }))
	yt.NotNil(t, callAndRecover(func() { Bounded(-1, 0) }))
	yt.NotNil(t, callAndRecover(func() { Bounded(0, -1) }))
	yt.NotNil(t, callAndRecover(func() { Bounded(2, 0) }))
	yt.Nil(t, callAndRecover(func() { Bounded(1, 3) }))
	yt.Nil(t, callAndRecover(func() { Bounded(0, 0) }))

	yt.Equal(t, time.Duration(2), Bounded(2, 2)(2))
	yt.Equal(t, time.Duration(2), BoundedMax(2)(2))
	yt.Equal(t, time.Duration(2), BoundedMin(2)(2))

	yt.Equal(t, time.Duration(2), Bounded(2, 2)(3))
	yt.Equal(t, time.Duration(2), BoundedMax(2)(3))
	yt.Equal(t, time.Duration(1), BoundedMax(2)(1))

	yt.Equal(t, time.Duration(2), Bounded(2, 2)(1))
	yt.Equal(t, time.Duration(2), BoundedMin(2)(1))
	yt.Equal(t, time.Duration(3), BoundedMin(2)(3))
}

func TestMaxTriesNil(t *testing.T) {
	retry, _ := MaxTries(1, nil)
	ready := retry.Retry()
	runtime.Gosched()
	select {
	case <-ready:
	default:
		y.Fail(t)
	}
}

func TestMaxTriesZero(t *testing.T) {
	_, maxtries := MaxTries(0, Retry(0))
	select {
	case <-maxtries:
	default:
		y.Fail(t)
	}
}

func TestMaxTriesNegative(t *testing.T) {
	_, maxtries := MaxTries(-1, Retry(0))
	select {
	case <-maxtries:
	default:
		y.Fail(t)
	}
}

func TestRetry(t *testing.T) {
	yt.NotNil(t, callAndRecover(func() { Retry(-1) }))
	yt.Nil(t, callAndRecover(func() { Retry(0) }))
	yt.Nil(t, callAndRecover(func() { Retry(0, Constant(0)) }))
	yt.Nil(t, callAndRecover(func() { Retry(0, Exponential(2), Bounded(1, 2)) }))
}

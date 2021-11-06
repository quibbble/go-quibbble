package timer

import "time"

// Timer counts down and performs an action on alarm
type Timer struct {
	duration time.Duration
	timer    *time.Timer
	endTime  time.Time
	alarm    chan bool
}

func NewTimer(duration time.Duration, alarm chan bool) *Timer {
	return &Timer{
		duration: duration,
		timer:    nil,
		alarm:    alarm,
	}
}

func (t *Timer) Start() {
	if t.timer != nil {
		t.Stop()
	}
	t.timer = time.NewTimer(t.duration)
	t.endTime = time.Now().Add(t.duration)
	go func() {
		<-t.timer.C
		t.alarm <- true
	}()
}

func (t *Timer) Stop() {
	if t.timer != nil {
		t.timer.Stop()
	}
}

func (t *Timer) Remaining() time.Duration {
	return t.endTime.Sub(time.Now())
}

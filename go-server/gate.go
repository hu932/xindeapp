package main

import (
	"sync"
	"sync/atomic"
	"time"
)

// ForwardGate：全局转发并发闸门（内存实现，替代 PHP 的文件 flock + JSON 重写）。
// 这是原 PHP 在高峰期“卡死/上传不了/超时”的根因之一：每个请求都要
// 打开锁文件 flock、读 JSON、改写 JSON、写盘，高并发下争用严重。
// 这里用信号量 + 原子计数，无磁盘 IO，几乎零争用。
type ForwardGate struct {
	sem           chan struct{}
	max           int
	ratePerSec    int
	waitMs        int
	active        atomic.Int64
	mu            sync.Mutex
	recent        []time.Time
	history       map[int64]int // unixSec -> count
	busyHistory   map[int64]int
	busyTotal     atomic.Int64
	lastGateWait  atomic.Int64 // 毫秒
}

func newForwardGate() *ForwardGate {
	g := &ForwardGate{
		max:         forwardMaxConcurrency,
		ratePerSec:  forwardRatePerSec,
		waitMs:      forwardGateMaxWaitMs,
		history:     map[int64]int{},
		busyHistory: map[int64]int{},
	}
	if g.max > 0 {
		g.sem = make(chan struct{}, g.max)
	}
	return g
}

func (g *ForwardGate) enabled() bool { return g.max > 0 || g.ratePerSec > 0 }

// tryAcquireOnce：无等待尝试获取；返回是否成功、等待毫秒
func (g *ForwardGate) tryAcquireOnce(now time.Time) bool {
	if !g.enabled() {
		return true
	}
	sec := now.Unix()

	g.mu.Lock()
	// 清理 rate 窗口
	if g.ratePerSec > 0 && len(g.recent) > 0 {
		cut := now.Add(-time.Second)
		kept := g.recent[:0]
		for _, t := range g.recent {
			if t.After(cut) {
				kept = append(kept, t)
			}
		}
		g.recent = kept
	}
	// 清理历史
	for s := range g.history {
		if s <= sec-60 {
			delete(g.history, s)
		}
	}
	for s := range g.busyHistory {
		if s <= sec-60 {
			delete(g.busyHistory, s)
		}
	}

	ok := true
	if g.max > 0 {
		if int(g.active.Load()) >= g.max {
			ok = false
		}
	}
	if ok && g.ratePerSec > 0 && len(g.recent) >= g.ratePerSec {
		ok = false
	}
	if ok {
		g.active.Add(1)
		if g.ratePerSec > 0 {
			g.recent = append(g.recent, now)
		}
		g.history[sec]++
	}
	g.mu.Unlock()
	return ok
}

// acquire：尝试获取，并发满时最多等待 waitMs
func (g *ForwardGate) acquire() (ok bool, waitMs int) {
	if !g.enabled() {
		return true, 0
	}
	start := time.Now()
	if g.tryAcquireOnce(start) {
		g.lastGateWait.Store(0)
		return true, 0
	}
	wait := g.waitMs
	if wait <= 0 {
		g.recordBusy()
		g.lastGateWait.Store(elapsedMs(start, time.Now()))
		return false, int(elapsedMs(start, time.Now()))
	}
	deadline := start.Add(time.Duration(wait) * time.Millisecond)
	for time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
		if g.tryAcquireOnce(start) {
			ms := elapsedMs(start, time.Now())
			g.lastGateWait.Store(ms)
			return true, int(ms)
		}
	}
	g.recordBusy()
	ms := elapsedMs(start, time.Now())
	g.lastGateWait.Store(ms)
	return false, int(ms)
}

func (g *ForwardGate) release() {
	if !g.enabled() {
		return
	}
	g.active.Add(-1)
}

func (g *ForwardGate) recordBusy() {
	sec := time.Now().Unix()
	g.mu.Lock()
	g.busyHistory[sec]++
	g.mu.Unlock()
	g.busyTotal.Add(1)
}

func elapsedMs(start, end time.Time) int64 {
	return end.Sub(start).Milliseconds()
}

// metrics 供负载监控使用
func (g *ForwardGate) metrics() map[string]any {
	nowSec := time.Now().Unix()
	g.mu.Lock()
	active := int(g.active.Load())
	rate1, rate10, rate60, busy10, busy60 := 0, 0, 0, 0, 0
	series := make([]map[string]any, 0, 60)
	for i := int64(59); i >= 0; i-- {
		s := nowSec - i
		c := g.history[s]
		b := g.busyHistory[s]
		if i == 0 {
			rate1 = c
		}
		if i < 10 {
			rate10 += c
			busy10 += b
		}
		rate60 += c
		busy60 += b
		series = append(series, map[string]any{
			"ts": s, "time": time.Unix(s, 0).In(locAsiaShanghai).Format("15:04:05"),
			"count": c, "busy": b,
		})
	}
	g.mu.Unlock()

	return map[string]any{
		"active":            active,
		"max_concurrency":   g.max,
		"rate_limit_per_sec": g.ratePerSec,
		"wait_ms":           g.waitMs,
		"rate_1s":           rate1,
		"rate_10s":          rate10,
		"rate_60s":          rate60,
		"busy_10s":          busy10,
		"busy_60s":          busy60,
		"series":            series,
	}
}

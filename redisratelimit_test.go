package redisratelimit

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"
)

type mockRedis struct {
	list map[string][]int64
}

func (r *mockRedis) Eval(ctx context.Context, script string, keys []string, args ...any) (any, error) {
	switch script {
	case scriptRateLimitSecond, scriptRateLimitMilliSecond:
		return r.ratelimit(keys[0], args[0].(int64), args[1].(int), args[2].(int64))
	default:
		return nil, errors.New("mock redis: unsupported script")
	}
}

func (r *mockRedis) ratelimit(key string, window int64, limit int, now int64) (string, error) {
	items := r.list[key]
	if len(items) >= limit {
		dur := now - items[0]
		if dur < window {
			return strconv.FormatInt(window-dur, 10), nil
		}
		items = items[1:]
	}
	items = append(items, now)
	if r.list == nil {
		r.list = make(map[string][]int64)
	}
	r.list[key] = items
	return "OK", nil
}

func TestAccessInSecond(t *testing.T) {
	redis := new(mockRedis)

	ctx := context.Background()
	key := "ratelimit:your:biz"
	window := int64(10)
	limit := 1

	ok, next, err := AccessInSecond(ctx, redis, key, window, limit)
	if err != nil {
		t.Fatalf("AccessInSecond: %v", err)
	}
	if !ok {
		t.Fatalf("AccessInSecond not ok")
	}
	if next != 0 {
		t.Fatalf("AccessInSecond ok next: %v", next)
	}

	ok, next, err = AccessInSecond(ctx, redis, key, window, limit)
	if err != nil {
		t.Fatalf("AccessInSecond: %v", err)
	}
	if ok {
		t.Fatalf("AccessInSecond ok second")
	}
	if next != window {
		t.Fatalf("AccessInSecond not ok next: %v", next)
	}

	ok, next, err = AccessInSecond(ctx, redis, key, window, limit, time.Now().Unix()+window)
	if err != nil {
		t.Fatalf("AccessInSecond: %v", err)
	}
	if !ok {
		t.Fatalf("AccessInSecond not ok third")
	}
	if next != 0 {
		t.Fatalf("AccessInSecond not ok third next: %v", next)
	}
}

func TestAccessInMilliSecond(t *testing.T) {
	redis := new(mockRedis)

	ctx := context.Background()
	key := "ratelimit:your:biz"
	window := int64(1000)
	limit := 1

	ok, next, err := AccessInMilliSecond(ctx, redis, key, window, limit)
	if err != nil {
		t.Fatalf("AccessInMilliSecond: %v", err)
	}
	if !ok {
		t.Fatalf("AccessInMilliSecond not ok")
	}
	if next != 0 {
		t.Fatalf("AccessInMilliSecond ok next: %v", next)
	}

	ok, next, err = AccessInMilliSecond(ctx, redis, key, window, limit)
	if err != nil {
		t.Fatalf("AccessInMilliSecond: %v", err)
	}
	if ok {
		t.Fatalf("AccessInMilliSecond ok second")
	}
	if next != window {
		t.Fatalf("AccessInMilliSecond not ok next: %v", next)
	}

	ok, next, err = AccessInMilliSecond(ctx, redis, key, window, limit, time.Now().UnixMilli()+window)
	if err != nil {
		t.Fatalf("AccessInMilliSecond: %v", err)
	}
	if !ok {
		t.Fatalf("AccessInMilliSecond not ok third")
	}
	if next != 0 {
		t.Fatalf("AccessInMilliSecond ok third next: %v", next)
	}
}

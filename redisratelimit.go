package redisratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// usage:
// RateLimit key window limit now
const scriptRateLimitSecond = `local key = KEYS[1]
local window = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local now = ARGV[3]
if redis.call("LLEN", key) >= limit then
	local dur = tonumber(now) - tonumber(redis.call("LINDEX", key, 0))
	if dur < window then
		return tostring(window-dur)
	end
	redis.call("LPOP", key)
end
redis.call("RPUSH", key, now)
redis.call("EXPIRE", key, window)
return "OK"
`

const scriptRateLimitMilliSecond = `local key = KEYS[1]
local window = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local now = ARGV[3]
if redis.call("LLEN", key) >= limit then
	local dur = tonumber(now) - tonumber(redis.call("LINDEX", key, 0))
	if dur < window then
		return tostring(window-dur)
	end
	redis.call("LPOP", key)
end
redis.call("RPUSH", key, now)
redis.call("PEXPIRE", key, window)
return "OK"
`

type RedisClient interface {
	Eval(ctx context.Context, script string, keys []string, args ...any) (any, error)
}

// access 返回是否被限流。
// 如果被限流，第二个返回值表示下次可访问需等待时长。
// 参数：
// window: 时间窗口。
// limit: 频率次数限制。
// now: 当前时间戳。
// window和now需要统一单位。
// 如果window单位为秒，now需要为秒级时间戳；
// 如果window单位为毫秒，now需要为毫秒级时间戳；
// window和limit表示：在window时间内，允许访问limit次。
func access(ctx context.Context, rdb RedisClient, script string, key string, window int64, limit int, now int64) (bool, int64, error) {
	if window < 1 {
		window = 1
	}

	result, err := rdb.Eval(ctx, script, []string{key},
		window, limit, now)
	if err != nil {
		return false, 0, err
	}

	value, ok := result.(string)
	if !ok {
		return false, 0, fmt.Errorf("redis ratelimit: lua eval result type is not string: %T", result)
	}

	if value == "OK" {
		return true, 0, nil
	}

	wait, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return false, 0, fmt.Errorf("redis ratelimit: parse lua eval result %q: %w", value, err)
	}
	return false, wait, nil
}

// AccessInSecond 返回是否被限流。
// 如果被限流，第二个返回值表示下次可访问需等待秒数。
// 注意不能和AccessInMilliSecond混用。
func AccessInSecond(ctx context.Context, rdb RedisClient, key string, window int64, limit int, nowUnix ...int64) (bool, int64, error) {
	var now int64
	if len(nowUnix) > 0 {
		now = nowUnix[0]
	} else {
		now = time.Now().Unix()
	}
	return access(ctx, rdb, scriptRateLimitSecond, key, window, limit, now)
}

// AccessInMilliSecond 返回是否被限流。
// 如果被限流，第二个返回值表示下次可访问需等待毫秒数。
// 注意不能和AccessInSecond混用。
func AccessInMilliSecond(ctx context.Context, rdb RedisClient, key string, window int64, limit int, nowUnixMilli ...int64) (bool, int64, error) {
	var now int64
	if len(nowUnixMilli) > 0 {
		now = nowUnixMilli[0]
	} else {
		now = time.Now().UnixMilli()
	}
	return access(ctx, rdb, scriptRateLimitMilliSecond, key, window, limit, now)
}

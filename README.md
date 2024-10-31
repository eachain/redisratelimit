# redisratelimit

redisratelimit依赖redis实现了一个简单的限流方案：在一定时间内，只允许有限数量的请求通过。

实现方式为，redis维护一个时间窗口，随时间流动，窗口向前滑动。当新请求来时，如果窗口内请求次数已超过限制，则限流，否则记录当前时间戳，用于以后判定是否超出时间窗口。

## 示例

```go
package main

import (
	"context"
	"fmt"

	"github.com/eachain/redisratelimit"
	"github.com/go-redis/redis/v8"
)

type redisClient redis.Client

func (r *redisClient) Eval(ctx context.Context, script string, keys []string, args ...any) (any, error) {
	return (*redis.Client)(r).Eval(ctx, script, keys, args...).Result()
}

func main() {
	var rdb *redis.Client

	rdb = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "password",
	})

	ok, next, err := redisratelimit.AccessInSecond(context.Background(), (*redisClient)(rdb),
		"mutex:biz:key", 10, 3)
	if err != nil {
		fmt.Printf("ratelimit: %v\n", err)
		return
	}
	if !ok {
		fmt.Printf("ratelimit not access, next access after %vs\n", next)
		return
	}

	fmt.Println("do your biz here")
}
```

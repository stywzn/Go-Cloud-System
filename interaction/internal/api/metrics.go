package api

import (
	"github.com/prometheus/client_golang/prometheus"
)

// 定义指标
var (
	// HTTP请求总数 (注意：带标签必须用 NewCounterVec)
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "interaction_service_http_requests_total",
			Help: "Total number of HTTP requests handled by the interaction service",
		},
		[]string{"method", "path", "status"},
	)

	// 抽奖/秒杀请求总数 (替换掉旧的点赞指标)
	seckillTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "interaction_service_seckill_total",
			Help: "Total number of lottery seckill requests",
		},
		[]string{"result"}, // success, failed, duplicate
	)

	// 被限流拦截请求总数 (修复了拼写和缺少的逗号)
	rateLimitBlockedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "interaction_service_rate_limit_blocked_total",
			Help: "Total number of requests blocked by rate limiting", // 这里的逗号非常重要
		},
	)
)

// init 函数会在包被导入时自动执行，注册这些指标
func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(seckillTotal)
	prometheus.MustRegister(rateLimitBlockedTotal)
}

func RecordHTTPRequest(method, path, status string) {
	httpRequestsTotal.WithLabelValues(method, path, status).Inc()
}

func RecordSeckillRequest(result string) {
	seckillTotal.WithLabelValues(result).Inc()
}

func RecordRateLimitBlocked() {
	rateLimitBlockedTotal.Inc()
}

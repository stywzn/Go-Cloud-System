// 管理Prometheus 的指标

package api

import(
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// 定义指标
// http请求总数，点赞总数，限流拦截总数

var (
	//http请求总数
	httpRequestsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "interaction_service_http_requests_total",
			Help: "Total number of HTTP requests handled by the interaction service",
		},
		[]string{"method", "path", "status"}
	)
	//点赞总数
	likesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "interaction_service_likes_total",
			Help: "Total number of likes/unlikes requests",
		},
		[]string{"action","result"}
	)
	// 被限流拦截请求总数
	reteLimiteBlocked = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "interaction_service_rate_limit_blocked_total",
			Help: "Total number of requests blocked by rate limiting",
		}
	)
)

func InitMetrics() {

}

func RecordHTTPRequest(method, path, status string) {
	httpRequestsTotal.WithLabelValues(path,method,status).Inc()
}

func RecordLikeRequest(action,result string) {
	LikeRequestTotal.WithLabelValues(action,result).Inc()
}

func RecordRateLimitBlocked() {
	rateLimitBlockedTotal.Inc()
}
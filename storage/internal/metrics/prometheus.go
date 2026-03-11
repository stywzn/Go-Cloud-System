package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// 分片上传计数
	UploadPartCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "upload_parts_total",
			Help: "Total number of uploaded parts",
		},
		[]string{"status"},
	)

	// 分片上传失败计数
	UploadPartFailureCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "upload_parts_failed_total",
			Help: "Total number of failed uploads",
		},
		[]string{"error_type"},
	)

	// 分片上传耗时直方图
	UploadPartDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "upload_part_duration_seconds",
			Help:    "Duration of uploading a part in seconds",
			Buckets: []float64{.1, .5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"size_range"},
	)

	// 用户已用容量仪表
	UserStorageUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "user_storage_usage_bytes",
			Help: "User storage usage in bytes",
		},
		[]string{"user_id"},
	)

	// 库存Gauge（当前可用配额）
	StorageQuotaRemaining = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "storage_quota_remaining_bytes",
			Help: "Remaining storage quota in bytes",
		},
		[]string{"user_id"},
	)

	// 完成传输计数
	UploadCompleteCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "upload_complete_total",
			Help: "Total number of completed uploads",
		},
		[]string{"status"},
	)

	// QPS (API请求)
	APIRequestCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_requests_total",
			Help: "Total number of API requests",
		},
		[]string{"method", "path", "status"},
	)

	// 请求耗时
	APIRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_request_duration_seconds",
			Help:    "Duration of API requests in seconds",
			Buckets: []float64{.01, .05, .1, .5, 1, 2, 5, 10},
		},
		[]string{"method", "path"},
	)
)

// RecordUploadPartSuccess 记录分片上传成功
func RecordUploadPartSuccess() {
	UploadPartCounter.WithLabelValues("success").Inc()
}

// RecordUploadPartFailure 记录分片上传失败
func RecordUploadPartFailure(errType string) {
	UploadPartCounter.WithLabelValues("failure").Inc()
	UploadPartFailureCounter.WithLabelValues(errType).Inc()
}

// RecordUploadComplete 记录上传完成
func RecordUploadComplete(status string) {
	UploadCompleteCounter.WithLabelValues(status).Inc()
}

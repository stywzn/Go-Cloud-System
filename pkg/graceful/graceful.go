package graceful

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// ShutdownManager 优雅停机管理器
type ShutdownManager struct {
	server    *http.Server
	shutdown  chan struct{}
	done      chan struct{}
	timeout   time.Duration
	onShutdown []func() error
}

// NewShutdownManager 创建优雅停机管理器
func NewShutdownManager(server *http.Server, timeout time.Duration) *ShutdownManager {
	return &ShutdownManager{
		server:   server,
		shutdown: make(chan struct{}),
		done:     make(chan struct{}),
		timeout:  timeout,
	}
}

// AddShutdownHook 添加停机钩子
func (sm *ShutdownManager) AddShutdownHook(hook func() error) {
	sm.onShutdown = append(sm.onShutdown, hook)
}

// WaitForShutdown 等待停机信号
func (sm *ShutdownManager) WaitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Printf("收到信号 %v，开始优雅停机...", sig)
	case <-sm.shutdown:
		log.Println("收到内部停机请求，开始优雅停机...")
	}

	// 执行停机钩子
	for _, hook := range sm.onShutdown {
		if err := hook(); err != nil {
			log.Printf("执行停机钩子失败: %v", err)
		}
	}

	// 创建停机上下文
	ctx, cancel := context.WithTimeout(context.Background(), sm.timeout)
	defer cancel()

	// 停止HTTP服务器
	if err := sm.server.Shutdown(ctx); err != nil {
		log.Printf("服务器停机失败: %v", err)
		sm.server.Close()
	}

	close(sm.done)
	log.Println("优雅停机完成")
}

// Shutdown 触发优雅停机
func (sm *ShutdownManager) Shutdown() {
	close(sm.shutdown)
}

// Done 返回停机完成通道
func (sm *ShutdownManager) Done() <-chan struct{} {
	return sm.done
}

// IsShuttingDown 检查是否正在停机
func (sm *ShutdownManager) IsShuttingDown() bool {
	select {
	case <-sm.shutdown:
		return true
	default:
		return false
	}
}

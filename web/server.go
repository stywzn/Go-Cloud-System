package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	// 检查端口环境变量，默认8084
	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	// 设置静态文件目录
	fs := http.FileServer(http.Dir("."))

	// 处理根路径，默认提供简化版页面
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "simple.html")
		} else {
			fs.ServeHTTP(w, r)
		}
	})

	log.Printf("🌐 前端服务启动成功！")
	log.Printf("📍 简化版界面: http://localhost:%s", port)
	log.Printf("📍 完整版界面: http://localhost:%s/index.html", port)
	log.Printf("🔗 API网关: http://localhost:8080")
	log.Printf("☁️  存储服务: http://localhost:8083")
	log.Printf("🎁 抽奖服务: http://localhost:8082")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("前端服务启动失败: %v", err)
	}
}

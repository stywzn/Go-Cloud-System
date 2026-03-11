package middleware

import (
	"log"
	"net/http"
)

func Auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")

		if token != "Bearer my-secret-token" {
			log.Printf("[安全拦截]未授权的访问尝试: IP=%s, 路径=%s", r.RemoteAddr, r.URL.Path)
			http.Error(w, "Unauthorized: 请提供有效的 Token", http.StatusUnauthorized)
			return
		}
		log.Printf("[安全通过]  验证成功, 放行请求: %s", r.URL.Path)
		next(w, r)
	}
}

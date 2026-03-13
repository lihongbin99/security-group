package server

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"security-group/internal/aliyun"
	"security-group/internal/auth"
)

// clientRealIP 依次尝试从反向代理头获取真实IP，拿不到则从TCP连接获取。
func clientRealIP(r *http.Request) string {
	// 优先 X-Real-IP
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	// 其次 X-Forwarded-For，取第一个IP
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if ip := strings.TrimSpace(strings.SplitN(xff, ",", 2)[0]); ip != "" {
			return ip
		}
	}
	// 兜底：TCP RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	IP      string `json:"ip,omitempty"`
}

type UserInfo struct {
	Username  string `json:"username"`
	PublicIP  string `json:"public_ip"`
	LocalIP   string `json:"local_ip"`
	UpdatedAt string `json:"updated_at"`
}

type Server struct {
	auth   *auth.Auth
	aliyun *aliyun.Client
	webFS  fs.FS

	mu    sync.RWMutex
	users map[string]*UserInfo // username -> info
}

func New(a *auth.Auth, al *aliyun.Client, webContent embed.FS) *Server {
	webFS, err := fs.Sub(webContent, "web")
	if err != nil {
		log.Fatalf("failed to load embedded web files: %v", err)
	}
	return &Server{
		auth:   a,
		aliyun: al,
		webFS:  webFS,
		users:  make(map[string]*UserInfo),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(s.webFS)))
	mux.HandleFunc("POST /api/update", s.handleUpdate)
	mux.HandleFunc("POST /api/users", s.handleUsers)
	return mux
}

type updateRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	LocalIP  string `json:"local_ip"`
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	clientIP := clientRealIP(r)

	if s.auth.IsBlocked(clientIP) {
		writeJSON(w, http.StatusForbidden, Response{Code: 2, Message: "IP 已被封禁，请稍后重试"})
		return
	}

	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Code: 1, Message: "请求格式错误"})
		return
	}

	if !s.auth.Authenticate(clientIP, req.Password) {
		writeJSON(w, http.StatusUnauthorized, Response{Code: 1, Message: "密码错误"})
		return
	}

	mu := s.auth.LockUser(req.Username)
	mu.Lock()
	defer mu.Unlock()

	msg, err := s.aliyun.UpdateIP(clientIP, req.Username)
	if err != nil {
		log.Printf("aliyun error for user=%s ip=%s: %v", req.Username, clientIP, err)
		writeJSON(w, http.StatusInternalServerError, Response{Code: 3, Message: "安全组操作失败: " + err.Error()})
		return
	}

	log.Printf("user=%s ip=%s local_ip=%s result=%s", req.Username, clientIP, req.LocalIP, msg)

	s.mu.Lock()
	s.users[req.Username] = &UserInfo{
		Username:  req.Username,
		PublicIP:  clientIP,
		LocalIP:   req.LocalIP,
		UpdatedAt: time.Now().Format("2006-01-02 15:04:05"),
	}
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, Response{Code: 0, Message: msg, IP: clientIP})
}

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Code: 1, Message: "请求格式错误"})
		return
	}
	if !s.auth.Authenticate(clientRealIP(r), req.Password) {
		writeJSON(w, http.StatusUnauthorized, Response{Code: 1, Message: "密码错误"})
		return
	}

	s.mu.RLock()
	list := make([]*UserInfo, 0, len(s.users))
	for _, u := range s.users {
		list = append(list, u)
	}
	s.mu.RUnlock()

	sort.Slice(list, func(i, j int) bool {
		return list[i].Username < list[j].Username
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func writeJSON(w http.ResponseWriter, status int, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

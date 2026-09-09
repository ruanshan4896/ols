package redirect

import "time"

type RedirectType string

const (
	TypeDomain RedirectType = "domain" // Chuyển hướng toàn bộ website (Full Domain)
	TypePath   RedirectType = "path"   // Chuyển hướng đường dẫn URL cụ thể (URL Path)
)

// RedirectRule đại diện cho một quy tắc chuyển hướng
type RedirectRule struct {
	ID           string       `json:"id"`            // Mã định danh quy tắc (e.g. rd-1a2b3c)
	Type         RedirectType `json:"type"`          // "domain" hoặc "path"
	Source       string       `json:"source"`        // Tên miền nguồn hoặc đường dẫn URL nguồn (e.g. "old.com" hoặc "/old-post")
	Target       string       `json:"target"`        // Đích đến (e.g. "https://new.com" hoặc "/new-post")
	StatusCode   int          `json:"status_code"`   // 301 (Permanent) hoặc 302 (Temporary)
	PreservePath bool         `json:"preserve_path"` // Giữ nguyên đường dẫn con ($1) khi chuyển domain
	Note         string       `json:"note"`          // Ghi chú mục đích
	CreatedAt    string       `json:"created_at"`    // Thời gian tạo
}

// RedirectTestResult kết quả kiểm tra chuyển hướng thực tế
type RedirectTestResult struct {
	OriginalURL  string `json:"original_url"`
	StatusCode   int    `json:"status_code"`
	RedirectURL  string `json:"redirect_url"`
	IsRedirect   bool   `json:"is_redirect"`
	ResponseTime string `json:"response_time"`
	Error        string `json:"error,omitempty"`
}

func NewRedirectRule(rType RedirectType, source, target string, code int, preservePath bool, note string) RedirectRule {
	if code != 301 && code != 302 {
		code = 301
	}
	id := generateRuleID()
	return RedirectRule{
		ID:           id,
		Type:         rType,
		Source:       source,
		Target:       target,
		StatusCode:   code,
		PreservePath: preservePath,
		Note:         note,
		CreatedAt:    time.Now().Format("2006-01-02 15:04:05"),
	}
}

func generateRuleID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	now := time.Now().UnixNano()
	for i := range b {
		b[i] = charset[int(now>>(i*5))%len(charset)]
	}
	return "rd-" + string(b)
}

package apiparser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// user.api 内容
const testUserAPIContent = `
syntax = "v1"

info (
	title: "用户服务API"
)

type User {
	Id   int64  ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}

@server (
	prefix: /api/v1
	group: user
)
service user-api {
	@doc "获取用户信息"
	@handler GetUser
	get /users/:id returns (User)

	@handler ListUsers
	get /users returns ([]User)

	@handler CreateUser
	post /users (User) returns (User)
}
`

// order.api 内容
const testOrderAPIContent = `
syntax = "v1"

info (
	title: "订单服务API"
)

@server (
	prefix: /api/v1
	group: order
)
service order-api {
	@handler GetOrder
	get /orders/:orderId returns (string)

	@handler GetOrderItem
	get /orders/:orderId/items/:itemId returns (string)
}
`

// 写入临时文件并返回路径
func writeTempAPIFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	filePath := filepath.Join(dir, name)
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("写入测试文件 %s 失败: %v", name, err)
	}
	return filePath
}

// TestPrintAPISpec 打印解析后的路由结构
func TestPrintAPISpec(t *testing.T) {
	tmpDir := t.TempDir()
	apiFile := writeTempAPIFile(t, tmpDir, "user.api", testUserAPIContent)

	parser := NewParser()
	routes, err := parser.ParseFile(apiFile)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	data, err := json.MarshalIndent(routes, "", "  ")
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	t.Logf("Routes:\n%s", string(data))
}

// TestParseSingleFile 测试解析单个文件
func TestParseSingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	apiFile := writeTempAPIFile(t, tmpDir, "user.api", testUserAPIContent)

	parser := NewParser()
	routes, err := parser.ParseFile(apiFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// spec 包会自动合并 @server 的 prefix，所以路径带 /api/v1
	expected := map[string]struct {
		handler     string
		serviceName string
		group       string
	}{
		"GET /api/v1/users/*": {"GetUser", "user-api", "user"},
		"GET /api/v1/users":   {"ListUsers", "user-api", "user"},
		"POST /api/v1/users":  {"CreateUser", "user-api", "user"},
	}

	if len(routes) != len(expected) {
		t.Fatalf("路由数量: got %d, want %d", len(routes), len(expected))
	}

	for _, r := range routes {
		key := r.Method + " " + r.Path
		exp, ok := expected[key]
		if !ok {
			t.Errorf("出现未预期的路由: %s", key)
			continue
		}
		if r.Handler != exp.handler {
			t.Errorf("路由[%s] Handler: got %s, want %s", key, r.Handler, exp.handler)
		}
		if r.ServiceName != exp.serviceName {
			t.Errorf("路由[%s] ServiceName: got %s, want %s", key, r.ServiceName, exp.serviceName)
		}
		if r.Group != exp.group {
			t.Errorf("路由[%s] Group: got %s, want %s", key, r.Group, exp.group)
		}
		delete(expected, key)
	}

	for key := range expected {
		t.Errorf("缺少预期的路由: %s", key)
	}

	for _, r := range routes {
		t.Logf("Service=%s Group=%s %s %s (original=%s, handler=%s, doc=%s)",
			r.ServiceName, r.Group, r.Method, r.Path, r.OriginalPath, r.Handler, r.Doc)
	}
}

// TestParseFiles 测试解析多个文件（核心场景）
func TestParseFiles(t *testing.T) {
	tmpDir := t.TempDir()
	userFile := writeTempAPIFile(t, tmpDir, "user.api", testUserAPIContent)
	orderFile := writeTempAPIFile(t, tmpDir, "order.api", testOrderAPIContent)

	parser := NewParser()
	result, err := parser.ParseFiles(userFile, orderFile)
	if err != nil {
		t.Fatalf("ParseFiles failed: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("结果文件数量: got %d, want 2", len(result))
	}

	// 校验 user.api
	userRoutes, ok := result["user.api"]
	if !ok {
		t.Fatal("结果中缺少 user.api")
	}
	if len(userRoutes) != 3 {
		t.Errorf("user.api 路由数量: got %d, want 3", len(userRoutes))
	}
	for _, r := range userRoutes {
		if r.ServiceName != "user-api" {
			t.Errorf("user.api ServiceName: got %s, want user-api", r.ServiceName)
		}
	}

	// 校验 order.api
	orderRoutes, ok := result["order.api"]
	if !ok {
		t.Fatal("结果中缺少 order.api")
	}
	if len(orderRoutes) != 2 {
		t.Errorf("order.api 路由数量: got %d, want 2", len(orderRoutes))
	}

	// order.api 里的动态路径应该被转换为通配符，且路径已带 prefix
	expectedHandlers := map[string]bool{
		"GetOrder":     false,
		"GetOrderItem": false,
	}

	for _, r := range orderRoutes {
		if r.Method != "GET" {
			t.Errorf("order.api Method: got %s, want GET", r.Method)
		}
		if r.Path != "/api/v1/orders/*" {
			t.Errorf("order.api Path: got %s, want /api/v1/orders/*", r.Path)
		}
		if r.ServiceName != "order-api" {
			t.Errorf("order.api ServiceName: got %s, want order-api", r.ServiceName)
		}
		if _, ok := expectedHandlers[r.Handler]; !ok {
			t.Errorf("order.api 未预期的 Handler: %s", r.Handler)
			continue
		}
		expectedHandlers[r.Handler] = true
	}
	for h, found := range expectedHandlers {
		if !found {
			t.Errorf("order.api 缺少预期的 Handler: %s", h)
		}
	}

	// 打印所有路由便于调试
	for fileName, routes := range result {
		t.Logf("=== 文件: %s ===", fileName)
		for _, r := range routes {
			t.Logf("  Service=%s Group=%s %s %s (original=%s, handler=%s, doc=%s)",
				r.ServiceName, r.Group, r.Method, r.Path, r.OriginalPath, r.Handler, r.Doc)
		}
	}
}

// TestConvertDynamicPath 测试动态路径转换
func TestConvertDynamicPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users/:id", "/users/*"},
		{"/order/:orderId/item/:id", "/order/*"},
		{"/users/:id/profile", "/users/*"},
		{"/users", "/users"},
		{"/", "/"},
		{"/api/v1/users/:id", "/api/v1/users/*"},
		{"/files/:path/download/:fileId", "/files/*"},
	}

	for _, tt := range tests {
		result := convertDynamicPath(tt.input)
		if result != tt.expected {
			t.Errorf("convertDynamicPath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

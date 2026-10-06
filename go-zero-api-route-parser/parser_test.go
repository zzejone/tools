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
	desc: "用户服务的API描述"
	version: "v1.0"
)

type User {
	Id   int64  ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}

@server (
	prefix: /api/v1
	group: user
	doc: server-doc
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
	desc: "订单服务的API描述"
	version: "v2.0"
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
	fileInfo, err := parser.ParseFile(apiFile)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	data, err := json.MarshalIndent(fileInfo, "", "  ")
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	t.Logf("FileInfo:\n%s", string(data))
}

// TestParseSingleFile 测试解析单个文件
func TestParseSingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	apiFile := writeTempAPIFile(t, tmpDir, "user.api", testUserAPIContent)

	parser := NewParser()
	fileInfo, err := parser.ParseFile(apiFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// 校验 FileInfo 元数据
	if fileInfo.Title != "用户服务API" {
		t.Errorf("Title: got %s, want 用户服务API", fileInfo.Title)
	}
	if fileInfo.Desc != "用户服务的API描述" {
		t.Errorf("Desc: got %s, want 用户服务的API描述", fileInfo.Desc)
	}
	if fileInfo.Version != "v1.0" {
		t.Errorf("Version: got %s, want v1.0", fileInfo.Version)
	}
	if fileInfo.Service.Name != "user-api" {
		t.Errorf("Service.Name: got %s, want user-api", fileInfo.Service.Name)
	}

	// 校验分组结构（user.api 仅一个 group）
	if len(fileInfo.Service.Groups) != 1 {
		t.Fatalf("分组数量: got %d, want 1", len(fileInfo.Service.Groups))
	}
	group := fileInfo.Service.Groups[0]
	if group.Name != "user" {
		t.Errorf("Group.Name: got %s, want user", group.Name)
	}
	if group.Annotation["prefix"] != "/api/v1" {
		t.Errorf("Annotation[prefix]: got %s, want /api/v1", group.Annotation["prefix"])
	}
	if group.Annotation["group"] != "user" {
		t.Errorf("Annotation[group]: got %s, want user", group.Annotation["group"])
	}
	if group.Annotation["doc"] != "server-doc" {
		t.Errorf("Annotation[doc]: got %s, want server-doc", group.Annotation["doc"])
	}

	// spec 包会自动合并 @server 的 prefix，所以路径带 /api/v1
	expected := map[string]string{
		"GET /api/v1/users/*": "GetUser",
		"GET /api/v1/users":   "ListUsers",
		"POST /api/v1/users":  "CreateUser",
	}

	if len(group.Routes) != len(expected) {
		t.Fatalf("路由数量: got %d, want %d", len(group.Routes), len(expected))
	}

	for _, r := range group.Routes {
		key := r.Method + " " + r.Path
		handler, ok := expected[key]
		if !ok {
			t.Errorf("出现未预期的路由: %s", key)
			continue
		}
		if r.Handler != handler {
			t.Errorf("路由[%s] Handler: got %s, want %s", key, r.Handler, handler)
		}
		delete(expected, key)
	}

	for key := range expected {
		t.Errorf("缺少预期的路由: %s", key)
	}

	for _, r := range group.Routes {
		t.Logf("Group=%s %s %s (original=%s, handler=%s, doc=%s)",
			group.Name, r.Method, r.Path, r.OriginalPath, r.Handler, r.Doc)
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
	userInfo, ok := result["user.api"]
	if !ok {
		t.Fatal("结果中缺少 user.api")
	}
	if userInfo.Title != "用户服务API" {
		t.Errorf("user.api Title: got %s, want 用户服务API", userInfo.Title)
	}
	if userInfo.Desc != "用户服务的API描述" {
		t.Errorf("user.api Desc: got %s, want 用户服务的API描述", userInfo.Desc)
	}
	if userInfo.Version != "v1.0" {
		t.Errorf("user.api Version: got %s, want v1.0", userInfo.Version)
	}
	if userInfo.Service.Name != "user-api" {
		t.Errorf("user.api Service.Name: got %s, want user-api", userInfo.Service.Name)
	}
	if len(userInfo.Service.Groups) != 1 {
		t.Errorf("user.api 分组数量: got %d, want 1", len(userInfo.Service.Groups))
	} else {
		userGroup := userInfo.Service.Groups[0]
		if userGroup.Name != "user" {
			t.Errorf("user.api Group.Name: got %s, want user", userGroup.Name)
		}
		if len(userGroup.Routes) != 3 {
			t.Errorf("user.api 路由数量: got %d, want 3", len(userGroup.Routes))
		}
	}

	// 校验 order.api
	orderInfo, ok := result["order.api"]
	if !ok {
		t.Fatal("结果中缺少 order.api")
	}
	if orderInfo.Title != "订单服务API" {
		t.Errorf("order.api Title: got %s, want 订单服务API", orderInfo.Title)
	}
	if orderInfo.Desc != "订单服务的API描述" {
		t.Errorf("order.api Desc: got %s, want 订单服务的API描述", orderInfo.Desc)
	}
	if orderInfo.Version != "v2.0" {
		t.Errorf("order.api Version: got %s, want v2.0", orderInfo.Version)
	}
	if orderInfo.Service.Name != "order-api" {
		t.Errorf("order.api Service.Name: got %s, want order-api", orderInfo.Service.Name)
	}
	if len(orderInfo.Service.Groups) != 1 {
		t.Fatalf("order.api 分组数量: got %d, want 1", len(orderInfo.Service.Groups))
	}
	orderGroup := orderInfo.Service.Groups[0]
	if orderGroup.Name != "order" {
		t.Errorf("order.api Group.Name: got %s, want order", orderGroup.Name)
	}
	if len(orderGroup.Routes) != 2 {
		t.Errorf("order.api 路由数量: got %d, want 2", len(orderGroup.Routes))
	}

	// order.api 里的动态路径应该被转换为通配符，且路径已带 prefix
	expectedHandlers := map[string]bool{
		"GetOrder":     false,
		"GetOrderItem": false,
	}

	for _, r := range orderGroup.Routes {
		if r.Method != "GET" {
			t.Errorf("order.api Method: got %s, want GET", r.Method)
		}
		if r.Path != "/api/v1/orders/*" {
			t.Errorf("order.api Path: got %s, want /api/v1/orders/*", r.Path)
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

	// 打印所有内容便于调试（文件 → 分组 → 路由）
	for fileName, info := range result {
		t.Logf("=== 文件: %s (service=%s) ===", fileName, info.Service.Name)
		for _, g := range info.Service.Groups {
			t.Logf("  --- 分组: %s (annotation=%v) ---", g.Name, g.Annotation)
			for _, r := range g.Routes {
				t.Logf("    %s %s (original=%s, handler=%s, doc=%s)",
					r.Method, r.Path, r.OriginalPath, r.Handler, r.Doc)
			}
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

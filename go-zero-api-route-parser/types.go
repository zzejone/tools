package apiparser

// RouteInfo 路由信息
type RouteInfo struct {
	Method       string `json:"method"`
	Path         string `json:"path"`
	OriginalPath string `json:"originalPath"`
	Handler      string `json:"handler"`
	Group        string `json:"group"`
	ServiceName  string `json:"serviceName"`
	Doc          string `json:"doc"`
}

// ParseResult key 是文件名，value 是路由列表
type ParseResult map[string][]RouteInfo

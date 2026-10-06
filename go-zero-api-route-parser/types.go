package apiparser

// RouteInfo 路由信息
type RouteInfo struct {
	Method       string `json:"method"`
	Path         string `json:"path"`
	OriginalPath string `json:"originalPath"`
	Handler      string `json:"handler"`
	Doc          string `json:"doc"`
}

// ParseResult key 是文件名，value 是路由列表
type ParseResult map[string]FileInfo

type ServiceInfo struct {
	Name   string      `json:"name"`
	Groups []GroupInfo `json:"groups"`
}

type GroupInfo struct {
	Name       string            `json:"name"`
	Annotation map[string]string `json:"annotation"`
	Routes     []RouteInfo       `json:"routes"`
}

type FileInfo struct {
	Title   string `json:"title"`
	Desc    string `json:"desc"`
	Version string `json:"version"`

	Service ServiceInfo `json:"service"`
}

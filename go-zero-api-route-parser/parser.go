package apiparser

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zeromicro/go-zero/tools/goctl/api/parser"
	"github.com/zeromicro/go-zero/tools/goctl/api/spec"
)

// Parser API 解析器
type Parser struct {
	KeepOriginalPath bool
}

// NewParser 创建解析器
func NewParser() *Parser {
	return &Parser{KeepOriginalPath: true}
}

// ParseFiles 解析多个 API 文件
func (p *Parser) ParseFiles(filePaths ...string) (ParseResult, error) {
	result := make(ParseResult)
	for _, filePath := range filePaths {
		routes, err := p.ParseFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("解析文件 %s 失败: %w", filePath, err)
		}
		result[filepath.Base(filePath)] = routes
	}
	return result, nil
}

// ParseFile 使用 goctl 官方 spec 包解析单个 API 文件
//
// 注意：spec 包不会自动合并 @server 的 prefix，需要手动拼接。
func (p *Parser) ParseFile(filePath string) (fileInfo FileInfo, err error) {
	var apiSpec *spec.ApiSpec
	apiSpec, err = parser.Parse(filePath)
	if err != nil {
		return
	}

	var serviceInfo ServiceInfo
	serviceInfo.Name = apiSpec.Service.Name
	serviceInfo.Groups = []GroupInfo{}

	for _, group := range apiSpec.Service.Groups {
		var routes []RouteInfo
		for _, route := range group.Routes {
			// 手动拼接 @server prefix
			originalPath := route.Path
			if prefix, ok := group.Annotation.Properties["prefix"]; ok && prefix != "" {
				originalPath = joinPath(prefix, originalPath)
			}

			// 处理 @doc 的引号
			doc := route.AtDoc.Text
			if len(doc) >= 2 && doc[0] == '"' && doc[len(doc)-1] == '"' {
				doc = doc[1 : len(doc)-1]
			}

			routeInfo := RouteInfo{
				Method:  strings.ToUpper(route.Method),
				Path:    convertDynamicPath(originalPath),
				Handler: route.Handler,
				Doc:     doc,
			}

			if p.KeepOriginalPath {
				routeInfo.OriginalPath = originalPath
			}

			routes = append(routes, routeInfo)
		}

		var _group GroupInfo
		_group.Name = group.Annotation.Properties["group"]
		_group.Annotation = group.Annotation.Properties
		_group.Routes = routes

		serviceInfo.Groups = append(serviceInfo.Groups, _group)
	}

	fileInfo.Title = apiSpec.Info.Title
	fileInfo.Desc = apiSpec.Info.Desc
	fileInfo.Version = apiSpec.Info.Version
	fileInfo.Service = serviceInfo

	return fileInfo, nil
}

// convertDynamicPath 将 :param 形式的动态路径转换为通配符形式
func convertDynamicPath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			return strings.Join(parts[:i], "/") + "/*"
		}
	}
	return path
}

// joinPath 拼接路径，确保中间只有一个斜杠
func joinPath(prefix, path string) string {
	prefix = strings.TrimSuffix(prefix, "/")
	path = strings.TrimPrefix(path, "/")
	if prefix == "" {
		return "/" + path
	}
	if path == "" {
		return prefix
	}
	return prefix + "/" + path
}

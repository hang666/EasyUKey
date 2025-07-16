package api

import (
	"html/template"
	"io"
	"path/filepath"

	"github.com/labstack/echo/v4"
)

// TemplateRenderer 模板渲染器
type TemplateRenderer struct {
	templates *template.Template
}

// NewTemplateRenderer 创建模板渲染器
func NewTemplateRenderer(templateDir string) *TemplateRenderer {
	// 加载模板文件
	templates := template.Must(template.ParseGlob(filepath.Join(templateDir, "*.html")))

	return &TemplateRenderer{
		templates: templates,
	}
}

// Render 渲染模板
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

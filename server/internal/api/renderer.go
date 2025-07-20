package api

import (
	"embed"
	"html/template"
	"io"
	"io/fs"

	"github.com/labstack/echo/v4"
)

// TemplateRenderer 模板渲染器
type TemplateRenderer struct {
	templates *template.Template
}

// NewEmbedTemplateRenderer 创建模板渲染器（从embed FS加载）
func NewEmbedTemplateRenderer(embedFS embed.FS) *TemplateRenderer {
	subFS, err := fs.Sub(embedFS, "template")
	if err != nil {
		panic("无法找到嵌入的template目录: " + err.Error())
	}

	templates := template.Must(template.ParseFS(subFS, "*.html"))

	return &TemplateRenderer{
		templates: templates,
	}
}

// Render 渲染模板
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

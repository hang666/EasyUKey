package api

import (
	"html/template"
	"io"

	"github.com/labstack/echo/v4"
)

// TemplateRenderer 模板渲染器
type TemplateRenderer struct {
	Templates *template.Template
}

// Render 渲染方法
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.Templates.ExecuteTemplate(w, name, data)
}

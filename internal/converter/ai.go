package converter

import (
	"fmt"

	"go.uber.org/zap"
)

// AIConvertRequest AI 转换请求（用于传递给 Claude）
type AIConvertRequest struct {
	Markdown     string // Markdown 内容
	Prompt       string // 完整的提示词
	Theme        string // 主题名称
	CustomPrompt string // 自定义提示词（如果有）
}

// AIConvertResult AI 转换结果
type AIConvertResult struct {
	HTML    string // 生成的 HTML
	Success bool
	Error   string
}

// aiConverter AI 模式转换器
type aiConverter struct {
	log   *zap.Logger
	theme *ThemeManager
}

// NewAIConverter 创建 AI 转换器
func NewAIConverter(log *zap.Logger, theme *ThemeManager) *aiConverter {
	return &aiConverter{
		log:   log,
		theme: theme,
	}
}

// convertViaAI 通过 AI 模式执行转换
// 注意：实际的 AI 调用由外部（Claude）执行，此方法准备请求结构
func (c *converter) convertViaAI(req *ConvertRequest) *ConvertResult {
	result := &ConvertResult{
		Mode:    ModeAI,
		Theme:   req.Theme,
		Success: false,
	}

	// 获取提示词
	prompt, err := c.buildAIPrompt(req)
	if err != nil {
		result.Error = fmt.Sprintf("build AI prompt failed: %s", err.Error())
		return result
	}

	// 提取图片引用
	images := c.ExtractImages(req.Markdown)

	// AI 模式由外部调用者处理，这里返回准备好的请求
	// 实际使用时，调用者应该：
	// 1. 获取 AIConvertRequest
	// 2. 发送给 Claude
	// 3. 获取返回的 HTML
	// 4. 调用 CompleteAIConversion 填充结果

	// 为了保持接口一致性，这里返回一个包含提示词的特殊结果
	result.Error = "AI_MODE_REQUEST:" + prompt
	result.Images = images

	c.log.Info("AI conversion request prepared",
		zap.String("theme", req.Theme),
		zap.Int("image_count", len(images)),
		zap.Int("prompt_length", len(prompt)))

	return result
}

// buildAIPrompt 构建 AI 提示词
func (c *converter) buildAIPrompt(req *ConvertRequest) (string, error) {
	var prompt string

	// 如果有自定义提示词，使用自定义
	if req.CustomPrompt != "" {
		prompt = BuildCustomAIPrompt(req.CustomPrompt)
	} else {
		// 否则使用内置主题的提示词
		theme, err := c.theme.GetTheme(req.Theme)
		if err != nil {
			// 如果找不到主题，使用通用提示词
			prompt = c.getGenericPrompt()
		} else if theme.Type != "ai" {
			// 不是 AI 主题，使用通用提示词
			c.log.Warn("theme is not AI mode, using generic prompt",
				zap.String("theme", req.Theme),
				zap.String("type", theme.Type))
			prompt = c.getGenericPrompt()
		} else {
			// 使用 PromptBuilder 构建完整 Prompt
			prompt, err = c.promptBuilder.BuildPromptFromTheme(theme, req.Markdown, nil)
			if err != nil {
				c.log.Warn("build prompt from theme failed, using raw prompt", zap.Error(err))
				prompt = theme.Prompt + "\n\n```\n" + req.Markdown + "\n```"
			} else {
				// 验证 Prompt 内容
				validation := ValidatePromptContent(prompt)
				if !validation.Valid {
					c.log.Warn("prompt validation failed",
						zap.Strings("errors", validation.Errors))
				}
				if len(validation.Warnings) > 0 {
					c.log.Debug("prompt validation warnings",
						zap.Strings("warnings", validation.Warnings))
				}
			}
			return prompt, nil
		}
	}

	// 添加 Markdown 内容
	fullPrompt := prompt + "\n\n```\n" + req.Markdown + "\n```"

	return fullPrompt, nil
}

// getGenericPrompt 获取通用提示词
func (c *converter) getGenericPrompt() string {
	return `你是一个专业的微信公众号排版助手。请将以下 Markdown 内容转换为微信公众号兼容的 HTML。

## 样式要求
- 使用内联 CSS（style 属性）
- 整洁大方的排版
- 适当的间距和行高

## 重要规则
1. 所有 CSS 必须使用内联 style 属性
2. 不使用外部样式表或 <style> 标签
3. 只使用安全的 HTML 标签
4. 图片使用占位符格式：<!-- IMG:index -->
5. 返回完整的 HTML，不需要其他说明文字`
}

// PrepareAIRequest 准备 AI 转换请求（供外部调用）
func (c *converter) PrepareAIRequest(req *ConvertRequest) (*AIConvertRequest, error) {
	prompt, err := c.buildAIPrompt(req)
	if err != nil {
		return nil, err
	}

	return &AIConvertRequest{
		Markdown:     req.Markdown,
		Prompt:       prompt,
		Theme:        req.Theme,
		CustomPrompt: req.CustomPrompt,
	}, nil
}

// CompleteAIConversion 完成 AI 转换（由外部调用 AI 后使用）
func CompleteAIConversion(html string, images []ImageRef, theme string) *ConvertResult {
	return &ConvertResult{
		HTML:    html,
		Mode:    ModeAI,
		Theme:   theme,
		Images:  images,
		Success: true,
	}
}

// IsAIRequest 检查结果是否是 AI 请求
func IsAIRequest(result *ConvertResult) bool {
	return result.Error != "" && len(result.Error) > 14 && result.Error[:15] == "AI_MODE_REQUEST"
}

// ExtractAIRequest 从结果中提取 AI 请求
func ExtractAIRequest(result *ConvertResult) string {
	if IsAIRequest(result) {
		return result.Error[14:] // 去掉前缀
	}
	return ""
}

// GetAIRequestInfo 获取 AI 请求的详细信息
func GetAIRequestInfo(result *ConvertResult) (prompt string, images []ImageRef, ok bool) {
	if !IsAIRequest(result) {
		return "", nil, false
	}
	return ExtractAIRequest(result), result.Images, true
}

// BuildAIRequestForExternal 为外部调用者构建 AI 请求
func BuildAIRequestForExternal(markdown, theme, customPrompt string, themeMgr *ThemeManager) (string, []ImageRef, error) {
	// 提取图片
	var images []ImageRef

	// 简单的图片提取逻辑
	lines := markdown
	imgIndex := 0
	for _, line := range splitLines(lines) {
		if containsImageSyntax(line) {
			images = append(images, ImageRef{
				Index:    imgIndex,
				Original: line,
			})
			imgIndex++
		}
	}

	// 构建提示词
	var prompt string
	if customPrompt != "" {
		prompt = BuildCustomAIPrompt(customPrompt)
	} else {
		builtInPrompt, err := themeMgr.GetAIPrompt(theme)
		if err != nil {
			prompt = getGenericPromptForExternal()
		} else {
			prompt = builtInPrompt
		}
	}

	// 添加 Markdown 内容
	fullPrompt := prompt + "\n\n```\n" + markdown + "\n```"

	return fullPrompt, images, nil
}

// 辅助函数
func splitLines(s string) []string {
	// 简单的按行分割
	lines := []string{}
	current := ""
	for _, ch := range s {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func containsImageSyntax(line string) bool {
	return len(line) > 4 && (line[0:2] == "![" || contains(line, "<!-- IMG:"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func getGenericPromptForExternal() string {
	return `你是一个专业的微信公众号排版助手。请将以下 Markdown 内容转换为微信公众号兼容的 HTML。

## 重要规则
1. 所有 CSS 必须使用内联 style 属性
2. 不使用外部样式表或 <style> 标签
3. 只使用安全的 HTML 标签
4. 图片使用占位符格式：<!-- IMG:index -->
5. 返回完整的 HTML，不需要其他说明文字`
}

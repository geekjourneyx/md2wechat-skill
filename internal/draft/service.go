package draft

import (
	"encoding/json"
	"fmt"
	stdhtml "html"
	"os"
	"regexp"
	"strings"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"github.com/geekjourneyx/md2wechat-skill/internal/wechat"
	"github.com/silenceper/wechat/v2/officialaccount/draft"
	"go.uber.org/zap"
)

// Service 草稿服务
type Service struct {
	cfg *config.Config
	log *zap.Logger
	ws  *wechat.Service
}

// NewService 创建草稿服务
func NewService(cfg *config.Config, log *zap.Logger) *Service {
	return &Service{
		cfg: cfg,
		log: log,
		ws:  wechat.NewService(cfg, log),
	}
}

// ArticleType 文章类型
type ArticleType string

const (
	ArticleTypeNews    ArticleType = "news"    // 图文消息（默认）
	ArticleTypeNewspic ArticleType = "newspic" // 小绿书/图片消息
)

// ImageItem 图片项（小绿书专用）
type ImageItem struct {
	ImageMediaID string `json:"image_media_id"`
}

// ImageInfo 图片信息（小绿书专用）
type ImageInfo struct {
	ImageList []ImageItem `json:"image_list"`
}

// DraftRequest 草稿请求
type DraftRequest struct {
	Articles []Article `json:"articles"`
}

// Article 文章
type Article struct {
	Title            string `json:"title"`
	Author           string `json:"author,omitempty"`
	Digest           string `json:"digest,omitempty"`
	Content          string `json:"content"`
	ContentSourceURL string `json:"content_source_url,omitempty"`
	ThumbMediaID     string `json:"thumb_media_id,omitempty"`
	ShowCoverPic     int    `json:"show_cover_pic,omitempty"`

	// 小绿书/图片消息专用字段
	ArticleType        ArticleType `json:"article_type,omitempty"`
	NeedOpenComment    int         `json:"need_open_comment,omitempty"`
	OnlyFansCanComment int         `json:"only_fans_can_comment,omitempty"`
	ImageInfo          *ImageInfo  `json:"image_info,omitempty"`
}

// DraftResult 草稿结果
type DraftResult struct {
	MediaID  string `json:"media_id"`
	DraftURL string `json:"draft_url,omitempty"`
}

// CreateDraftFromFile 从 JSON 文件创建草稿
func (s *Service) CreateDraftFromFile(jsonFile string) (*DraftResult, error) {
	s.log.Info("creating draft from file", zap.String("file", jsonFile))

	// 读取 JSON 文件
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// 解析请求
	var req DraftRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	// 验证
	if len(req.Articles) == 0 {
		return nil, fmt.Errorf("no articles in request")
	}

	// 转换为 SDK 格式
	articles, err := buildSDKArticles(req.Articles)
	if err != nil {
		return nil, err
	}

	// 调用微信 API
	result, err := s.ws.CreateDraft(articles)
	if err != nil {
		return nil, err
	}

	return &DraftResult{
		MediaID:  result.MediaID,
		DraftURL: result.DraftURL,
	}, nil
}

// UpdateDraftFromFile 从 JSON 文件更新已有草稿中的指定图文
func (s *Service) UpdateDraftFromFile(mediaID string, index int, jsonFile string) (*DraftResult, error) {
	s.log.Info("updating draft from file",
		zap.String("media_id", mediaID),
		zap.Int("index", index),
		zap.String("file", jsonFile))

	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var req DraftRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	article, err := articleForUpdate(req, index)
	if err != nil {
		return nil, err
	}

	sdkArticle, err := buildSDKArticle(article)
	if err != nil {
		return nil, fmt.Errorf("article %d: %w", index, err)
	}

	result, err := s.ws.UpdateDraft(mediaID, index, sdkArticle)
	if err != nil {
		return nil, err
	}

	return &DraftResult{
		MediaID:  result.MediaID,
		DraftURL: result.DraftURL,
	}, nil
}

// CreateDraft 创建草稿
func (s *Service) CreateDraft(articles []Article) (*DraftResult, error) {
	// 转换为 SDK 格式
	draftArticles, err := buildSDKArticles(articles)
	if err != nil {
		return nil, err
	}

	// 调用微信 API
	result, err := s.ws.CreateDraft(draftArticles)
	if err != nil {
		return nil, err
	}

	return &DraftResult{
		MediaID:  result.MediaID,
		DraftURL: result.DraftURL,
	}, nil
}

func buildSDKArticles(articles []Article) ([]*draft.Article, error) {
	sdkArticles := make([]*draft.Article, 0, len(articles))
	for i, article := range articles {
		sdkArticle, err := buildSDKArticle(article)
		if err != nil {
			return nil, fmt.Errorf("article %d: %w", i, err)
		}
		sdkArticles = append(sdkArticles, sdkArticle)
	}
	return sdkArticles, nil
}

func articleForUpdate(req DraftRequest, index int) (Article, error) {
	if len(req.Articles) == 0 {
		return Article{}, fmt.Errorf("no articles in request")
	}
	if index < 0 || index >= len(req.Articles) {
		return Article{}, fmt.Errorf("article index %d out of range", index)
	}
	return req.Articles[index], nil
}

func buildSDKArticle(article Article) (*draft.Article, error) {
	if article.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if article.Content == "" {
		return nil, fmt.Errorf("content is required")
	}

	sdkArticle := &draft.Article{
		Title:   article.Title,
		Content: article.Content,
		Digest:  article.Digest,
		Author:  article.Author,
	}

	if article.ThumbMediaID != "" {
		sdkArticle.ThumbMediaID = article.ThumbMediaID
		sdkArticle.ShowCoverPic = uint(article.ShowCoverPic)
	}

	if article.ContentSourceURL != "" {
		sdkArticle.ContentSourceURL = article.ContentSourceURL
	}

	return sdkArticle, nil
}

// GenerateDigestFromContent 从内容生成摘要
func GenerateDigestFromContent(content string, maxLen int) string {
	if maxLen == 0 {
		maxLen = 120
	}

	content = stripHTML(content)
	if content == "" {
		return ""
	}

	runes := []rune(content)
	if len(runes) > maxLen {
		content = string(runes[:maxLen]) + "..."
	}

	return content
}

var (
	scriptStylePattern = regexp.MustCompile(`(?is)<(script|style)\b[^>]*>.*?</(script|style)>`)
	blockTagPattern    = regexp.MustCompile(`(?i)</?(p|div|h[1-6]|li|tr|blockquote|section|article)[^>]*>|<br\s*/?>`)
	anyTagPattern      = regexp.MustCompile(`(?s)<[^>]+>`)
)

// stripHTML 去除 HTML 标签并规范化文本
func stripHTML(html string) string {
	result := scriptStylePattern.ReplaceAllString(html, " ")
	result = blockTagPattern.ReplaceAllString(result, "\n")
	result = anyTagPattern.ReplaceAllString(result, " ")
	result = stdhtml.UnescapeString(result)

	lines := strings.Split(result, "\n")
	cleanLines := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n")
}

package wechat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"github.com/silenceper/wechat/v2"
	wechatcache "github.com/silenceper/wechat/v2/cache"
	"github.com/silenceper/wechat/v2/officialaccount"
	wechatconfig "github.com/silenceper/wechat/v2/officialaccount/config"
	"github.com/silenceper/wechat/v2/officialaccount/draft"
	"github.com/silenceper/wechat/v2/officialaccount/material"
	"go.uber.org/zap"
)

// Service 微信服务
type Service struct {
	cfg *config.Config
	log *zap.Logger
	wc  *wechat.Wechat
}

// NewService 创建微信服务
func NewService(cfg *config.Config, log *zap.Logger) *Service {
	return &Service{
		cfg: cfg,
		log: log,
		wc:  wechat.NewWechat(),
	}
}

// getOfficialAccount 获取公众号实例
func (s *Service) getOfficialAccount() *officialaccount.OfficialAccount {
	memory := wechatcache.NewMemory()
	wechatCfg := &wechatconfig.Config{
		AppID:     s.cfg.WechatAppID,
		AppSecret: s.cfg.WechatSecret,
		Cache:     memory,
	}
	return s.wc.GetOfficialAccount(wechatCfg)
}

// UploadMaterialResult 上传素材结果
type UploadMaterialResult struct {
	MediaID   string `json:"media_id"`
	WechatURL string `json:"wechat_url"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

// UploadMaterial 上传素材到微信
func (s *Service) UploadMaterial(filePath string) (*UploadMaterialResult, error) {
	startTime := time.Now()
	oa := s.getOfficialAccount()
	mat := oa.GetMaterial()

	// 调用微信 API 上传（SDK 接受文件路径字符串）
	mediaID, url, err := mat.AddMaterial(material.MediaTypeImage, filePath)
	if err != nil {
		s.log.Error("upload material failed",
			zap.String("path", filePath),
			zap.Error(err))
		return nil, fmt.Errorf("upload material: %w", err)
	}

	duration := time.Since(startTime)
	s.log.Info("material uploaded",
		zap.String("path", filePath),
		zap.String("media_id", maskMediaID(mediaID)),
		zap.Duration("duration", duration))

	return &UploadMaterialResult{
		MediaID:   mediaID,
		WechatURL: url,
	}, nil
}

// CreateDraftResult 创建草稿结果
type CreateDraftResult struct {
	MediaID  string `json:"media_id"`
	DraftURL string `json:"draft_url,omitempty"`
}

// CreateDraft 创建草稿
func (s *Service) CreateDraft(articles []*draft.Article) (*CreateDraftResult, error) {
	startTime := time.Now()
	oa := s.getOfficialAccount()
	dm := oa.GetDraft()

	// 直接调用 SDK 方法，SDK 接受 []*draft.Article
	mediaID, err := dm.AddDraft(articles)
	if err != nil {
		s.log.Error("create draft failed", zap.Error(err))
		return nil, fmt.Errorf("create draft: %w", err)
	}

	duration := time.Since(startTime)
	s.log.Info("draft created",
		zap.String("media_id", maskMediaID(mediaID)),
		zap.Duration("duration", duration))

	// 构造草稿 URL
	draftURL := fmt.Sprintf("https://mp.weixin.qq.com/cgi-bin/appmsg?t=media/appmsg_edit_v2&action=edit&createType=0&token=")

	return &CreateDraftResult{
		MediaID:  mediaID,
		DraftURL: draftURL,
	}, nil
}

// UploadMaterialFromBytes 从字节数据上传素材
func (s *Service) UploadMaterialFromBytes(data []byte, filename string) (*UploadMaterialResult, error) {
	// 创建临时文件
	tmpDir := os.TempDir()
	tmpPath := filepath.Join(tmpDir, "md2wechat_"+filename)
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	defer os.Remove(tmpPath)

	return s.UploadMaterial(tmpPath)
}

// AccessTokenResult 获取 access_token 结果（用于调试）
type AccessTokenResult struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// GetAccessToken 获取 access_token（调试用）
func (s *Service) GetAccessToken() (*AccessTokenResult, error) {
	oa := s.getOfficialAccount()
	accessToken, err := oa.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	return &AccessTokenResult{
		AccessToken: accessToken,
		ExpiresIn:   7200, // 微信默认 7200 秒
	}, nil
}

// maskMediaID 遮蔽 media_id 用于日志
func maskMediaID(id string) string {
	if id == "" || len(id) < 8 {
		return "***"
	}
	return id[:4] + "***" + id[len(id)-4:]
}

// UploadMaterialWithRetry 带重试的上传
func (s *Service) UploadMaterialWithRetry(filePath string, maxRetries int) (*UploadMaterialResult, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		result, err := s.UploadMaterial(filePath)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if i < maxRetries-1 {
			time.Sleep(time.Second)
		}
	}
	return nil, lastErr
}

// DownloadFile 下载文件到临时目录，或返回本地文件路径
// 如果传入的是本地文件路径（不以 http:// 或 https:// 开头），则直接返回该路径
func DownloadFile(urlOrPath string) (string, error) {
	// 检查是否是本地文件路径（不是 HTTP URL）
	if !strings.HasPrefix(urlOrPath, "http://") && !strings.HasPrefix(urlOrPath, "https://") {
		// 本地文件 - 检查是否存在
		if _, err := os.Stat(urlOrPath); err == nil {
			return urlOrPath, nil // 直接返回本地路径
		}
		return "", fmt.Errorf("local file not found: %s", urlOrPath)
	}

	// HTTP URL - 下载文件
	url := urlOrPath

	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// 发起请求
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// 创建临时文件
	tmpDir := os.TempDir()
	// 从 URL 路径中提取扩展名，排除查询参数
	ext := ".jpg" // 默认扩展名
	if parsedURL, err := neturl.Parse(url); err == nil {
		if pathExt := filepath.Ext(parsedURL.Path); pathExt != "" {
			ext = pathExt
		}
	}
	tmpPath := filepath.Join(tmpDir, "md2wechat_download_"+ext)
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer tmpFile.Close()

	// 写入文件
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("write file: %w", err)
	}

	return tmpPath, nil
}

// CreateMultipartFormData 创建 multipart 表单数据
func CreateMultipartFormData(fieldName, filename string, data []byte) (string, *bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		writer.Close()
		return "", nil, ""
	}

	if _, err := part.Write(data); err != nil {
		writer.Close()
		return "", nil, ""
	}

	contentType := writer.FormDataContentType()
	writer.Close()

	return contentType, body, filename
}

// JSONMarshal 自定义 JSON 序列化
func JSONMarshal(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// NewspicImageItem 小绿书图片项
type NewspicImageItem struct {
	ImageMediaID string `json:"image_media_id"`
}

// NewspicImageInfo 小绿书图片信息
type NewspicImageInfo struct {
	ImageList []NewspicImageItem `json:"image_list"`
}

// NewspicArticle 小绿书文章
type NewspicArticle struct {
	Title              string           `json:"title"`
	Content            string           `json:"content"`
	ArticleType        string           `json:"article_type"`
	ImageInfo          NewspicImageInfo `json:"image_info"`
	NeedOpenComment    int              `json:"need_open_comment,omitempty"`
	OnlyFansCanComment int              `json:"only_fans_can_comment,omitempty"`
}

// NewspicDraftRequest 小绿书草稿请求
type NewspicDraftRequest struct {
	Articles []NewspicArticle `json:"articles"`
}

// NewspicDraftResponse 微信 API 响应
type NewspicDraftResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	MediaID string `json:"media_id"`
}

// CreateNewspicDraft 创建小绿书草稿（直接调用微信 API，SDK 不支持 newspic）
func (s *Service) CreateNewspicDraft(articles []NewspicArticle) (*CreateDraftResult, error) {
	startTime := time.Now()

	// 获取 access_token
	oa := s.getOfficialAccount()
	accessToken, err := oa.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	// 构造请求
	req := NewspicDraftRequest{Articles: articles}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 调用微信 API
	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/draft/add?access_token=%s", accessToken)

	httpResp, err := http.Post(apiURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("call wechat api: %w", err)
	}
	defer httpResp.Body.Close()

	// 解析响应
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var resp NewspicDraftResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// 检查错误
	if resp.ErrCode != 0 {
		s.log.Error("create newspic draft failed",
			zap.Int("errcode", resp.ErrCode),
			zap.String("errmsg", resp.ErrMsg))
		return nil, fmt.Errorf("wechat api error: %d - %s", resp.ErrCode, resp.ErrMsg)
	}

	duration := time.Since(startTime)
	s.log.Info("newspic draft created",
		zap.String("media_id", maskMediaID(resp.MediaID)),
		zap.Duration("duration", duration))

	return &CreateDraftResult{
		MediaID:  resp.MediaID,
		DraftURL: fmt.Sprintf("https://mp.weixin.qq.com/cgi-bin/appmsg?t=media/appmsg_edit_v2&action=edit&createType=0&token="),
	}, nil
}

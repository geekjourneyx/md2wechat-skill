package wechat

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
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

var (
	downloadLookupIP      = net.LookupIP
	newDownloadHTTPClient = func() *http.Client {
		return &http.Client{
			Timeout: 60 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return errors.New("stopped after 5 redirects")
				}
				return validateRemoteDownloadURL(req.URL)
			},
		}
	}
)

// Service 微信服务
type Service struct {
	cfg                *config.Config
	log                *zap.Logger
	wc                 *wechat.Wechat
	httpClient         *http.Client
	sleep              func(time.Duration)
	uploadMaterialFunc func(string) (*UploadMaterialResult, error)
}

// NewService 创建微信服务
func NewService(cfg *config.Config, log *zap.Logger) *Service {
	return &Service{
		cfg:        cfg,
		log:        log,
		wc:         wechat.NewWechat(),
		httpClient: &http.Client{Timeout: 60 * time.Second},
		sleep:      time.Sleep,
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
	if s.uploadMaterialFunc != nil {
		return s.uploadMaterialFunc(filePath)
	}

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

// PreviewDraftResult contains WeChat's preview-send response.
type PreviewDraftResult struct {
	MediaID    string `json:"media_id"`
	OpenID     string `json:"openid,omitempty"`
	WxName     string `json:"wxname,omitempty"`
	MsgID      int64  `json:"msg_id"`
	MsgDataID  int64  `json:"msg_data_id"`
	MsgStatus  string `json:"msg_status,omitempty"`
	PreviewURL string `json:"preview_url,omitempty"`
}

type previewDraftResponse struct {
	ErrCode   int    `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
	MsgID     int64  `json:"msg_id"`
	MsgDataID int64  `json:"msg_data_id"`
	MsgStatus string `json:"msg_status"`
}

// UpdateDraftRequest 微信草稿更新请求
type UpdateDraftRequest struct {
	MediaID  string         `json:"media_id"`
	Index    int            `json:"index"`
	Articles *draft.Article `json:"articles"`
}

// UpdateDraftResponse 微信草稿更新响应
type UpdateDraftResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
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
		return nil, fmt.Errorf("create draft: %w", ExplainDraftError(err))
	}

	duration := time.Since(startTime)
	s.log.Info("draft created",
		zap.String("media_id", maskMediaID(mediaID)),
		zap.Duration("duration", duration))

	return &CreateDraftResult{
		MediaID: mediaID,
	}, nil
}

// GetDraft 获取已有草稿图文列表
func (s *Service) GetDraft(mediaID string) ([]*draft.Article, error) {
	startTime := time.Now()
	oa := s.getOfficialAccount()
	dm := oa.GetDraft()

	articles, err := dm.GetDraft(mediaID)
	if err != nil {
		s.log.Error("get draft failed",
			zap.String("media_id", maskMediaID(mediaID)),
			zap.Error(err))
		return nil, fmt.Errorf("get draft: %w", ExplainDraftError(err))
	}

	duration := time.Since(startTime)
	s.log.Info("draft fetched",
		zap.String("media_id", maskMediaID(mediaID)),
		zap.Int("articles", len(articles)),
		zap.Duration("duration", duration))

	return articles, nil
}

// PreviewDraftToOpenID sends an mpnews draft preview to one follower by openid.
func (s *Service) PreviewDraftToOpenID(mediaID string, openID string) (*PreviewDraftResult, error) {
	return s.previewDraft(mediaID, openID, "")
}

// PreviewDraftToWxName sends an mpnews draft preview to one follower by WeChat ID.
func (s *Service) PreviewDraftToWxName(mediaID string, wxName string) (*PreviewDraftResult, error) {
	return s.previewDraft(mediaID, "", wxName)
}

func (s *Service) previewDraft(mediaID string, openID string, wxName string) (*PreviewDraftResult, error) {
	mediaID = strings.TrimSpace(mediaID)
	openID = strings.TrimSpace(openID)
	wxName = strings.TrimSpace(wxName)
	if mediaID == "" {
		return nil, errors.New("media_id is required")
	}
	if (openID == "") == (wxName == "") {
		return nil, errors.New("exactly one of openid or wxname is required")
	}
	startTime := time.Now()
	oa := s.getOfficialAccount()
	accessToken, err := oa.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	req := map[string]any{
		"mpnews": map[string]any{
			"media_id": mediaID,
		},
		"msgtype": "mpnews",
	}
	if openID != "" {
		req["touser"] = openID
	} else {
		req["towxname"] = wxName
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/message/mass/preview?access_token=%s", accessToken)
	httpReq, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := s.getHTTPClient().Do(httpReq)
	if err != nil {
		s.log.Error("preview draft failed",
			zap.String("media_id", maskMediaID(mediaID)),
			zap.Error(err))
		return nil, fmt.Errorf("call wechat api: %w", err)
	}
	defer func() {
		_ = httpResp.Body.Close()
	}()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var resp previewDraftResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if resp.ErrCode != 0 {
		return nil, ExplainDraftError(fmt.Errorf("wechat api error: %d - %s", resp.ErrCode, resp.ErrMsg))
	}

	duration := time.Since(startTime)
	s.log.Info("draft preview sent",
		zap.String("media_id", maskMediaID(mediaID)),
		zap.Duration("duration", duration))

	return &PreviewDraftResult{
		MediaID:   mediaID,
		OpenID:    openID,
		WxName:    wxName,
		MsgID:     resp.MsgID,
		MsgDataID: resp.MsgDataID,
		MsgStatus: resp.MsgStatus,
	}, nil
}

// UpdateDraft 更新已有草稿中的单篇图文
func (s *Service) UpdateDraft(mediaID string, index int, article *draft.Article) (*CreateDraftResult, error) {
	startTime := time.Now()

	oa := s.getOfficialAccount()
	accessToken, err := oa.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	req := UpdateDraftRequest{
		MediaID:  mediaID,
		Index:    index,
		Articles: article,
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/draft/update?access_token=%s", accessToken)
	httpReq, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := s.getHTTPClient().Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call wechat api: %w", err)
	}
	defer func() {
		_ = httpResp.Body.Close()
	}()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var resp UpdateDraftResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if resp.ErrCode != 0 {
		s.log.Error("update draft failed",
			zap.Int("errcode", resp.ErrCode),
			zap.String("errmsg", resp.ErrMsg))
		return nil, fmt.Errorf("%s", ExplainDraftAPIError(resp.ErrCode, resp.ErrMsg))
	}

	duration := time.Since(startTime)
	s.log.Info("draft updated",
		zap.String("media_id", maskMediaID(mediaID)),
		zap.Int("index", index),
		zap.Duration("duration", duration))

	return &CreateDraftResult{
		MediaID: mediaID,
	}, nil
}

// UploadMaterialFromBytes 从字节数据上传素材
func (s *Service) UploadMaterialFromBytes(data []byte, filename string) (*UploadMaterialResult, error) {
	ext := filepath.Ext(filepath.Base(filename))
	if ext == "." {
		ext = ""
	}

	tmpFile, err := os.CreateTemp("", "md2wechat-upload-*"+ext)
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("write temp file: %w", err)
	}

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
			s.getSleepFunc()(time.Second)
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
	parsedURL, err := neturl.Parse(url)
	if err != nil {
		return "", fmt.Errorf("parse download url: %w", err)
	}
	if err := validateRemoteDownloadURL(parsedURL); err != nil {
		return "", err
	}

	// 创建 HTTP 客户端
	client := newDownloadHTTPClient()

	// 发起请求
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("download file: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// 从 URL 路径中提取扩展名，排除查询参数
	ext := ".jpg" // 默认扩展名
	if pathExt := filepath.Ext(parsedURL.Path); pathExt != "" {
		ext = pathExt
	}
	tmpFile, err := os.CreateTemp("", "md2wechat-download-*"+ext)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// 写入文件
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("write file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("close temp file: %w", err)
	}

	return tmpPath, nil
}

func validateRemoteDownloadURL(parsedURL *neturl.URL) error {
	if parsedURL == nil {
		return fmt.Errorf("invalid download url")
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported download scheme: %s", parsedURL.Scheme)
	}

	host := parsedURL.Hostname()
	if host == "" {
		return fmt.Errorf("download url missing host")
	}
	if err := validateDownloadPort(parsedURL.Port()); err != nil {
		return err
	}
	if err := validateDownloadHost(host); err != nil {
		return err
	}
	return nil
}

func validateDownloadPort(port string) error {
	if port == "" || port == "80" || port == "443" {
		return nil
	}
	return fmt.Errorf("download url uses disallowed port: %s", port)
}

func validateDownloadHost(host string) error {
	lowerHost := strings.ToLower(strings.TrimSpace(host))
	if lowerHost == "" {
		return fmt.Errorf("download url missing host")
	}
	if lowerHost == "localhost" || strings.HasSuffix(lowerHost, ".localhost") {
		return fmt.Errorf("download host is not allowed: %s", host)
	}

	if ip := net.ParseIP(lowerHost); ip != nil {
		if err := validateDownloadIP(ip); err != nil {
			return fmt.Errorf("download host is not allowed: %w", err)
		}
		return nil
	}

	ips, err := downloadLookupIP(host)
	if err != nil {
		return fmt.Errorf("resolve download host %s: %w", host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("resolve download host %s: no addresses found", host)
	}
	for _, ip := range ips {
		if err := validateDownloadIP(ip); err != nil {
			return fmt.Errorf("download host is not allowed: %w", err)
		}
	}
	return nil
}

func validateDownloadIP(ip net.IP) error {
	if ip == nil {
		return fmt.Errorf("invalid ip")
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return fmt.Errorf("ip %s is private or local", ip.String())
	}
	return nil
}

// CreateMultipartFormData 创建 multipart 表单数据
func CreateMultipartFormData(fieldName, filename string, data []byte) (string, *bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	boundary := writer.Boundary()

	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		_ = writer.Close()
		return "", nil, ""
	}

	if _, err := part.Write(data); err != nil {
		_ = writer.Close()
		return "", nil, ""
	}

	contentType := writer.FormDataContentType()
	if err := writer.Close(); err != nil {
		return "", nil, ""
	}

	return contentType, body, boundary
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
	httpReq, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := s.getHTTPClient().Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call wechat api: %w", err)
	}
	defer func() {
		_ = httpResp.Body.Close()
	}()

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
		return nil, fmt.Errorf("%s", ExplainDraftAPIError(resp.ErrCode, resp.ErrMsg))
	}

	duration := time.Since(startTime)
	s.log.Info("newspic draft created",
		zap.String("media_id", maskMediaID(resp.MediaID)),
		zap.Duration("duration", duration))

	return &CreateDraftResult{
		MediaID: resp.MediaID,
	}, nil
}

func (s *Service) getSleepFunc() func(time.Duration) {
	if s != nil && s.sleep != nil {
		return s.sleep
	}
	return time.Sleep
}

func (s *Service) getHTTPClient() *http.Client {
	if s != nil && s.httpClient != nil {
		return s.httpClient
	}
	return &http.Client{Timeout: 60 * time.Second}
}

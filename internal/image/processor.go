package image

import (
	"context"
	"fmt"
	"os"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"go.uber.org/zap"
)

// DownloadFunc downloads a remote asset into a local temporary file.
type DownloadFunc func(string) (string, error)

// UploadFunc uploads a local file to the target backend.
type UploadFunc func(string) (*UploadResult, error)

// Option customizes a Processor.
type Option func(*Processor)

// WithDownloadFunc injects the download dependency used by remote assets.
func WithDownloadFunc(fn DownloadFunc) Option {
	return func(p *Processor) {
		p.downloadFile = fn
	}
}

// WithUploadFunc injects the upload dependency used by published assets.
func WithUploadFunc(fn UploadFunc) Option {
	return func(p *Processor) {
		p.uploadMaterial = fn
	}
}

// WithProvider injects a custom image-generation provider.
func WithProvider(provider Provider) Option {
	return func(p *Processor) {
		p.provider = provider
	}
}

// Processor 图片处理器
type Processor struct {
	cfg            *config.Config
	log            *zap.Logger
	compressor     *Compressor
	provider       Provider
	downloadFile   func(string) (string, error)
	uploadMaterial func(string) (*UploadResult, error)
}

// NewProcessor 创建图片处理器
func NewProcessor(cfg *config.Config, log *zap.Logger, opts ...Option) *Processor {
	if cfg == nil {
		cfg = &config.Config{}
	}
	if log == nil {
		log = zap.NewNop()
	}

	// 创建图片生成 Provider
	provider, err := NewProvider(cfg)
	if err != nil {
		// 如果配置了 API Key 但创建失败，记录警告
		if cfg.ImageAPIKey != "" {
			log.Warn("failed to create image provider, AI image generation will be unavailable", zap.Error(err))
		}
	}

	processor := &Processor{
		cfg:        cfg,
		log:        log,
		compressor: NewCompressor(log, cfg.MaxImageWidth, cfg.MaxImageSize),
		provider:   provider,
	}
	for _, opt := range opts {
		opt(processor)
	}
	return processor
}

// UploadResult 上传结果
type UploadResult struct {
	MediaID   string `json:"media_id"`
	WechatURL string `json:"wechat_url"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

// UploadLocalImage 上传本地图片
func (p *Processor) UploadLocalImage(filePath string) (*UploadResult, error) {
	p.log.Info("uploading local image", zap.String("path", filePath))

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	// 检查图片格式
	if !IsValidImageFormat(filePath) {
		return nil, fmt.Errorf("unsupported image format: %s", filePath)
	}

	// 如果需要压缩，先处理
	processedPath := filePath
	if p.cfg.CompressImages {
		compressedPath, compressed, err := p.compressor.CompressImage(filePath)
		if err != nil {
			p.log.Warn("compress failed, using original", zap.Error(err))
		} else if compressed {
			processedPath = compressedPath
			defer func() {
				_ = os.Remove(compressedPath)
			}()
			p.log.Info("using compressed image", zap.String("path", processedPath))
		}
	}

	// 上传到微信
	result, err := p.upload(processedPath)
	if err != nil {
		return nil, err
	}

	return &UploadResult{
		MediaID:   result.MediaID,
		WechatURL: result.WechatURL,
	}, nil
}

// DownloadAndUpload 下载在线图片并上传
func (p *Processor) DownloadAndUpload(url string) (*UploadResult, error) {
	p.log.Info("downloading and uploading image", zap.String("url", url))

	// 下载图片
	tmpPath, err := p.download(url)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	// 检查格式
	if !IsValidImageFormat(tmpPath) {
		return nil, fmt.Errorf("downloaded file is not a valid image")
	}

	// 压缩（如果需要）
	processedPath := tmpPath
	if p.cfg.CompressImages {
		compressedPath, compressed, err := p.compressor.CompressImage(tmpPath)
		if err != nil {
			p.log.Warn("compress failed, using original", zap.Error(err))
		} else if compressed {
			processedPath = compressedPath
			defer func() {
				_ = os.Remove(compressedPath)
			}()
			p.log.Info("using compressed image", zap.String("path", processedPath))
		}
	}

	// 上传到微信
	result, err := p.upload(processedPath)
	if err != nil {
		return nil, err
	}

	return &UploadResult{
		MediaID:   result.MediaID,
		WechatURL: result.WechatURL,
	}, nil
}

// GenerateAndUploadResult AI 生成图片结果
type GenerateAndUploadResult struct {
	Prompt      string `json:"prompt"`
	OriginalURL string `json:"original_url"`
	MediaID     string `json:"media_id"`
	WechatURL   string `json:"wechat_url"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

// GenerateAndUpload AI 生成图片并上传
func (p *Processor) GenerateAndUpload(prompt string) (*GenerateAndUploadResult, error) {
	p.log.Info("generating image via AI", zap.String("prompt", prompt))

	return p.generateAndUploadWithProvider(prompt, p.provider)
}

// GenerateAndUploadWithSubject generates an image guided by a portrait reference.
func (p *Processor) GenerateAndUploadWithSubject(prompt, subjectReference string) (*GenerateAndUploadResult, error) {
	provider, ok := p.provider.(SubjectReferenceProvider)
	if !ok {
		return nil, fmt.Errorf("configured image provider does not support subject references")
	}
	return p.generateAndUploadWithGenerator(prompt, func(ctx context.Context) (*GenerateResult, error) {
		return provider.GenerateWithSubject(ctx, prompt, subjectReference)
	})
}

// GenerateAndUploadWithSize AI 生成指定尺寸的图片并上传
func (p *Processor) GenerateAndUploadWithSize(prompt string, size string) (*GenerateAndUploadResult, error) {
	p.log.Info("generating image via AI with size",
		zap.String("prompt", prompt),
		zap.String("size", size))

	if err := p.cfg.ValidateForImageGeneration(); err != nil {
		return nil, err
	}

	cfgCopy := *p.cfg
	cfgCopy.ImageSize = size

	newProvider, err := NewProvider(&cfgCopy)
	if err != nil {
		return nil, fmt.Errorf("create provider with size: %w", err)
	}

	return p.generateAndUploadWithProvider(prompt, newProvider)
}

func (p *Processor) generateAndUploadWithProvider(prompt string, provider Provider) (*GenerateAndUploadResult, error) {
	var generate func(context.Context) (*GenerateResult, error)
	if provider != nil {
		generate = func(ctx context.Context) (*GenerateResult, error) {
			return provider.Generate(ctx, prompt)
		}
	}
	return p.generateAndUploadWithGenerator(prompt, generate)
}

func (p *Processor) generateAndUploadWithGenerator(prompt string, generate func(context.Context) (*GenerateResult, error)) (*GenerateAndUploadResult, error) {
	if err := p.cfg.ValidateForImageGeneration(); err != nil {
		return nil, err
	}
	if generate == nil {
		return nil, fmt.Errorf("图片生成服务未配置，请检查配置文件中的 api.image_provider 和 api.image_key")
	}

	ctx := context.Background()
	result, err := generate(ctx)
	if err != nil {
		return nil, fmt.Errorf("generate image: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("generate image: provider returned no result")
	}
	p.log.Info("image generated",
		zap.String("url", result.URL),
		zap.String("provider", result.Model),
		zap.String("size", result.Size))
	if result.URL == "" {
		return nil, fmt.Errorf("generate image: provider returned empty image URL/path")
	}

	// 下载生成的图片
	tmpPath, err := p.download(result.URL)
	if err != nil {
		return nil, fmt.Errorf("download generated image: %w", err)
	}
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	processedPath := tmpPath
	if p.cfg.CompressImages {
		compressedPath, compressed, err := p.compressor.CompressImage(tmpPath)
		if err != nil {
			p.log.Warn("compress failed, using original", zap.Error(err))
		} else if compressed {
			processedPath = compressedPath
			defer func() {
				_ = os.Remove(compressedPath)
			}()
			p.log.Info("using compressed image", zap.String("path", processedPath))
		}
	}

	// 上传到微信
	uploadResult, err := p.upload(processedPath)
	if err != nil {
		return nil, err
	}

	return &GenerateAndUploadResult{
		Prompt:      prompt,
		OriginalURL: result.URL,
		MediaID:     uploadResult.MediaID,
		WechatURL:   uploadResult.WechatURL,
	}, nil
}

// GetImageInfo 获取图片信息
func (p *Processor) GetImageInfo(filePath string) (*ImageInfo, error) {
	return GetImageInfo(filePath)
}

// CompressImage 压缩图片（公开方法）
func (p *Processor) CompressImage(filePath string) (string, bool, error) {
	return p.compressor.CompressImage(filePath)
}

// SetCompressQuality 设置压缩质量
func (p *Processor) SetCompressQuality(quality int) {
	p.compressor.SetQuality(quality)
}

func (p *Processor) download(url string) (string, error) {
	if p.downloadFile != nil {
		return p.downloadFile(url)
	}
	return "", fmt.Errorf("download helper is not configured")
}

func (p *Processor) upload(filePath string) (*UploadResult, error) {
	if p.uploadMaterial != nil {
		return p.uploadMaterial(filePath)
	}
	return nil, fmt.Errorf("upload helper is not configured")
}

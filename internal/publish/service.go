package publish

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/geekjourneyx/md2wechat-skill/internal/converter"
	"github.com/geekjourneyx/md2wechat-skill/internal/image"
	"go.uber.org/zap"
)

// MarkdownConverter is the conversion dependency needed by the publish service.
type MarkdownConverter interface {
	Convert(req *converter.ConvertRequest) *converter.ConvertResult
}

// AssetProcessor is the asset-processing dependency needed by the publish service.
type AssetProcessor interface {
	UploadLocalImage(filePath string) (*image.UploadResult, error)
	DownloadAndUpload(url string) (*image.UploadResult, error)
	GenerateAndUpload(prompt string) (*image.GenerateAndUploadResult, error)
}

// DraftCreator is the draft adapter dependency needed by the publish service.
type DraftCreator interface {
	CreateDraft(artifact Artifact) (*DraftResult, error)
}

// CoverUploader uploads a cover image and returns the target platform media ID.
type CoverUploader func(string) (string, error)

// ConvertInput is the normalized publish request consumed by the service.
type ConvertInput struct {
	Source         ArticleSource
	Intent         PublishIntent
	ConvertRequest *converter.ConvertRequest
	MarkdownDir    string
	OutputFile     string
	SaveDraftPath  string
	CoverImagePath string
	CoverMediaID   string
}

// ConvertOutput is the normalized publish result returned by the service.
type ConvertOutput struct {
	Artifact     Artifact
	Conversion   *converter.ConvertResult
	DraftSaved   string
	DraftResult  *DraftResult
	CoverMediaID string
}

// Service coordinates the publish pipeline without binding it to the CLI layer.
type Service struct {
	log         *zap.Logger
	converter   MarkdownConverter
	assets      *AssetPipeline
	drafts      DraftCreator
	uploadCover CoverUploader
}

// NewService creates a publish pipeline service.
func NewService(log *zap.Logger, conv MarkdownConverter, assets AssetProcessor, drafts DraftCreator, uploadCover CoverUploader) *Service {
	return &Service{
		log:         log,
		converter:   conv,
		assets:      NewAssetPipeline(assets),
		drafts:      drafts,
		uploadCover: uploadCover,
	}
}

// Convert executes the normalized publish pipeline.
func (s *Service) Convert(input *ConvertInput) (*ConvertOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("convert input is required")
	}
	if input.ConvertRequest == nil {
		return nil, fmt.Errorf("convert request is required")
	}
	if s.converter == nil {
		return nil, fmt.Errorf("markdown converter is required")
	}
	if err := validateConvertInput(input); err != nil {
		return nil, err
	}
	if input.Intent.CreateDraft {
		if s.drafts == nil {
			return nil, fmt.Errorf("draft creator is required")
		}
		if strings.TrimSpace(input.CoverImagePath) != "" && s.uploadCover == nil {
			return nil, fmt.Errorf("cover uploader is required")
		}
	}

	result := s.converter.Convert(input.ConvertRequest)
	if result == nil {
		return nil, fmt.Errorf("converter returned nil result")
	}

	output := &ConvertOutput{
		Conversion: result,
		Artifact: Artifact{
			OutputFile: input.OutputFile,
			Metadata:   input.Source.Metadata,
			Assets:     assetRefsFromImages(result.Images, input.MarkdownDir),
		},
	}

	if converter.IsAIRequest(result) {
		return output, nil
	}
	if !result.Success {
		return nil, fmt.Errorf("conversion failed: %s", result.Error)
	}

	output.Artifact.HTML = result.HTML

	if input.Intent.Upload || input.Intent.CreateDraft {
		if s.assets == nil {
			return nil, fmt.Errorf("asset pipeline is required")
		}
		assetOutput, err := s.assets.Process(&ProcessInput{
			HTML:        output.Artifact.HTML,
			Assets:      output.Artifact.Assets,
			MarkdownDir: input.MarkdownDir,
		})
		if err != nil {
			return nil, &AssetError{Err: err}
		}
		output.Artifact.HTML = assetOutput.HTML
		output.Artifact.Assets = assetOutput.Assets
	}

	if input.SaveDraftPath != "" {
		if err := s.saveDraft(output.Artifact, input.SaveDraftPath); err != nil {
			return nil, &DraftSaveError{Err: err}
		}
		output.DraftSaved = input.SaveDraftPath
	}

	if input.Intent.CreateDraft {
		if s.drafts == nil {
			return nil, fmt.Errorf("draft creator is required")
		}
		coverMediaID := input.CoverMediaID
		if strings.TrimSpace(coverMediaID) == "" {
			if s.uploadCover == nil {
				return nil, fmt.Errorf("cover uploader is required")
			}
			var err error
			coverMediaID, err = s.uploadCover(input.CoverImagePath)
			if err != nil {
				return nil, &DraftCreateError{Err: fmt.Errorf("上传封面图片失败: %w", err)}
			}
		}
		output.CoverMediaID = coverMediaID
		output.Artifact.CoverMediaID = coverMediaID

		draftResult, err := s.drafts.CreateDraft(output.Artifact)
		if err != nil {
			return nil, &DraftCreateError{Err: fmt.Errorf("create draft: %w", err)}
		}
		output.DraftResult = draftResult
		output.Artifact.DraftMediaID = draftResult.MediaID
		output.Artifact.DraftURL = draftResult.DraftURL
	}

	return output, nil
}

func validateConvertInput(input *ConvertInput) error {
	coverPath := input.CoverImagePath
	coverMediaID := strings.TrimSpace(input.CoverMediaID)
	hasCoverPath := strings.TrimSpace(coverPath) != ""
	hasCoverMediaID := coverMediaID != ""

	if !input.Intent.CreateDraft {
		if hasCoverPath || hasCoverMediaID {
			return &DraftError{
				Message: "--cover 和 --cover-media-id 仅可用于创建草稿",
				Hint:    "请移除封面参数，或使用 --draft 创建微信草稿",
			}
		}
		return nil
	}

	if hasCoverPath && hasCoverMediaID {
		return &DraftError{
			Message: "创建草稿时 --cover 和 --cover-media-id 不能同时使用",
			Hint:    "请二选一：提供本地封面图片路径，或提供已存在的微信永久素材 media_id",
		}
	}
	if !hasCoverPath && !hasCoverMediaID {
		return &DraftError{
			Message: "创建草稿需要封面图片",
			Hint: "请使用 --cover 参数指定本地封面图片路径，例如: --cover /path/to/cover.jpg\n" +
				"或者使用 --cover-media-id 提供已存在的微信永久素材 media_id",
		}
	}
	if hasCoverMediaID {
		lowerMediaID := strings.ToLower(coverMediaID)
		if strings.HasPrefix(lowerMediaID, "http://") || strings.HasPrefix(lowerMediaID, "https://") {
			return &DraftError{
				Message: "--cover-media-id 需要微信永久素材 media_id，不能使用 URL",
				Hint:    "远程图片请先下载到本地并通过 --cover 指定，或提供已有的 media_id",
			}
		}
		return nil
	}

	info, err := os.Stat(coverPath)
	if err != nil {
		return &DraftError{
			Message: fmt.Sprintf("封面图片不可用: %s", coverPath),
			Hint:    "请确认 --cover 指向存在且可读取的本地图片文件",
		}
	}
	if info.IsDir() {
		return &DraftError{
			Message: fmt.Sprintf("封面图片路径不能是目录: %s", coverPath),
			Hint:    "请使用 --cover 指定本地图片文件",
		}
	}
	if !image.IsValidImageFormat(coverPath) {
		return &DraftError{
			Message: fmt.Sprintf("不支持的封面图片格式: %s", filepath.Ext(coverPath)),
			Hint:    "支持的格式: .jpg、.jpeg、.png、.gif、.bmp、.webp",
		}
	}
	return nil
}

func (s *Service) saveDraft(artifact Artifact, filePath string) error {
	jsonData, err := json.MarshalIndent(buildDraftPayload(artifact), "", "  ")
	if err != nil {
		return fmt.Errorf("marshal draft: %w", err)
	}
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("write draft file: %w", err)
	}
	return nil
}

func assetRefsFromImages(images []converter.ImageRef, markdownDir string) []AssetRef {
	if len(images) == 0 {
		return nil
	}

	assets := make([]AssetRef, 0, len(images))
	for _, img := range images {
		asset := AssetRef{
			Index:       img.Index,
			Source:      img.Original,
			Placeholder: img.Placeholder,
			Prompt:      img.AIPrompt,
		}
		switch img.Type {
		case converter.ImageTypeLocal:
			asset.Kind = AssetKindLocal
			if img.Original != "" {
				if filepath.IsAbs(img.Original) {
					asset.ResolvedSource = img.Original
				} else if markdownDir != "" {
					asset.ResolvedSource = filepath.Join(markdownDir, img.Original)
				}
			}
		case converter.ImageTypeOnline:
			asset.Kind = AssetKindRemote
		case converter.ImageTypeAI:
			asset.Kind = AssetKindAI
		}
		assets = append(assets, asset)
	}
	return assets
}

func buildDraftPayload(artifact Artifact) map[string]any {
	return map[string]any{
		"articles": []map[string]any{
			{
				"title":          artifact.Metadata.Title,
				"author":         artifact.Metadata.Author,
				"digest":         artifact.Metadata.Digest,
				"content":        artifact.HTML,
				"thumb_media_id": artifact.CoverMediaID,
				"show_cover_pic": showCover(artifact.CoverMediaID),
			},
		},
	}
}

// AssetError reports failures in the asset-processing stage.
type AssetError struct {
	Err error
}

func (e *AssetError) Error() string {
	if e == nil || e.Err == nil {
		return "asset processing failed"
	}
	return e.Err.Error()
}

func (e *AssetError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// DraftSaveError reports failures while writing a draft JSON artifact.
type DraftSaveError struct {
	Err error
}

func (e *DraftSaveError) Error() string {
	if e == nil || e.Err == nil {
		return "save draft failed"
	}
	return e.Err.Error()
}

func (e *DraftSaveError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// DraftCreateError reports failures while publishing a draft to the backend.
type DraftCreateError struct {
	Err error
}

func (e *DraftCreateError) Error() string {
	if e == nil || e.Err == nil {
		return "create draft failed"
	}
	return e.Err.Error()
}

func (e *DraftCreateError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func IsAssetError(err error) bool {
	var target *AssetError
	return errors.As(err, &target)
}

func IsDraftSaveError(err error) bool {
	var target *DraftSaveError
	return errors.As(err, &target)
}

func IsDraftCreateError(err error) bool {
	var target *DraftCreateError
	return errors.As(err, &target)
}

func showCover(coverMediaID string) int {
	if coverMediaID == "" {
		return 0
	}
	return 1
}

// DraftError keeps the user-facing hint for draft creation failures in the publish layer.
type DraftError struct {
	Message string
	Hint    string
}

func (e *DraftError) Error() string {
	msg := fmt.Sprintf("草稿错误: %s", e.Message)
	if e.Hint != "" {
		msg += fmt.Sprintf("\n💡 提示:\n   %s", e.Hint)
	}
	return msg
}

package publish

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/geekjourneyx/md2wechat-skill/internal/converter"
)

// ImagePostInput is the normalized image-post request consumed by the publish layer.
type ImagePostInput struct {
	Title        string
	Content      string
	Images       []string
	FromMarkdown string
	OpenComment  bool
	FansOnly     bool
}

// ImagePostPreviewImage describes one preview image entry.
type ImagePostPreviewImage struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Size   int64  `json:"size,omitempty"`
}

// ImagePostPreview is the dry-run view returned for image posts.
type ImagePostPreview struct {
	Title       string                  `json:"title"`
	Content     string                  `json:"content,omitempty"`
	ImageCount  int                     `json:"image_count"`
	Images      []ImagePostPreviewImage `json:"images"`
	OpenComment bool                    `json:"open_comment,omitempty"`
	FansOnly    bool                    `json:"fans_only,omitempty"`
}

// ImagePostCreator is the adapter dependency needed by the image-post flow.
type ImagePostCreator interface {
	CreateImagePost(artifact ImagePostArtifact) (*ImagePostResult, error)
}

// ImagePostService coordinates image-post preparation and publishing without binding to the CLI.
type ImagePostService struct {
	assets  *AssetPipeline
	creator ImagePostCreator
}

// NewImagePostService creates the publish-layer image-post service.
func NewImagePostService(assets AssetProcessor, creator ImagePostCreator) *ImagePostService {
	return &ImagePostService{
		assets:  NewAssetPipeline(assets),
		creator: creator,
	}
}

// PreviewImagePost validates and previews a normalized image-post request.
func (s *ImagePostService) PreviewImagePost(input *ImagePostInput) (*ImagePostPreview, error) {
	source, err := normalizeImagePostInput(input)
	if err != nil {
		return nil, err
	}
	assets, err := validateAndResolveAssets(&ProcessInput{Assets: source.Assets})
	if err != nil {
		return nil, err
	}
	source.Assets = assets

	preview := &ImagePostPreview{
		Title:       source.Title,
		Content:     source.Content,
		ImageCount:  len(source.Assets),
		Images:      make([]ImagePostPreviewImage, 0, len(source.Assets)),
		OpenComment: source.OpenComment,
		FansOnly:    source.FansOnly,
	}

	for _, asset := range source.Assets {
		detail := ImagePostPreviewImage{
			Path: firstNonEmptyImagePost(asset.ResolvedSource, asset.Source),
		}
		switch asset.Kind {
		case AssetKindLocal:
			info, err := os.Stat(detail.Path)
			if err != nil {
				return nil, err
			}
			detail.Exists = true
			detail.Size = info.Size()
		case AssetKindRemote, AssetKindAI:
			detail.Exists = true
		}
		preview.Images = append(preview.Images, detail)
	}

	return preview, nil
}

// CreateImagePost uploads assets and dispatches the canonical artifact to the draft adapter.
func (s *ImagePostService) CreateImagePost(input *ImagePostInput) (*ImagePostResult, error) {
	source, err := normalizeImagePostInput(input)
	if err != nil {
		return nil, err
	}
	if s.assets == nil {
		return nil, fmt.Errorf("asset pipeline is required")
	}
	if s.creator == nil {
		return nil, fmt.Errorf("image post creator is required")
	}

	output, err := s.assets.Process(&ProcessInput{
		Assets: source.Assets,
	})
	if err != nil {
		return nil, err
	}

	artifact := ImagePostArtifact{
		Title:       source.Title,
		Content:     source.Content,
		Assets:      output.Assets,
		OpenComment: source.OpenComment,
		FansOnly:    source.FansOnly,
	}
	return s.creator.CreateImagePost(artifact)
}

func normalizeImagePostInput(input *ImagePostInput) (*ImagePostSource, error) {
	if input == nil {
		return nil, fmt.Errorf("image post input is required")
	}
	if input.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if input.FansOnly && !input.OpenComment {
		return nil, fmt.Errorf("fans-only comments require open-comment")
	}

	assets := make([]AssetRef, 0, len(input.Images))
	for _, imagePath := range input.Images {
		if imagePath == "" {
			continue
		}
		asset := AssetRef{
			Index:  len(assets),
			Kind:   AssetKindLocal,
			Source: imagePath,
		}
		if filepath.IsAbs(imagePath) {
			asset.ResolvedSource = imagePath
		}
		assets = append(assets, asset)
	}

	if input.FromMarkdown != "" {
		extracted, err := ExtractAssetsFromMarkdown(input.FromMarkdown)
		if err != nil {
			return nil, err
		}
		for _, asset := range extracted {
			asset.Index = len(assets)
			assets = append(assets, asset)
		}
	}

	if len(assets) == 0 {
		return nil, fmt.Errorf("no images provided")
	}
	if len(assets) > 20 {
		return nil, fmt.Errorf("too many images: %d (max 20)", len(assets))
	}

	return &ImagePostSource{
		Title:       input.Title,
		Content:     input.Content,
		Assets:      assets,
		OpenComment: input.OpenComment,
		FansOnly:    input.FansOnly,
	}, nil
}

// ExtractAssetsFromMarkdown resolves markdown images from a markdown file into publish assets.
func ExtractAssetsFromMarkdown(mdFile string) ([]AssetRef, error) {
	content, err := os.ReadFile(mdFile)
	if err != nil {
		return nil, err
	}

	mdDir := filepath.Dir(mdFile)
	assets := make([]AssetRef, 0)
	for _, img := range converter.ParseMarkdownImages(string(content)) {
		asset := AssetRef{
			Index:  len(assets),
			Source: img.Original,
		}

		switch img.Type {
		case converter.ImageTypeLocal:
			resolved := img.Original
			if !filepath.IsAbs(resolved) {
				resolved = filepath.Join(mdDir, img.Original)
			}
			asset.Kind = AssetKindLocal
			asset.ResolvedSource = resolved
		case converter.ImageTypeOnline:
			asset.Kind = AssetKindRemote
		case converter.ImageTypeAI:
			asset.Kind = AssetKindAI
			asset.Prompt = img.AIPrompt
		default:
			continue
		}

		assets = append(assets, asset)
	}

	return assets, nil
}

func firstNonEmptyImagePost(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/geekjourneyx/md2wechat-skill/internal/atomicfile"
	"github.com/geekjourneyx/md2wechat-skill/internal/draft"
	"github.com/geekjourneyx/md2wechat-skill/internal/publish"
	"github.com/spf13/cobra"
)

type imagePostService interface {
	PreviewImagePost(input *publish.ImagePostInput) (*publish.ImagePostPreview, error)
	CreateImagePost(input *publish.ImagePostInput) (*publish.ImagePostResult, error)
}

var (
	imagePostTitle       string
	imagePostContent     string
	imagePostImages      string
	imagePostFromMD      string
	imagePostOpenComment bool
	imagePostFansOnly    bool
	imagePostDryRun      bool
	imagePostOutput      string

	newImagePostService = func() imagePostService {
		return publish.NewImagePostService(newRuntimeImageProcessor(), draft.NewImagePostCreator(cfg, log))
	}
	isTerminalFn = isTerminal
)

var createImagePostCmd = &cobra.Command{
	Use:   "create_image_post",
	Short: "Create WeChat image post (小绿书/newspic)",
	Long: `Create a WeChat Official Account image post (小绿书/图片消息).

This command allows you to create image-only posts (newspic type) with up to 20 images.

Examples:
  # Create with comma-separated images
  md2wechat create_image_post -t "Weekend Trip" --images photo1.jpg,photo2.jpg,photo3.jpg

  # Extract images from Markdown file
  md2wechat create_image_post -t "Travel Diary" -m article.md

  # With description and comment settings
  md2wechat create_image_post -t "Food Blog" -c "Today's lunch" --images food.jpg --open-comment

  # Read description from stdin
  echo "Daily check-in" | md2wechat create_image_post -t "Daily" --images pic.jpg

  # Preview mode (dry-run)
  md2wechat create_image_post -t "Test" --images a.jpg,b.jpg --dry-run`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		response, err := runCreateImagePost()
		if err != nil {
			return err
		}
		if imagePostDryRun {
			responseSuccessWith(codeImagePostPreviewReady, "Image post preview prepared", response)
			return nil
		}
		responseSuccessWith(codeImagePostCreated, "Image post created successfully", response)
		return nil
	},
}

func runCreateImagePost() (any, error) {
	req := &publish.ImagePostInput{
		Title:       imagePostTitle,
		Content:     imagePostContent,
		OpenComment: imagePostOpenComment,
		FansOnly:    imagePostFansOnly,
	}

	if imagePostImages != "" {
		for _, img := range strings.Split(imagePostImages, ",") {
			img = strings.TrimSpace(img)
			if img != "" {
				req.Images = append(req.Images, img)
			}
		}
	}

	if imagePostFromMD != "" {
		req.FromMarkdown = imagePostFromMD
	}

	if imagePostContent == "" && !isTerminalFn() {
		scanner := bufio.NewScanner(os.Stdin)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if len(lines) > 0 {
			req.Content = strings.Join(lines, "\n")
		}
	}

	if req.Title == "" {
		return nil, newCLIError(codeImagePostInvalid, "--title is required")
	}

	if len(req.Images) == 0 && req.FromMarkdown == "" {
		return nil, newCLIError(codeImagePostInvalid, "--images or --from-markdown is required")
	}

	if err := resolveExplicitWeChatAccountIfProvided(); err != nil {
		return nil, err
	}

	svc := newImagePostService()
	preview, err := svc.PreviewImagePost(req)
	if err != nil {
		return nil, wrapCLIError(codeImagePostPreviewFailed, err, err.Error())
	}

	if imagePostDryRun {
		if imagePostOutput != "" {
			data, _ := json.MarshalIndent(preview, "", "  ")
			if err := os.WriteFile(imagePostOutput, data, 0644); err != nil {
				return nil, wrapCLIError(codeImagePostPreviewFailed, err, err.Error())
			}
		}

		return map[string]any{
			"mode":    "dry-run",
			"preview": preview,
		}, nil
	}

	if imagePostOutput != "" {
		if err := atomicfile.Probe(imagePostOutput); err != nil {
			return nil, wrapCLIError(
				codeImagePostCreateFailed,
				err,
				err.Error(),
			)
		}
	}

	if err := prepareWeChatSideEffect(); err != nil {
		return nil, err
	}

	result, err := svc.CreateImagePost(req)
	if err != nil {
		return nil, wrapCLIError(codeImagePostCreateFailed, err, err.Error())
	}

	if imagePostOutput != "" {
		data, _ := json.MarshalIndent(result, "", "  ")
		if _, err := atomicfile.Write(imagePostOutput, data); err != nil {
			return nil, newCLIErrorWithDetails(
				codeImagePostCreateFailed,
				fmt.Sprintf(
					"image post draft %s was created, but saving --output failed: %v",
					result.MediaID,
					err,
				),
				map[string]any{
					"draft_created": true,
					"media_id":      result.MediaID,
					"output_saved":  false,
					"output_file":   imagePostOutput,
				},
				[]string{
					"Do not retry create_image_post automatically; inspect the WeChat draft first.",
				},
			)
		}
	}

	return result, nil
}

// isTerminal 检查 stdin 是否是终端
func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return true
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func init() {
	createImagePostCmd.Flags().StringVarP(&imagePostTitle, "title", "t", "", "Post title (required)")
	createImagePostCmd.Flags().StringVarP(&imagePostContent, "content", "c", "", "Post description text")
	createImagePostCmd.Flags().StringVar(&imagePostImages, "images", "", "Image paths, comma-separated")
	createImagePostCmd.Flags().StringVarP(&imagePostFromMD, "from-markdown", "m", "", "Extract images from Markdown file")
	createImagePostCmd.Flags().BoolVar(&imagePostOpenComment, "open-comment", false, "Enable comments")
	createImagePostCmd.Flags().BoolVar(&imagePostFansOnly, "fans-only", false, "Only fans can comment")
	createImagePostCmd.Flags().BoolVar(&imagePostDryRun, "dry-run", false, "Preview mode without creating draft")
	createImagePostCmd.Flags().StringVarP(&imagePostOutput, "output", "o", "", "Save result to JSON file")
}

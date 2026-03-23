package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/geekjourneyx/md2wechat-skill/internal/draft"
	"github.com/geekjourneyx/md2wechat-skill/internal/publish"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	draftExecutionStageRead   = "read"
	draftExecutionStageCover  = "cover"
	draftExecutionStageCreate = "create"
)

type draftExecutionInput struct {
	HTMLFile   string
	CoverImage string
	Title      string
	Digest     string
}

type draftExecutionError struct {
	Stage string
	Err   error
}

func (e *draftExecutionError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *draftExecutionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

var (
	createDraftFile  string
	createDraftCover string
	createDraftTitle string
	createDraftDesc  string
)

var createDraftCmd = &cobra.Command{
	Use:   "create_draft --file article.html --cover cover.jpg --title \"Article Title\" [--desc \"Article summary\"]",
	Short: "Create WeChat draft article from HTML file",
	Args:  cobra.NoArgs,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := runCreateDraft()
		if err != nil {
			return err
		}
		responseSuccess(result)
		return nil
	},
}

func init() {
	createDraftCmd.Flags().StringVarP(&createDraftFile, "file", "f", "", "HTML file path")
	createDraftCmd.Flags().StringVarP(&createDraftCover, "cover", "c", "", "Cover image path")
	createDraftCmd.Flags().StringVarP(&createDraftTitle, "title", "t", "", "Article title")
	createDraftCmd.Flags().StringVarP(&createDraftDesc, "desc", "d", "", "Article summary; auto-generated from HTML when omitted")
}

func runCreateDraft() (*publish.DraftResult, error) {
	if err := validateCreateDraftInput(); err != nil {
		return nil, err
	}

	result, err := executeDraftCreation(draftExecutionInput{
		HTMLFile:   createDraftFile,
		CoverImage: createDraftCover,
		Title:      createDraftTitle,
		Digest:     createDraftDesc,
	})
	if err != nil {
		if _, ok := extractCLIError(err); ok {
			return nil, err
		}
		return nil, wrapDraftExecutionError(err, codeDraftCreateFailed, codeDraftCreateFailed, codeDraftCreateFailed)
	}

	return result, nil
}

func validateCreateDraftInput() error {
	if strings.TrimSpace(createDraftFile) == "" {
		return newCLIError(codeDraftCreateInvalid, "--file is required")
	}
	if strings.TrimSpace(createDraftCover) == "" {
		return newCLIError(codeDraftCreateInvalid, "--cover is required")
	}
	if strings.TrimSpace(createDraftTitle) == "" {
		return newCLIError(codeDraftCreateInvalid, "--title is required")
	}
	return nil
}

func executeDraftCreation(input draftExecutionInput) (*publish.DraftResult, error) {
	if err := cfg.ValidateForWeChat(); err != nil {
		return nil, wrapCLIError(codeConfigInvalid, err, err.Error())
	}

	html, err := os.ReadFile(input.HTMLFile)
	if err != nil {
		return nil, &draftExecutionError{
			Stage: draftExecutionStageRead,
			Err:   fmt.Errorf("read HTML file: %w", err),
		}
	}

	title := strings.TrimSpace(input.Title)
	digest := strings.TrimSpace(input.Digest)
	if digest == "" {
		digest = draft.GenerateDigestFromContent(string(html), 120)
	}

	log.Info("creating draft from HTML",
		zap.Int("html_length", len(html)),
		zap.String("cover", input.CoverImage),
		zap.String("title", title))

	log.Info("uploading cover image", zap.String("path", input.CoverImage))
	coverMediaID, err := uploadCoverImageFn(input.CoverImage)
	if err != nil {
		return nil, &draftExecutionError{
			Stage: draftExecutionStageCover,
			Err:   fmt.Errorf("upload cover: %w", err),
		}
	}
	log.Info("cover uploaded", zap.String("media_id", maskMediaID(coverMediaID)))

	svc := newDraftCreator()
	result, err := svc.CreateDraft(publish.Artifact{
		HTML: string(html),
		Metadata: publish.Metadata{
			Title:  title,
			Digest: digest,
		},
		CoverMediaID: coverMediaID,
	})
	if err != nil {
		return nil, &draftExecutionError{
			Stage: draftExecutionStageCreate,
			Err:   fmt.Errorf("create draft: %w", err),
		}
	}

	return result, nil
}

func wrapDraftExecutionError(err error, readCode, coverCode, createCode string) error {
	var execErr *draftExecutionError
	if !errors.As(err, &execErr) {
		return wrapCLIError(createCode, err, err.Error())
	}

	code := createCode
	switch execErr.Stage {
	case draftExecutionStageRead:
		code = readCode
	case draftExecutionStageCover:
		code = coverCode
	}

	return wrapCLIError(code, execErr.Err, execErr.Err.Error())
}

func buildDraftResponse(result *publish.DraftResult) map[string]any {
	response := map[string]any{
		"media_id": result.MediaID,
		"message":  "Draft created successfully! You can check it in WeChat backend.",
	}
	if result.DraftURL != "" {
		response["draft_url"] = result.DraftURL
	}
	return response
}

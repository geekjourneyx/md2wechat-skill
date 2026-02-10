# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.9.0] - 2025-02-06

### Added
- **Image Post (小绿书/Newspic)**: Create WeChat image-only posts with up to 20 images
  - New `create_image_post` command for 小绿书 creation
  - Support comma-separated image paths: `--images photo1.jpg,photo2.jpg`
  - Extract images from Markdown files: `--from-markdown article.md`
  - Comment settings: `--open-comment`, `--fans-only`
  - Preview mode: `--dry-run` for testing without upload
  - Stdin support for description content
- **WeChat API Documentation**: Updated `references/wechat-api.md` with complete draft/add API documentation
  - Added `article_type` field (`news`/`newspic`)
  - Added comment settings fields
  - Added image_info structure for newspic
  - Added cover cropping fields
- **v2 API Themes (38 total)**: Massive theme expansion with new series system
  - **Minimal Series (8)**: minimal-gold, minimal-green, minimal-blue, minimal-orange, minimal-red, minimal-navy, minimal-gray, minimal-sky
  - **Focus Series (8)**: focus-gold, focus-green, focus-blue, focus-orange, focus-red, focus-navy, focus-gray, focus-sky
  - **Elegant Series (8)**: elegant-gold, elegant-green, elegant-blue, elegant-orange, elegant-red, elegant-navy, elegant-gray, elegant-sky
  - **Bold Series (8)**: bold-gold, bold-green, bold-blue, bold-orange, bold-red, bold-navy, bold-gray, bold-sky
  - Theme version tracking: v1.0 (6 basic) vs v2.0 (32 new)
  - Theme preview gallery: https://md2wechat.app/theme-gallery
- **Background Type Control**: New `--background-type` parameter for API mode
  - Options: `default` (default background), `grid` (grid texture), `none` (transparent)
  - Configuration file support: `api.background_type`
  - Environment variable: `DEFAULT_BACKGROUND_TYPE`
- **Google Gemini Image Provider**: Native Google SDK integration
  - Official `google.golang.org/genai` SDK
  - Direct API calls without third-party wrappers
  - Supports official Gemini image models: gemini-3-pro-image-preview, gemini-2.0-flash-exp-image-generation
  - Full aspect ratio support with official size mappings
- **OpenRouter Image Provider**: Multi-model provider access
  - Support for various AI models through OpenRouter API
  - OpenRouter format with image_size and aspect_ratio parameters
  - Configuration: `image_provider: openrouter`

### Changed
- **Article Structure**: Extended with newspic support fields (backward compatible with `omitempty`)
  - `article_type`: news (default) or newspic
  - `need_open_comment`: comment settings
  - `only_fans_can_comment`: fan-only comment settings
  - `image_info`: image list for newspic
- **API Base URL Configuration**: New `md2wechat_base_url` parameter
  - Support for v2 API endpoint: https://md2wechat.app (internal testing)
  - Configuration file: `api.md2wechat_base_url`
  - Environment variable: `MD2WECHAT_BASE_URL`
- **Theme File Structure**: Reorganized with version tracking
  - v1.0 themes: 6 basic themes (default, bytedance, apple, sports, chinese, cyber)
  - v2.0 themes: 32 new series themes (minimal/focus/elegant/bold series)
  - New `themes/api.yaml`: Complete theme catalog for reference

### Technical Details
- **New Files**:
  - `cmd/md2wechat/create_image_post.go` - CLI command implementation
  - `internal/image/gemini.go` - Google Gemini provider with official SDK
  - `internal/image/gemini_test.go` - Gemini provider tests
  - `internal/image/openrouter.go` - OpenRouter provider implementation
  - `internal/image/openrouter_test.go` - OpenRouter provider tests
  - `themes/api.yaml` - Complete v2 theme catalog
  - `themes/elegant-gold.yaml`, `themes/focus-green.yaml`, `themes/minimal-blue.yaml`, `themes/bold-red.yaml` - Series examples
- **Modified Files**:
  - `internal/draft/service.go` - Added `CreateImagePost`, `GetImagePostPreview`, `extractImagesFromMarkdown`
  - `internal/wechat/service.go` - Added `CreateNewspicDraft` for direct API call
  - `cmd/md2wechat/main.go` - Registered `createImagePostCmd`
  - `cmd/md2wechat/convert.go` - Added background_type flag, updated help text
  - `internal/config/config.go` - Added DefaultBackgroundType, MD2WechatBaseURL fields
  - `internal/converter/api.go` - Added BackgroundType to APIRequest, added base URL support
  - `internal/converter/converter.go` - Added BackgroundType to ConvertRequest
  - `internal/image/provider.go` - Added gemini, openrouter providers
  - `skills/md2wechat/references/themes.md` - Updated with v2 themes and background type
  - `themes/*.yaml` - Reorganized with version markers (v1.0 vs v2.0)
  - `docs/CONFIG.md` - Added md2wechat_base_url, background_type documentation
  - `docs/IMAGE_PROVISIONERS.md` - Added Gemini and OpenRouter provider guides
  - `README.md` - Updated with v2 API themes, background type, upgraded API announcement

### Migration Guide
No migration required. All new features are additive:
- `create_image_post` is a new command
- New v2.0 themes are opt-in via `--theme` parameter
- `background_type` defaults to "default" (existing behavior)
- Image providers are configurable, no breaking changes

### Technical Details
- **New Files**:
  - `cmd/md2wechat/create_image_post.go` - CLI command implementation
- **Modified Files**:
  - `internal/draft/service.go` - Added `CreateImagePost`, `GetImagePostPreview`, `extractImagesFromMarkdown`
  - `internal/wechat/service.go` - Added `CreateNewspicDraft` for direct API call (SDK doesn't support newspic)
  - `cmd/md2wechat/main.go` - Registered `createImagePostCmd`
  - `skills/md2wechat/SKILL.md` - Added image post documentation
  - `skills/md2wechat/references/wechat-api.md` - Complete draft API documentation

### Migration Guide
No migration required. The `create_image_post` command is a new feature and doesn't affect existing functionality.

---

## [1.8.0] - 2025-02-05

### Added
- **OpenClaw Platform Support**: Full compatibility with OpenClaw AI Agent platform
  - ClawHub installation: `clawhub install md2wechat`
  - One-click install script: `scripts/install-openclaw.sh`
  - OpenClaw configuration guide: `docs/OPENCLAW.md`
  - SKILL.md metadata for OpenClaw compatibility
- **OpenClaw Badge**: Added to README.md header

### Changed
- **Directory Structure Simplification**:
  - Renamed `skill/` to `skills/` (unified naming)
  - Removed redundant `plugins/` directory
  - Removed `plugin.json` (keeping only `marketplace.json`)
  - Removed `manifest.json` (version hardcoded in `run.sh`)
- **Binary Provisioner Refactoring** (`run.sh`):
  - Reduced from 375 lines to 154 lines (-59%)
  - Changed cache directory from `~/.cache/claude/` to `~/.cache/md2wechat/`
  - Simplified version management (plain text file instead of JSON)
  - Removed invalid jsDelivr mirror (binaries are in GitHub Releases, not in repo)
  - Improved error messages with cleaner formatting
- **Documentation Updates**:
  - Updated README.md with OpenClaw support section
  - Fixed project structure diagram
  - Added OpenClaw vs Claude Code comparison table

### Removed
- `plugins/` directory (redundant copy of skills)
- `.claude-plugin/plugin.json` (replaced by marketplace.json)
- `skill/md2wechat/manifest.json` (version now in run.sh)
- `scripts/sync.sh` (no longer needed without plugins/)

### Technical Details
- **New Files**:
  - `docs/OPENCLAW.md` - OpenClaw installation guide
  - `scripts/install-openclaw.sh` - OpenClaw installer script
- **Modified Files**:
  - `.claude-plugin/marketplace.json` - source changed to `"."`
  - `skills/md2wechat/SKILL.md` - added `metadata.openclaw` for compatibility
  - `skills/md2wechat/scripts/run.sh` - complete refactoring

### Migration Guide
No migration required for existing Claude Code users. The skill continues to work the same way.

For OpenClaw users:
1. Install via ClawHub: `clawhub install md2wechat`
2. Configure in `~/.openclaw/openclaw.json`
3. See `docs/OPENCLAW.md` for detailed instructions

---

## [1.7.0] - 2025-01-25

### Added
- **ModelScope Image Provider**: Native integration with Alibaba ModelScope image generation service
  - Async API support with task-based polling mechanism (task_id + status check)
  - Configurable polling interval (default 5s) and max polling time (default 120s)
  - Model: `Tongyi-MAI/Z-Image-Turbo` with support for custom image sizes
  - Free tier available for testing
- **Write Command Stdin Support**: Pipe input support for non-interactive usage
  - `echo "content" | md2wechat write --style dan-koe`
  - Heredoc support for multi-line content input
- **Image Size Parameter**: ModelScope provider supports separate `width` and `height` parameters
  - Default size: 1024x1024
  - Configurable via `IMAGE_SIZE` or `image_size` in config
- **Documentation Updates**:
  - ModelScope configuration guide in README.md
  - New FAQ entries for content size limit, ModelScope setup, and stdin usage
  - Updated `docs/IMAGE_PROVISIONERS.md` with ModelScope documentation

### Changed
- `internal/image/provider.go`: Added ModelScope (`modelscope`, `ms`) provider to factory
- `internal/image/provider.go`: Updated error hints to include ModelScope in supported providers list
- `cmd/md2wechat/write.go`: Added `readStdin()` function for pipe/redirection detection
- `cmd/md2wechat/write.go`: Added `io` import for stdin reading functionality
- `internal/wechat/service.go`: Fixed `maskMediaID()` to handle empty strings safely
- `skills/md2wechat/SKILL.md`: Added stdin/heredoc usage examples
- `skills/md2wechat/references/writing-guide.md`: Added non-interactive input methods section

### Technical Details
- **New Files**:
  - `internal/image/modelscope.go` - ModelScope provider implementation (372 lines)
    - `ModelScopeProvider` struct with async polling support
    - `parseSize()` function for WIDTHxHEIGHT format parsing
    - `createTask()` for async job creation
    - `pollTaskStatus()` for status polling with timeout
    - `getTaskStatus()` for task status retrieval
    - `handleErrorResponse()` for comprehensive error handling
  - `internal/image/modelscope_test.go` - Unit tests (9 test cases, all passing)

### Configuration
```yaml
# ModelScope Configuration Example
api:
  image_provider: modelscope
  image_key: ms-your-token-here
  image_base_url: https://api-inference.modelscope.cn
  image_model: Tongyi-MAI/Z-Image-Turbo
  image_size: 1024x1024
```

### Migration Guide
No migration required. ModelScope is a new optional image provider. To use it:

1. Get API Key from [modelscope.cn](https://modelscope.cn/my/myaccesstoken)
2. Set `IMAGE_PROVIDER=modelscope`
3. Set `IMAGE_API_KEY=ms-your-token-here`
4. Run: `md2wechat generate_image "A golden cat"`

---

## [1.6.0] - 2025-01-19

### Added
- **AI Writing Trace Removal (Humanizer)**: Remove AI-generated text patterns to make content sound more natural
  - New `humanize` command for standalone AI trace removal
  - Support for 24 AI writing pattern types based on humanizer-zh method
  - Three intensity levels: `gentle`, `medium` (default), `aggressive`
  - Quality scoring system with 5 dimensions (directness, rhythm, trust, authenticity, conciseness)
  - Style priority principle: preserves writing style characteristics when combined with creator styles
- **Write + Humanize Integration**: Combine writing style generation with AI trace removal
  - New `--humanize` flag for `write` command
  - New `--humanize-intensity` flag to control processing intensity
  - `--show-changes` flag to view modification details and quality scores
  - `-o, --output` flag for `humanize` command to save results to file
- **Pattern Detection Categories**:
  - Content patterns: overemphasis, promotional language, vague attribution
  - Language patterns: AI vocabulary, negative parallelism, three-part formula
  - Style patterns: dash overuse, bold abuse, emoji decoration
  - Filler avoidance: padding phrases, over-qualification, generic conclusions
  - Collaboration traces: conversational filler, knowledge cutoff disclaimers
- **Documentation**:
  - New `references/humanizer.md` with complete AI trace removal guide
  - Updated SKILL.md with humanizer natural language examples
  - Updated manifest.json version to 1.6.0

### Changed
- `cmd/md2wechat/main.go`: Added `humanizeCmd` to root command registration
- `cmd/md2wechat/humanize.go`: Implemented complete humanize command with flags
- `cmd/md2wechat/write.go`: Added humanizer integration flags and output handling

### Technical Details
- **New Files**:
  - `internal/humanizer/` - AI writing trace removal module
    - `result.go` - Core data structures (HumanizeRequest, HumanizeResult, Score, Change)
    - `prompt.go` - Humanizer-zh prompt builder with 24 pattern types
    - `humanizer.go` - Core processing logic and response parsing
  - `cmd/md2wechat/humanize.go` - Humanize command implementation
  - `skills/md2wechat/references/humanizer.md` - Humanizer documentation

### Breaking Changes
- None

### Migration Guide
No migration required. The humanize command is a new feature and doesn't affect existing functionality.

---

## [1.5.0] - 2025-01-17

### Added
- **Writer Style Assistant**: AI-powered writing assistance with customizable creator styles
  - New `write` command for assisted article generation
  - Support for multiple input types: idea, fragment, outline, title
  - Automatic cover prompt generation matching writing style
- **Dan Koe Style**: First built-in creator writing style
  - Profound, sharp, grounded tone for personal growth and opinion pieces
  - Complete writing framework with hooks, structures, and quote extraction
  - Victorian Woodcut/Etching style cover generation
- **Custom Style System**: YAML-based style definitions in `writers/` directory
  - Easy to add custom creator styles
  - Configurable writing prompts and cover styles
- **Image Size Control**: New `--size` parameter for `generate_image` command
  - Support for 16:9 ratio (2560x1440) for WeChat cover images
  - Multiple preset sizes: 2048x2048, 1920x1920, 2560x1440, 1440x2560, etc.
- **Documentation**:
  - New `writers/README.md` with custom style guide
  - New `docs/WRITING_FAQ.md` for writing beginners
  - New `references/writing-guide.md` with complete write command reference
  - Updated `references/image-syntax.md` with size parameter documentation
  - Enhanced README.md with write command workflows and diagrams
  - Enhanced SKILL.md with writing assistance natural language examples

### Changed
- README: Added API service notice at top with md2wechat.cn contact information
- README: Updated author section with donation information and QR codes
- README: Added writer style comparison table and workflow diagrams
- SKILL.md: Improved LLM instruction following with explicit trigger conditions
- Sync script: Added `writing-guide.md` to file synchronization list

### Technical Details
- **New Files**:
  - `internal/writer/` - Writer style assistant module
    - `types.go` - Core data structures
    - `style.go` - Style management system
    - `generator.go` - Article generation logic
    - `assistant.go` - High-level assistant API
    - `cover_generator.go` - Cover prompt generation
  - `cmd/md2wechat/write.go` - Write command implementation
  - `writers/dan-koe.yaml` - Dan Koe style configuration
  - `writers/README.md` - Custom style guide
  - `docs/WRITING_FAQ.md` - Writing functionality FAQ
  - `skills/md2wechat/references/writing-guide.md` - Writing command reference

### Breaking Changes
- None

### Migration Guide
No migration required. The write command is a new feature and doesn't affect existing functionality.

---

## [1.4.0] - 2025-01-14

### Added
- **Provider Pattern**: Extensible image generation service architecture
- **TuZi Integration**: Support for TuZi (tu-zi.com) image generation service
  - Models: `doubao-seedream-4-5-251128` (default), `gemini-3-pro-image-preview`
  - Sizes: 2048x2048 (default), 1920x1920, 2560x1440, 1440x2560, 3072x2048, 2048x3072, 3840x2160, 2160x3840
- **Natural Language Image Generation**: Generate images via conversational interface
  - Method 1: Insert into article at specific position
  - Method 2: Standalone image generation
  - Method 3: Manual Markdown syntax `![alt](__generate:prompt__)`
- **Configuration Fields**: `image_provider`, `image_api_base`, `image_model`, `image_size`
- **Documentation**: New `docs/IMAGE_PROVISIONERS.md` with complete provider configuration guide
- **Sync Script**: `scripts/sync.sh` to keep `skill/` and `plugins/` directories synchronized
- **Makefile Target**: `make sync` for easy directory synchronization

### Changed
- README: Enhanced platform-specific download instructions (Mac Intel vs Apple Silicon)
- README: Added AI image generation section with natural language examples
- `scripts/install.sh`: Added Linux ARM64 detection and support
- Default TuZi image size: `1024x1024` → `2048x2048` (TuZi requires minimum 3.7M pixels)

### Fixed
- URL download with query parameters: Fixed file name too long error when downloading generated images
- Platform detection in install script for ARM64 systems
- API base URL documentation: Updated to correct `https://api.tu-zi.com/v1`

### Technical Details
- **New Files**:
  - `internal/image/provider.go` - Provider interface and factory
  - `internal/image/openai.go` - OpenAI DALL-E provider
  - `internal/image/tuzi.go` - TuZi image generation provider
  - `docs/IMAGE_PROVISIONERS.md` - Provider configuration guide
  - `scripts/sync.sh` - Directory synchronization script

---

## [1.3.1] - 2025-01-12

### Added
- Auto-download binary from GitHub releases on first run
- User-friendly error messages with actionable guidance
- System dependencies declaration in `manifest.json`
- XDG-compliant cache directory (`~/.cache/claude/`) for binary storage
- Fallback to local `bin/` directory for development/offline usage
- Automatic version checking and update prompt when binary is outdated

### Changed
- Binary naming format now uses hyphens (`md2wechat-linux-amd64`) to match GitHub releases
- Error messages now show human-readable platform names (e.g., "macOS (Apple Silicon)")
- Download progress displays concise status information

### Fixed
- Binary name mismatch between workflow and run.sh that caused download failures

---

## [1.3.0] - 2025-01-11

### Added
- Plugin Marketplace support with `.claude-plugin/marketplace.json`
- One-command Claude Code installation via `/plugin marketplace add`
- Prominent Claude Code installation section at top of README
- Claude Code badge for quick identification
- Detailed binary installation instructions with location guidance
- Installation steps for Windows, Mac, and Linux users

### Changed
- Updated QUICKSTART.md with Claude Code section at the beginning
- Enhanced docs/USAGE.md with Claude Code integration guide
- Improved download table with installation locations
- Added collapsible installation steps for each platform

### Installation
```bash
# Claude Code users (simplest)
/plugin marketplace add geekjourneyx/md2wechat-skill
/plugin install md2wechat@geekjourneyx-md2wechat-skill
```

---

## [1.2.0] - 2025-01-11

### Added
- Claude Code Skill support with `.claude-plugin/plugin.json`
- Claude Code Skill in `skills/md2wechat/` directory for distribution
- New API themes: `bytedance`, `apple`, `sports`, `chinese`, `cyber`
- Comprehensive troubleshooting guide in SKILL.md
- API theme selection section in README.md
- CHANGELOG.md with version history and upgrade guide

### Changed
- Updated themes.md with complete API and AI theme documentation
- Enhanced HTML guide with AI theme specific requirements
- Improved SKILL.md with detailed error handling and FAQ
- Updated command help text to reflect all available themes
- Enhanced FAQ.md with IP whitelist configuration guide

### Removed
- `leo` theme (deprecated)

---

## [1.1.0] - 2025-01-11

### Added
- YAML-based theme system for AI mode
- AI themes: `autumn-warm`, `spring-fresh`, `ocean-calm`
- Custom theme support with `custom.yaml`
- Theme configuration with color schemes and style info
- Reference documentation for themes, HTML guide, image syntax, and WeChat API

### Changed
- Refactored theme system to use YAML files instead of hardcoded prompts
- Updated SKILL.md with new AI theme workflow
- Enhanced themes.md with detailed style specifications

---

## [1.0.1] - 2025-01-11

### Fixed
- Mermaid diagrams rendering for GitHub documentation

---

## [1.0.0] - 2025-01-11

### Added
- Initial release of md2wechat
- API mode conversion using md2wechat.cn API
- AI mode conversion with Claude AI
- WeChat draft upload functionality
- Image upload to WeChat material library
- Configuration management with YAML support
- Command-line interface with cobra
- Multi-platform binary support (Windows, macOS, Linux)
- Comprehensive documentation (README, FAQ, USAGE)
- Test draft command for HTML validation

### Features
- Convert Markdown to WeChat Official Account formatted HTML
- Support for local images, online images, and AI-generated images
- Automatic image compression and optimization
- Draft creation with cover image support
- Environment variable and config file support
- Claude Code Skill integration

---

## Version History Summary

| Version | Date | Description |
|---------|------|-------------|
| [1.9.0] | 2025-02-06 | Image Post (小绿书/newspic) support |
| [1.8.0] | 2025-02-05 | OpenClaw support, directory simplification, run.sh refactoring |
| [1.7.0] | 2025-01-25 | ModelScope image provider, write command stdin support |
| [1.6.0] | 2025-01-19 | AI writing trace removal (Humanizer), write + humanize integration |
| [1.5.0] | 2025-01-17 | Writer style assistant, Dan Koe style, image size control |
| [1.4.0] | 2025-01-14 | TuZi image provider, natural language image generation |
| [1.3.1] | 2025-01-12 | Auto binary download, user-friendly errors, system dependencies |
| [1.3.0] | 2025-01-11 | Plugin Marketplace support, enhanced installation docs |
| [1.2.0] | 2025-01-11 | Claude Code plugin support, new API themes |
| [1.1.0] | 2025-01-11 | YAML theme system, AI themes (autumn-warm, spring-fresh, ocean-calm) |
| [1.0.1] | 2025-01-11 | Fixed Mermaid diagrams for GitHub rendering |
| [1.0.0] | 2025-01-11 | Initial release with full md2wechat functionality |

---

## Upgrade Guide

### From v1.1.0 to Unreleased

**New API Themes Available:**
```bash
# New themes available
md2wechat convert article.md --theme bytedance
md2wechat convert article.md --theme apple
md2wechat convert article.md --theme sports
md2wechat convert article.md --theme chinese
md2wechat convert article.md --theme cyber
```

**Claude Code Skill Integration:**
```bash
# Install as Claude Code Skill
cp -r skills/md2wechat ~/.claude/skills/
```

### From v1.0.0 to v1.1.0

**Theme System Migration:**
- Old theme names have been updated
- AI themes now use YAML configuration files
- Update your commands to use new theme names:
  - `autumn-warm` instead of `elegant`
  - `spring-fresh` instead of `minimal`
  - `ocean-calm` instead of `tech`

---

## Links

- [GitHub Repository](https://github.com/geekjourneyx/md2wechat-skill)
- [Documentation](README.md)
- [Issues](https://github.com/geekjourneyx/md2wechat-skill/issues)

---
name: md2wechat
description: Convert Markdown to WeChat Official Account HTML. Use this whenever the user wants WeChat article conversion, draft upload, image generation for articles, cover or infographic generation, image-post creation, writer-style drafting, AI trace removal, or needs to inspect supported providers, themes, and prompt templates before running the workflow.
---

# MD to WeChat

Converts Markdown articles to WeChat Official Account formatted HTML with inline CSS and optionally uploads to draft box. Supported natively via AI mode:

- **AI Mode**: Generate themed AI requests / prompts for external models such as Claude Code

## Quick Start

```bash
# Generate AI request / prompt (AI mode)
md2wechat convert article.md --mode ai --theme autumn-warm --preview

# Upload to WeChat draft box
md2wechat convert article.md --draft --cover cover.jpg

# Create a draft directly from existing HTML
md2wechat create_draft --file article.html --cover cover.jpg --title "Article Title" --desc "Article summary"
```

### Config Location

Before asking the user to edit configuration, check these in order:

1. `~/.config/md2wechat/config.yaml` (default and recommended)
2. Environment variables such as `MD2WECHAT_BASE_URL`
3. Project-local `md2wechat.yaml` / `md2wechat.yml` / `md2wechat.json`

Default conversion mode:

- If the user does not explicitly pass `--mode`, `convert` should be treated as `api`
- Only use AI mode when the user explicitly asks for `--mode ai` or clearly requests AI themed conversion

Built-in assets:

- Default themes and the default writer style are bundled with the binary
- Prefer `MD2WECHAT_THEMES_DIR` / `MD2WECHAT_WRITERS_DIR` for explicit overrides
- Then check the project-local `themes/` / `writers/` directories
- Then check `~/.config/md2wechat/themes/` / `~/.config/md2wechat/writers/`
- Do not assume the repository directory exists on the agent host

Prompt catalog:

- Before guessing supported providers, themes, or prompt templates, inspect the CLI first
- Use `capabilities --json` as the first discovery step
- Use `providers list --json`, `themes list --json`, and `prompts list --json` before picking resources
- Prompt overrides use: `MD2WECHAT_PROMPTS_DIR` → project-local `prompts/` → `~/.config/md2wechat/prompts/` → bundled prompts
- Do not assume prompt YAML files exist on disk outside these locations
- Treat CLI discovery output as the source of truth; use repository references only as workflow help or style examples

Recommended discovery flow:

```bash
md2wechat capabilities --json
md2wechat providers list --json
md2wechat themes list --json
md2wechat prompts list --json
md2wechat prompts list --kind image --archetype cover --json
```

For image presets, do not assume `archetype` is the only truth. Some infographic presets can also be used as covers. Inspect `primary_use_case`, `compatible_use_cases`, `recommended_aspect_ratios`, and `default_aspect_ratio` from `prompts show --json` before choosing a preset.

When a task depends on a specific template, inspect it first:

```bash
md2wechat prompts show cover-default --kind image --json
md2wechat prompts show cover-hero --kind image --archetype cover --tag hero --json
md2wechat prompts show infographic-dark-ticket-cn --kind image --archetype infographic --tag ticket --json
md2wechat prompts show infographic-handdrawn-sketchnote --kind image --archetype infographic --tag sketchnote --json
md2wechat prompts show infographic-apple-keynote-premium --kind image --archetype infographic --tag apple --json
md2wechat prompts show infographic-victorian-engraving-banner --kind image --archetype infographic --tag victorian --json
md2wechat prompts render cover-default --kind image --var article_title='Example' --json
md2wechat generate_cover --article article.md
md2wechat generate_infographic --article article.md --preset infographic-comparison
md2wechat generate_infographic --article article.md --preset infographic-dark-ticket-cn --aspect 21:9
md2wechat generate_infographic --article article.md --preset infographic-handdrawn-sketchnote
md2wechat generate_infographic --article article.md --preset infographic-apple-keynote-premium
md2wechat generate_infographic --article article.md --preset infographic-victorian-engraving-banner --aspect 21:9
md2wechat generate_image --preset cover-hero --article article.md --model gemini-3-pro-image-preview
```

### Natural Language Image Generation

You can also ask me to generate images using natural language:

#### Generate Image for Article (Insert into Markdown)

```
"Help me generate a product concept image at the beginning of article.md"
"Add an image showing the product features after the second paragraph"
"Create a diagram for the comparison section in article.md"
```

I will:
1. Read the article to understand the context
2. Insert the AI image generation syntax at the appropriate location
3. Call the conversion command to generate and upload the image

#### Generate Standalone Image (Not for Article)

```
"Generate an image of a cute cat sitting on a windowsill"
"Create a product concept image: modern smart home device, white design"
"Make a diagram showing the user flow"
```

I will:
1. Call the image generation command directly
2. Return the generated image URL and WeChat material ID

### Natural Language Writing Assistance

You can also ask me to help write articles using creator styles:

#### Write Article from Idea

```
"Write an article about self-discipline using Dan Koe style"
"Help me write a post about productivity with a sharp, grounded tone"
"Create a story-style article about my travel experience"
```

I will:
1. Understand your idea or topic
2. Use the appropriate writing style (default: Dan Koe)
3. Generate article structure and content
4. Extract memorable quotes
5. Optionally generate matching cover image

#### Refine Existing Content

```
"Rewrite this article with a more engaging style"
"Polish my article.md with Dan Koe's writing style"
"Make this content more profound and sharp"
```

I will:
1. Read your existing content
2. Apply the selected writing style
3. Maintain original meaning while improving expression

#### Generate Cover Only

```
"Generate a cover image for my article about self-discipline"
"Create a Victorian woodcut style cover for my philosophy piece"
```

#### AI Writing Trace Removal (Humanizer)

```
"Remove AI traces from this article: article.md"
"Humanize this text to make it sound more natural"
"Remove AI writing traces with gentle intensity"
"Rewrite this to sound less like AI generated"
```

I will:
1. Read the text to identify AI writing patterns (24 types)
2. Remove or rewrite problematic phrases
3. Inject natural human-like expressions
4. Preserve core meaning and tone
5. Return quality score (5 dimensions, /50)

**Humanizer can be combined with writing styles:**
```
"Write with Dan Koe style and remove AI traces"
"Use dan-koe style, then humanize the result"
```

#### List Available Styles

```
"Show me all available writing styles"
"What writing styles can I use?"
```

**Available Writing Styles:**
- **Dan Koe** (default): Profound, sharp, grounded - great for personal growth and opinion pieces

Users can add custom styles in `writers/` directory. See `writers/README.md` for details.

### Discovery-first Rule

When working as an Agent:

1. Check `capabilities --json` before assuming a feature exists
2. Check `providers list --json` before selecting an image provider
3. Check `themes list --json` before selecting a theme
4. Check `prompts list --json` before selecting or rendering a prompt template
5. Only fall back to repository references when the CLI discovery output is insufficient

## Workflow Checklist

Copy this checklist to track progress:

```
Progress:
- [ ] Step 1: Analyze Markdown structure and images
- [ ] Step 2: Confirm conversion mode (API/AI) and theme
- [ ] Step 3: Generate HTML with inline CSS
- [ ] Step 4: Process images (upload to WeChat)
- [ ] Step 5: Replace image URLs in HTML
- [ ] Step 6: Preview or upload to draft
```

---

## Step 1: Analyze Markdown

Read the markdown file and extract:

| Element | How to Extract |
|---------|----------------|
| **Title** | First `# heading` or filename |
| **Author** | Look for `Author:` or `作者:` in frontmatter |
| **Digest** | First paragraph or generate from content (max 120 chars) |
| **Images** | Collect all `![alt](src)` references |
| **Structure** | Headings, lists, code blocks, quotes, tables |

**Image Reference Types**:

| Type | Syntax | Processing |
|------|--------|------------|
| Local | `![alt](./path/image.png)` | Upload to WeChat |
| Online | `![alt](https://example.com/image.png)` | Download then upload |
| AI Generate | `![alt](__generate:prompt__)` | Generate via AI then upload |

---

## Step 2: Confirm Mode and Theme

### Conversion Mode

| Mode | Speed | Style | Best For |
|------|-------|-------|----------|
| **API** | Fast (seconds) | Clean, standard | Quick publishing, technical content |
| **AI** | Slower (10-30s) | Beautiful themed | Important articles, brand content |

Before picking a theme, inspect the live CLI result:

```bash
md2wechat themes list --json
```

### AI Themes

These are common built-in examples, not the source of truth:

| Theme | Description | Best For |
|-------|-------------|----------|
| **autumn-warm** | Warm orange tones, emotional, literary | Stories, lifestyle, essays |
| **spring-fresh** | Fresh green tones, natural, vibrant | Travel, nature, outdoor |
| **ocean-calm** | Calm blue tones, professional, rational | Tech articles, business analysis |
| **custom** | Use custom prompt | Brand customization |

### API Themes (Fast)

These are representative examples. The authoritative list is `themes list --json`.

| Theme | Description | Best For |
|-------|-------------|----------|
| **default** | Default theme, clean and professional | General content |
| **bytedance** | ByteDance style | Tech news |
| **apple** | Apple minimalist style | Product reviews |
| **sports** | Active sports style | Sports content |
| **chinese** | Traditional Chinese culture style | Cultural articles |
| **cyber** | Cyberpunk style | Frontier tech |

**Ask the user**: "Which mode and theme would you like?" - Only ask if the user doesn't specify in their request.

- **AI mode** (themed): autumn-warm, spring-fresh, ocean-calm

**Default**: ai mode.

Read [references/themes.md](references/themes.md) for visual intent and prompt examples only. Do not treat it as the authoritative theme inventory.

---

## Step 3: Generate HTML



### AI Mode

Read the selected style prompt from `references/themes.md` and generate an AI request / prompt for an external model to continue into WeChat-safe HTML with **inline CSS**.

**Important Rules**:

1. All CSS must be **inline** (in `style` attributes)
2. No external stylesheets or scripts
3. Use WeChat-safe HTML tags only
4. Image placeholder format: `<!-- IMG:0 -->`, `<!-- IMG:1 -->`, etc.

**Safe HTML Tags**:
- `<p>`, `<br>`, `<strong>`, `<em>`, `<u>`, `<a>`
- `<h1>` to `<h6>`
- `<ul>`, `<ol>`, `<li>`
- `<blockquote>`, `<pre>`, `<code>`
- `<table>`, `<thead>`, `<tbody>`, `<tr>`, `<th>`, `<td>`
- `<section>`, `<span>` (with inline styles)

**Avoid**:
- `<script>`, `<iframe>`, `<form>`
- External CSS/JS references
- Complex positioning (fixed, absolute)

**Critical for WeChat**:
- Create a main `<div>` container immediately after `<body>` to hold all global styles
- Specify `color` explicitly for each `<p>` tag (WeChat resets to black otherwise)
- Use two `<span>` tags for heading symbols: one with color+text-shadow, one with solid color

---

## Step 4: Process Images

### Image Generation Methods

There are **three ways** to generate AI images:

#### Method 1: Natural Language - For Article (Recommended)

Simply describe what you want in plain language:

```
User: "Generate a product concept image at the beginning of article.md"

User: "Add a comparison chart after the third paragraph"

User: "Create an image showing the workflow diagram in article.md"
```

**How I process natural language requests:**

1. **Understand the intent** - Identify where to insert the image
2. **Read the article** - Analyze context to create an appropriate prompt
3. **Insert the syntax** - Add `![alt](__generate:prompt__)` at the correct location
4. **Show the prompt** - Display the generated prompt for transparency
5. **Generate and upload** - Call the conversion command to complete

**Note**: Proceed directly with generation. Only ask for confirmation if the prompt is complex or ambiguous.

**Example conversation:**

```
User: "Add a product image at the start of my article"
Claude: "I'll add a product concept image at the beginning of article.md.
Based on your article about 'Smart Home Hub', I'll use this prompt:
'A modern smart home hub device, sleek white design with LED indicator
lights, minimalist product photography on a clean white background'
I'll proceed with generating the image."
```

#### Method 2: Natural Language - Standalone Image

Generate an image without any article:

```
User: "Generate an image of a cute cat sitting on a windowsill"
User: "Create a product concept: modern smart home device"
User: "Make a diagram showing user signup flow"
```

**I will:**
1. Create an appropriate prompt based on your description
2. Prefer a bundled preset when the image is a cover or infographic
3. Otherwise call: `md2wechat generate_image "prompt"`
3. Return the WeChat URL and media ID

**Use when:** You just need an image, not for any article.

#### Method 3: Manual Syntax

Write the image generation syntax directly in Markdown:

```markdown
![Product Concept](__generate:A futuristic smart home hub device, sleek design__)
```

**Syntax format:** `![alt text](__generate:prompt__)`

---

### Processing Images by Type

For each image reference in order:

#### Local Image

```bash
md2wechat upload_image "/path/to/image.png"
```

Response:
```json
{"success": true, "wechat_url": "https://mmbiz.qpic.cn/...", "media_id": "xxx"}
```

#### Online Image

```bash
md2wechat download_and_upload "https://example.com/image.png"
```

#### AI Generated Image (via CLI)

```bash
# Generate with default size (2048x2048 square)
md2wechat generate_image "A cute cat sitting on a windowsill"

# Generate a cover image from bundled prompt presets
md2wechat generate_cover --article article.md

# Generate an infographic from bundled prompt presets
md2wechat generate_infographic --article article.md --preset infographic-process

# Generate with 16:9 ratio for WeChat cover (recommended)
md2wechat generate_image --preset cover-hero --article article.md --size 2560x1440
```

**WeChat Cover Images**: For article covers, use 16:9 horizontal ratio (2560x1440 recommended) as it displays better in WeChat's feed and article list. Square images (2048x2048) are cropped in preview.

**Note**: AI image generation requires `IMAGE_API_KEY` environment variable.

**Image Processing Pipeline**:
1. If AI generation: Call image API → get URL
2. If online: Download image to temp
3. If local: Read file
4. Compress if width > 1920px (configurable)
5. Upload to WeChat material API
6. Return `wechat_url` and `media_id`
7. Store result for HTML replacement

---

## Step 5: Replace Image URLs

Replace placeholders in HTML:

```html
<!-- Before -->
<!-- IMG:0 -->
<!-- IMG:1 -->

<!-- After -->
<img src="https://mmbiz.qpic.cn/..." />
<img src="https://mmbiz.qpic.cn/..." />
```

Use the WeChat URLs returned from image processing.

---

## Step 6: Preview or Upload

Ask user:

1. **Preview only** - Show HTML for review
2. **Upload to draft** - Create WeChat draft article

### Preview Mode

Display HTML in markdown code block for user to copy.

### Upload Mode

Create draft and run:

```bash
md2wechat create_draft --file article.html --cover cover.jpg --title "Article Title" --desc "Article summary"
```

**Required for draft**:
- `WECHAT_APPID` environment variable
- `WECHAT_SECRET` environment variable
- Cover image (use `--cover` or first image in content)

Response:
```json
{"success": true, "media_id": "draft_media_id", "draft_url": "https://mp.weixin.qq.com/..."}
```

---

## Configuration

### Required for WeChat API

| Variable | Description | Required |
|----------|-------------|----------|
| `WECHAT_APPID` | WeChat Official Account AppID | Yes, for draft upload |
| `WECHAT_SECRET` | WeChat API Secret | Yes, for draft upload |

### Optional for AI Features

| Variable | Description | Required |
|----------|-------------|----------|
| `IMAGE_API_KEY` | Image generation API key | For AI images |
| `IMAGE_API_BASE` | Image API base URL | For AI images |
| `COMPRESS_IMAGES` | Compress images > 1920px (true/false) | No, default true |
| `MAX_IMAGE_WIDTH` | Max width in pixels | No, default 1920 |

### How to Get AppID and Secret

1. Visit [WeChat Developer Platform](https://developers.weixin.qq.com/platform)
2. Login and select your Official Account
3. Go to **Settings & Development** → **Basic Configuration**
4. Find in **Developer ID** section:
   - **Developer ID (AppID)**: Copy directly
   - **Developer Password (AppSecret)**: Click "Reset" to get
5. Add these values to environment variables or config file

> **Warning**: AppSecret is very important, keep it secure!

### Config File Location

```
~/.config/md2wechat/config.yaml    # Global config
./md2wechat.yaml                    # Project config (higher priority)
```

---

## Error Handling

| Error | Action |
|-------|--------|
| Missing config | Ask user to set environment variable or run `md2wechat config init` |
| Image upload fails | Log error, continue with placeholder |
| WeChat API fails | Show error message, return HTML for manual upload |
| Markdown parse error | Ask user to check file format |
| IP not in whitelist | Guide user to add IP to WeChat whitelist (see Troubleshooting) |

---

## Complete Examples

### Example 1: Simple Article (No Images)

**Input**: `simple.md`
```markdown
# My First Article

This is a simple article with no images.
```

**Process**:
1. Generate AI prompt
2. Skip image processing
3. Ask: preview or upload?
4. If upload → create draft

### Example 2: Article with Local Images

**Input**: `with-images.md`
```markdown
# Travel Diary

Day 1 in Paris:

![Eiffel Tower](./photos/eiffel.jpg)
```

**Process**:
1. Analyze: 1 local image
2. Generate HTML with `<!-- IMG:0 -->` placeholder
3. Run: `upload_image "./photos/eiffel.jpg"`
4. Replace placeholder with WeChat URL
5. Preview or upload

### Example 3: AI Mode with Theme

**Input**: `story.md`
```markdown
# The Old Library

A story about memories...
```

**Process**:
1. User selects AI mode + autumn-warm theme
2. Read theme prompt from references/themes.md
3. Generate themed AI request / prompt
4. Continue in Claude Code or another compatible AI environment to produce HTML

### Example 4: AI Image Generation via Natural Language

**User Request:**
```
"Help me add a product concept image at the beginning of article.md"
```

**Process:**
1. Read article.md to understand the product
2. Create an appropriate image prompt based on context
3. Confirm with user: "I'll use this prompt: '...'"
4. Insert `![Product Concept](__generate:...)` at line 2
5. Run conversion command to generate and upload

**Result:** Image generated and uploaded to WeChat

---

### Example 5: Article with Pre-written Image Syntax

**Input**: `mixed.md`
```markdown
# Tech Review

![Product Photo](./product.jpg)

![Comparison Chart](https://example.com/chart.png)

![Concept Art](__generate:Futuristic gadget design__)
```

**Process:**
1. Process 3 images in order
2. Each returns WeChat URL
3. Replace all placeholders
4. Final HTML with all WeChat-hosted images

---

## References

- [Style Themes](references/themes.md) - Detailed style prompts for AI themes
- [HTML Guide](references/html-guide.md) - WeChat HTML constraints and best practices
- [Image Syntax](references/image-syntax.md) - Image reference syntax and generation
- [Writing Guide](references/writing-guide.md) - Writer style assistant documentation
- [Humanizer](references/humanizer.md) - AI writing trace removal documentation 🆕
- [WeChat API](references/wechat-api.md) - API reference

---

## Troubleshooting

### Configuration Issues

**Q: "AppID not configured" error**
A: Set `WECHAT_APPID` and `WECHAT_SECRET` environment variables, or run:
```bash
md2wechat config init
```

**Q: Config file not working**
A: Check config file location. Supported locations:
- `./md2wechat.yaml` (current directory, highest priority)
- `~/.md2wechat.yaml`
- `~/.config/md2wechat/config.yaml`

### Image Issues

**Q: Image upload fails with "invalid filetype"**
A: WeChat supports JPG, PNG, GIF. Ensure image is in correct format:
```bash
# Convert using ImageMagick
convert input.tiff output.jpg
```

**Q: Images not showing in draft**
A: Images must use WeChat-hosted URLs (`mmbiz.qpic.cn`), not external URLs.

**Q: AI image generation fails**
A: Check `IMAGE_API_KEY` is set and API base URL is correct.

### WeChat API Issues

**Q: "IP not in whitelist" error**
A: Add your server IP to WeChat whitelist:

1. Get your public IP:
```bash
curl ifconfig.me
# or
curl ip.sb
```

2. Add IP to WeChat:
   - Visit [WeChat Developer Platform](https://developers.weixin.qq.com/platform)
   - Go to **Settings & Development** → **Basic Configuration**
   - Find **IP Whitelist** section
   - Click "Set" and add your IP
   - Wait a few minutes for changes to take effect

**Q: "access_token expired" error**
A: Program auto-refreshes tokens. If persists:
```bash
# Check config
md2wechat config show

# Re-init if needed
md2wechat config init
```

**Q: "create draft failed" error**
A: Possible causes:
1. Insufficient permissions - ensure account is verified
2. Sensitive content - check article content
3. Draft limit reached - check existing drafts
4. **Content size out of limit (errcode=45002)** - article content exceeds WeChat API limit

**Q: "content size out of limit" error (errcode=45002)**
A: WeChat draft API has content limits:
- **Characters**: < 20,000 characters (中文算1个字符)
- **Size**: < 1 MB

If you encounter this error:
1. Shorten your article content
2. Reduce unnecessary formatting (inline CSS adds size)
3. Consider splitting into multiple articles
4. Use simpler themes with less inline styling


**Q: API rate limit exceeded**
A: WeChat has API limits. Wait and retry:
```bash
# Wait 60 seconds
sleep 60
# Retry
md2wechat convert article.md --draft
```

### HTML/Style Issues

**Q: Styles not working in WeChat editor**
A: Check:
1. CSS uses inline `style` attributes (not `<style>` tags)
2. CSS properties are in allowed list (see HTML Guide)
3. No syntax errors (unclosed tags, etc.)

**Q: Background color lost in WeChat**
A: WeChat strips `<body>` styles. Use main container:
```html
<div style="background-color: #faf9f5; padding: 40px 10px;">
  <!-- All content here -->
</div>
```

**Q: Text color not as expected**
A: WeChat resets `<p>` color to black. Always specify:
```html
<p style="color: #4a413d;">Your text here</p>
```

### Command Issues

**Q: "command not found: md2wechat"**
A: Coding-agent skill 默认要求 `md2wechat` 已经安装到 `PATH`，不再自带 runtime wrapper 或执行时下载逻辑。

Install the CLI first, then rerun the skill:
```bash
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.0.3/install.sh | bash
npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
```

If the installer finished but the current shell still cannot find `md2wechat`, run:

```bash
export PATH="$HOME/.local/bin:$PATH"
md2wechat version --json
```

**Q: AI mode very slow**
A: AI mode requires Claude API call and takes 10-30 seconds.

---

## CLI Commands Reference

This skill assumes the `md2wechat` CLI is already installed and available on `PATH`:

```bash
# Show help
md2wechat --help

# Convert and preview
md2wechat convert article.md --preview

# Convert with AI theme
md2wechat convert article.md --mode ai --theme autumn-warm --preview

# Convert and upload to draft
md2wechat convert article.md --draft --cover cover.jpg

# Upload single image
md2wechat upload_image photo.jpg

# Download and upload online image
md2wechat download_and_upload https://example.com/image.jpg

# Generate AI image (requires IMAGE_API_KEY)
md2wechat generate_image "A cute cat sitting on a windowsill"

# Generate cover image from bundled presets
md2wechat generate_cover --article article.md

# Generate infographic image from bundled presets
md2wechat generate_infographic --article article.md --preset infographic-comparison

# Generate with 16:9 ratio for WeChat cover via preset (recommended)
md2wechat generate_image --preset cover-hero --article article.md --size 2560x1440

# Initialize config
md2wechat config init

# Show config
md2wechat config show

# List available writing styles
md2wechat write --list

# Write with creator style (interactive)
md2wechat write

# Write with specific style (via stdin/piped)
echo "你的想法或内容" | md2wechat write --style dan-koe

# Write with title and heredoc
md2wechat write --style dan-koe --title "文章标题" <<EOF
你的内容
EOF

# Write with specific style
md2wechat write --style dan-koe

# Generate cover prompt only
md2wechat write --style dan-koe --cover-only

# Remove AI writing traces (humanize)
md2wechat humanize article.md

# Humanize with intensity
md2wechat humanize article.md --intensity aggressive

# Write with humanize
md2wechat write --style dan-koe --humanize

# Create image post (小绿书/newspic)
md2wechat create_image_post -t "Title" --images photo1.jpg,photo2.jpg

# Extract images from Markdown
md2wechat create_image_post -t "Title" -m article.md

# With description and comments
md2wechat create_image_post -t "Title" -c "Description" --images photo.jpg --open-comment

# Preview mode (dry-run)
md2wechat create_image_post -t "Test" --images a.jpg,b.jpg --dry-run
```

---

## Image Posts (小绿书/Newspic)

Create WeChat image-only posts (小绿书/图片消息) with up to 20 images.

### Natural Language

```
"Create an image post titled 'Weekend Trip' with photos from ./photos/"
"Make a 小绿书 post with travel.md images"
"Upload these images as an image post: a.jpg, b.jpg, c.jpg"
```

### Command Options

| Flag | Short | Description |
|------|-------|-------------|
| `--title` | `-t` | Post title (required) |
| `--content` | `-c` | Description text |
| `--images` | | Comma-separated image paths |
| `--from-markdown` | `-m` | Extract images from Markdown file |
| `--open-comment` | | Enable comments |
| `--fans-only` | | Only fans can comment |
| `--dry-run` | | Preview without uploading |
| `--output` | `-o` | Save result to JSON file |

### Examples

```bash
# Basic image post
md2wechat create_image_post \
  -t "Weekend Trip" \
  --images photo1.jpg,photo2.jpg,photo3.jpg

# Extract images from article
md2wechat create_image_post \
  -t "Travel Diary" \
  -m article.md

# With description and comments enabled
md2wechat create_image_post \
  -t "Food Blog" \
  -c "Today's lunch" \
  --images food.jpg \
  --open-comment

# Read description from stdin
echo "Daily check-in" | md2wechat create_image_post \
  -t "Daily" \
  --images pic.jpg

# Preview mode
md2wechat create_image_post \
  -t "Test" \
  --images a.jpg,b.jpg \
  --dry-run
```

### Notes

- **Image limit**: Maximum 20 images per post
- **Image format**: JPG, PNG, GIF (same as regular articles)
- **Content**: Plain text only, not HTML
- **Requires**: `WECHAT_APPID` and `WECHAT_SECRET` for upload

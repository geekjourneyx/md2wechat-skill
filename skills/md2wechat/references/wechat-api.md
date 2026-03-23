# 微信公众号 API 参考

## SDK 使用

使用 `github.com/silenceper/wechat/v2` SDK。

```go
import (
    "github.com/silenceper/wechat/v2"
    "github.com/silenceper/wechat/v2/officialaccount/config"
    "github.com/silenceper/wechat/v2/officialaccount/material"
    "github.com/silenceper/wechat/v2/officialaccount/draft"
)
```

## 初始化

```go
// 创建微信实例
wc := wechat.NewWechat()

// 配置缓存（必需）
memory := cache.NewMemory()

// 配置公众号参数
cfg := &config.Config{
    AppID:     appID,
    AppSecret: appSecret,
    Cache:     memory,
}

// 获取公众号实例
officialAccount := wc.GetOfficialAccount(cfg)
```

## API 1: 上传永久素材

上传图片到微信素材库，返回 media_id 和 URL。

### 调用方式

```bash
md2wechat upload_image <file_path>
md2wechat download_and_upload <url>
```

### Go 实现

```go
// 获取素材管理器
materialManager := officialAccount.GetMaterial()

// 上传永久素材
mediaID, url, err := materialManager.AddMaterial(
    material.MediaTypeImage,  // 素材类型
    file,                       // *os.File 或 io.Reader
)
```

### 响应格式

```json
{
  "success": true,
  "media_id": "media_id_xxx",
  "wechat_url": "https://mmbiz.qpic.cn/mmbiz_jpg/xxx/0?wx_fmt=jpeg"
}
```

### 错误码

| 错误码 | 说明 | 处理方式 |
|--------|------|----------|
| 40001 | AppID 错误 | 检查配置 |
| 40004 | 文件为空 | 检查文件路径 |
| 40005 | 文件类型不支持 | 检查图片格式 |
| 40006 | 文件大小超限 | 压缩图片 |
| 42001 | AppSecret 错误 | 检查配置 |

## API 2: 新建草稿 (draft/add)

创建素材到公众号草稿箱。支持两种类型：

- **图文消息 (news)**: 传统图文文章
- **图片消息 (newspic)**: 小绿书，纯图片帖子

> **注意**: 上传到草稿箱中的素材被群发或发布后，该素材将从草稿箱中移除。

### 调用方式

```
POST https://api.weixin.qq.com/cgi-bin/draft/add?access_token=ACCESS_TOKEN
```

### CLI 命令

```bash
# 图文消息
md2wechat create_draft --file article.html --cover cover.jpg --title "文章标题" --desc "文章摘要"

# 图片消息（小绿书）
md2wechat create_image_post -t "标题" --images photo1.jpg,photo2.jpg
```

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| articles | array | 是 | 图文素材集合 |

### 文章字段说明 (articles[])

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| article_type | string | 否 | 文章类型：`news`（图文消息，默认）、`newspic`（图片消息/小绿书） |
| title | string | 是 | 标题，不超过 32 字符 |
| author | string | 否 | 作者，不超过 16 字符 |
| digest | string | 否 | 摘要，不超过 128 字符（仅单图文有效） |
| content | string | 是 | 正文内容。图文消息：HTML，< 2 万字符，< 1MB；图片消息：纯文本 |
| content_source_url | string | 否 | 原文链接，不超过 1KB |
| thumb_media_id | string | 条件 | 封面图素材 ID（图文消息必填，必须是永久 MediaID） |
| need_open_comment | number | 否 | 是否打开评论：0 不打开（默认），1 打开 |
| only_fans_can_comment | number | 否 | 是否仅粉丝可评论：0 所有人（默认），1 仅粉丝 |
| pic_crop_235_1 | string | 否 | 封面 2.35:1 裁剪坐标，格式：`X1_Y1_X2_Y2` |
| pic_crop_1_1 | string | 否 | 封面 1:1 裁剪坐标，格式：`X1_Y1_X2_Y2` |
| image_info | object | 条件 | 图片消息必填，图片信息 |
| cover_info | object | 否 | 图片消息封面裁剪信息 |
| product_info | object | 否 | 商品信息（暂不支持） |

### 图片信息 (image_info)

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| image_list | array | 是 | 图片列表，最多 20 张，首张为封面 |
| image_list[].image_media_id | string | 是 | 图片素材 ID（必须是永久 MediaID） |

### 封面裁剪 (cover_info)

| 字段 | 类型 | 说明 |
|------|------|------|
| crop_percent_list | array | 裁剪信息列表 |
| crop_percent_list[].ratio | string | 裁剪比例：`1_1`、`16_9`、`2.35_1` |
| crop_percent_list[].x1/y1 | string | 左上角坐标（0-1） |
| crop_percent_list[].x2/y2 | string | 右下角坐标（0-1） |

### 返回参数

| 参数 | 类型 | 说明 |
|------|------|------|
| media_id | string | 草稿 media_id（不超过 128 字符） |

### 请求示例

#### 图文消息 (news)

```json
{
    "articles": [
        {
            "article_type": "news",
            "title": "文章标题",
            "author": "作者",
            "digest": "摘要",
            "content": "<p>正文内容</p>",
            "content_source_url": "https://example.com",
            "thumb_media_id": "R7Ifp6ogGOmtr3u...",
            "need_open_comment": 1,
            "only_fans_can_comment": 0,
            "pic_crop_235_1": "0.1945_0_1_0.5236",
            "pic_crop_1_1": "0.25_0_0.75_1"
        }
    ]
}
```

#### 图片消息 (newspic/小绿书)

```json
{
    "articles": [
        {
            "article_type": "newspic",
            "title": "周末出游",
            "content": "今天天气真好",
            "need_open_comment": 1,
            "only_fans_can_comment": 0,
            "image_info": {
                "image_list": [
                    {"image_media_id": "IMAGE_MEDIA_ID_1"},
                    {"image_media_id": "IMAGE_MEDIA_ID_2"},
                    {"image_media_id": "IMAGE_MEDIA_ID_3"}
                ]
            },
            "cover_info": {
                "crop_percent_list": [
                    {
                        "ratio": "1_1",
                        "x1": "0.166454",
                        "y1": "0",
                        "x2": "0.833545",
                        "y2": "1"
                    }
                ]
            }
        }
    ]
}
```

### 返回示例

```json
{
    "media_id": "MEDIA_ID"
}
```

### Go 实现

#### 图文消息（使用 SDK）

```go
draftManager := officialAccount.GetDraft()

articles := []draft.Article{
    {
        Title:            "文章标题",
        Author:           "作者",
        Digest:           "摘要",
        Content:          "<p>正文</p>",
        ThumbMediaID:     "thumb_media_id",
        ShowCoverPic:     1,
        ContentSourceURL: "https://example.com",
    },
}

mediaID, err := draftManager.AddDraft(articles)
```

#### 图片消息（直接调用 API，SDK 不支持）

```go
type NewspicArticle struct {
    Title              string           `json:"title"`
    Content            string           `json:"content"`
    ArticleType        string           `json:"article_type"`
    ImageInfo          NewspicImageInfo `json:"image_info"`
    NeedOpenComment    int              `json:"need_open_comment,omitempty"`
    OnlyFansCanComment int              `json:"only_fans_can_comment,omitempty"`
}

type NewspicImageInfo struct {
    ImageList []NewspicImageItem `json:"image_list"`
}

type NewspicImageItem struct {
    ImageMediaID string `json:"image_media_id"`
}

// 构造请求
req := map[string]any{
    "articles": []NewspicArticle{
        {
            Title:       "周末出游",
            Content:     "今天天气真好",
            ArticleType: "newspic",
            ImageInfo: NewspicImageInfo{
                ImageList: []NewspicImageItem{
                    {ImageMediaID: "media_id_1"},
                    {ImageMediaID: "media_id_2"},
                },
            },
            NeedOpenComment: 1,
        },
    },
}

// POST to API
url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/draft/add?access_token=%s", accessToken)
resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBody))
```

### 错误码

| 错误码 | 说明 | 解决方案 |
|--------|------|----------|
| 45002 | content size out of limit | 内容超过 2 万字符或 1MB，需精简 |
| 45004 | title too long | 标题超过 32 字符 |
| 45005 | digest too long | 摘要超过 128 字符 |
| 53404 | 账号已被限制带货能力 | 删除商品后重试 |
| 53405 | 插入商品信息有误 | 检查商品参数及状态 |
| 53406 | 请先开通带货能力 | 开通带货功能 |

### 适用范围

| 账号类型 | 可调用 |
|----------|--------|
| 公众号 | ✔ |
| 服务号 | ✔ |

## 认证配置

### 环境变量

```bash
export WECHAT_APPID="your_appid"
export WECHAT_SECRET="your_secret"
```

### Go 代码读取

```go
func loadConfig() (*Config, error) {
    return &Config{
        AppID:     os.Getenv("WECHAT_APPID"),
        AppSecret: os.Getenv("WECHAT_SECRET"),
    }, nil
}
```

## 图片生成 API

使用兼容 OpenAI DALL-E 的 API。

### 配置

```bash
export IMAGE_API_KEY="your_api_key"
export IMAGE_API_BASE="https://api.example.com/v1"
```

### 调用

```go
// 生成图片
type ImageAPIRequest struct {
    Prompt string `json:"prompt"`
    Size   string `json:"size"` // 1024x1024
    N      int    `json:"n"`    // 1
}

type ImageAPIResponse struct {
    Data []struct {
        URL string `json:"url"`
    } `json:"data"`
}

// POST /images/generations
```

### 错误处理

| 错误 | 处理方式 |
|------|----------|
| API Key 无效 | 返回错误，提示检查配置 |
| 配额超限 | 返回错误，提示稍后重试 |
| 生成失败 | 返回错误，跳过该图片 |
| 超时 | 重试 1 次，仍失败则跳过 |

## 最佳实践

### 1. 并发限制

微信公众号 API 有调用频率限制：

| API | 限制 |
|-----|------|
| 上传素材 | 100 次/天 |
| 创建草稿 | 100 次/天 |

建议：
- 批量处理时控制并发数
- 失败后等待 1 秒再重试

### 2. 缓存策略

```go
// 缓存已上传的图片，避免重复上传
type ImageCache struct {
    sync.RWMutex
    cache map[string]string // localPath -> mediaID
}

func (c *ImageCache) Get(path string) (string, bool) {
    c.RLock()
    defer c.RUnlock()
    id, ok := c.cache[path]
    return id, ok
}

func (c *ImageCache) Set(path, id string) {
    c.Lock()
    defer c.Unlock()
    c.cache[path] = id
}
```

### 3. 错误重试

```go
func retry(fn func() error, maxAttempts int) error {
    var err error
    for i := 0; i < maxAttempts; i++ {
        err = fn()
        if err == nil {
            return nil
        }
        time.Sleep(time.Second)
    }
    return err
}
```

### 4. 日志记录

```go
type Logger struct {
    *zap.Logger
}

func (l *Logger) LogUpload(filePath string, mediaID, url string, duration time.Duration) {
    l.Info("image uploaded",
        zap.String("file", filePath),
        zap.String("media_id", maskMediaID(mediaID)),
        zap.String("url", url),
        zap.Duration("duration", duration),
    )
}

func maskMediaID(id string) string {
    if len(id) < 8 {
        return "***"
    }
    return id[:4] + "***" + id[len(id)-4:]
}
```

## 完整调用示例

```go
package main

import (
    "fmt"
    "os"
    "github.com/silenceper/wechat/v2"
    "github.com/silenceper/wechat/v2/cache"
    "github.com/silenceper/wechat/v2/officialaccount/config"
    "github.com/silenceper/wechat/v2/officialaccount/material"
    "github.com/silenceper/wechat/v2/officialaccount/draft"
)

func main() {
    // 1. 初始化
    wc := wechat.NewWechat()
    memory := cache.NewMemory()
    cfg := &config.Config{
        AppID:     os.Getenv("WECHAT_APPID"),
        AppSecret: os.Getenv("WECHAT_SECRET"),
        Cache:     memory,
    }
    oa := wc.GetOfficialAccount(cfg)

    // 2. 上传图片
    mat := oa.GetMaterial()
    file, _ := os.Open("image.jpg")
    defer file.Close()
    mediaID, url, _ := mat.AddMaterial(material.MediaTypeImage, file)
    fmt.Printf("Uploaded: %s, %s\n", mediaID, url)

    // 3. 创建草稿
    dm := oa.GetDraft()
    articles := []draft.Article{
        {
            Title:        "测试文章",
            Content:      "<p>正文内容</p>",
            ThumbMediaID: mediaID,
            ShowCoverPic: 1,
        },
    }
    draftID, _ := dm.AddDraft(articles)
    fmt.Printf("Draft: %s\n", draftID)
}
```

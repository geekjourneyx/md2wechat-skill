# 微信凭证与 IP 白名单指南

这份文档专门解决 4 个最常见的问题：

1. `WECHAT_APPID` 和 `WECHAT_SECRET` 去哪里拿
2. 为什么明明填了凭证，调用微信接口还是失败
3. 微信接口 IP 白名单在哪里配
4. 配好之后，怎么写回 `md2wechat` 并验证

如果你是第一次接触微信公众号开发，按这份文档一步一步做就够了。

> 扩展阅读：
> - 微信开发者平台：https://developers.weixin.qq.com/platform
> - 参考文章：https://www.md2wechat.com/zh/blog/wechat-appid-appsecret-guide

## 先知道两件事

### 1. `AppID` 和 `AppSecret` 是什么

- `AppID`：你的公众号在微信侧的唯一标识
- `AppSecret`：你的公众号调用微信接口时使用的密钥

`md2wechat` 在以下场景会用到它们：

- 上传图片到微信素材库
- 创建微信草稿
- 创建图片消息

### 2. 为什么还需要 IP 白名单

微信不是只看 `AppID` 和 `AppSecret`。  
对于很多接口，它还要求“发请求的机器公网 IP”必须在微信后台白名单里。

所以你会遇到这种情况：

- `AppID` 和 `AppSecret` 明明没填错
- 但接口仍返回：

```text
ip xxx.xxx.xxx.xxx not in whitelist
```

这不是 `md2wechat` 代码问题，而是微信的前置安全限制。

---

## 一、如何获取 AppID 和 AppSecret

### 步骤 1：登录微信开发者平台

打开：

```text
https://developers.weixin.qq.com/platform
```

使用你的公众号管理员微信扫码登录。

> 注：仓库现有文档和用户反馈表明，开发接口管理入口已经迁移到微信开发者平台，不要再只去旧版公众号后台里找。

### 步骤 2：选择你的公众号

登录后，选择你要接入 `md2wechat` 的那个公众号。

如果你有多个公众号，注意不要选错。

### 步骤 3：进入“开发接口管理”

进入公众号后，找到：

- `开发接口管理`

不同时间点微信后台文案可能略有变化，但核心目标是进入能看到开发者 ID 和开发者密码的页面。

### 步骤 4：复制 AppID

在开发接口管理页面，找到：

- `开发者ID(AppID)`

直接复制保存。

### 步骤 5：获取 AppSecret

在同一页面，找到：

- `开发者密码(AppSecret)`

通常需要点击：

- `重置`

然后完成管理员验证后，微信会给你新的 `AppSecret`。

> 注意：
> - `AppSecret` 很重要，不要发给别人
> - 不要提交到 Git 仓库
> - 如果你点击了“重置”，旧的 `AppSecret` 会失效

---

## 二、把凭证写到 md2wechat

推荐方式是写配置文件。

### 步骤 1：生成配置文件

```bash
md2wechat config init
```

默认配置文件路径：

```text
~/.config/md2wechat/config.yaml
```

### 步骤 2：填入微信配置

打开配置文件，填写：

```yaml
wechat:
  appid: "你的公众号 AppID"
  secret: "你的公众号 AppSecret"
```

如果你更喜欢环境变量，也可以：

```bash
export WECHAT_APPID="你的公众号 AppID"
export WECHAT_SECRET="你的公众号 AppSecret"
```

### 步骤 3：验证配置是否生效

```bash
md2wechat config validate
md2wechat config show --format json
```

你应该重点确认：

- `config_file`
- `wechat.appid`
- 当前是不是你预期的配置文件

---

## 三、如何配置微信接口 IP 白名单

### 什么时候必须配

如果你要做这些事，就要高度怀疑自己需要白名单：

- `md2wechat upload_image`
- `md2wechat convert --upload`
- `md2wechat convert --draft`
- `md2wechat create_image_post`
- `md2wechat test-draft`

### 白名单报错长什么样

最常见的报错形态：

```text
ip xxx.xxx.xxx.xxx not in whitelist
```

或类似含义的微信错误。

### 步骤 1：先查出当前执行机的公网 IP

在你实际运行 `md2wechat` 的那台机器上执行：

```bash
curl ifconfig.me
```

如果不可用，也可以试：

```bash
curl ip.sb
curl ipinfo.io/ip
```

> 一定要在“实际发起微信请求的机器”上执行。
>
> 例如：
> - 你在本地电脑运行，就查本地电脑当前公网 IP
> - 你在云服务器运行，就查云服务器公网 IP
> - 你在 CI 里运行，就查 CI 出口 IP

### 步骤 2：回到微信开发者平台

仍然进入：

- 公众号
- `开发接口管理`

### 步骤 3：找到“IP 白名单”

在开发接口管理页面中找到：

- `IP白名单`

然后点击：

- `设置`
  或
- `修改`

### 步骤 4：把公网 IP 加进去

把刚刚查到的公网 IP 填进去。

如果你有多个固定出口 IP，可以一起填。通常多个 IP 可按页面提示分隔填写。

### 步骤 5：保存并等待几分钟

白名单往往不是瞬间生效，通常建议：

- 等 1 到 5 分钟
- 然后再重试上传或建草稿

---

## 四、最常见的坑

### 1. 本地电脑和服务器不是同一个 IP

很多人会在本地查出一个 IP，结果程序实际跑在云服务器上。  
这样加到白名单里当然没用。

记住一句话：

**白名单里要加的是“真正发微信请求的那台机器的公网 IP”。**

### 2. 公司网络 / 家庭宽带 IP 会变

如果你是在家里电脑或公司网络环境里跑：

- 路由器重连后公网 IP 可能变化
- 运营商也可能重新分配出口 IP

这时你昨天能用，今天突然白名单报错，是正常现象。

### 3. GitHub Actions / 动态云环境不适合直接调微信

如果运行环境 IP 不固定，白名单维护会很痛苦。  
这种场景更适合：

- 固定一台有公网 IP 的服务器
- 或固定出口网关

### 4. 重置 AppSecret 后忘了更新配置

一旦你在微信后台点击了 `重置`：

- 旧 `AppSecret` 会失效
- 你必须同步更新 `~/.config/md2wechat/config.yaml` 或环境变量

---

## 五、推荐的最小验证顺序

配置完成后，建议按这个顺序测：

### 1. 先验证配置

```bash
md2wechat config validate
```

### 2. 再试单张图片上传

```bash
md2wechat upload_image ./cover.png --json
```

如果这一步失败，优先排查：

- `AppID` / `AppSecret`
- IP 白名单
- 图片格式

### 3. 再试建草稿

```bash
md2wechat test-draft ./article.html ./cover.png --json
```

### 4. 最后再跑完整主链

```bash
md2wechat convert article.md --upload --draft --cover cover.png --json
```

---

## 六、遇到问题先看什么

### 报 `WECHAT_APPID is required`

先看：

- [CONFIG.md](CONFIG.md)
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md)

### 报 `ip not in whitelist`

先看：

- 本文档的“如何配置微信接口 IP 白名单”
- [FAQ.md](FAQ.md)
- [SMOKE.md](SMOKE.md)

### 不确定现在到底用了哪份配置

执行：

```bash
md2wechat config show --format json
```

重点看：

- `config_file`
- 当前图片 provider

---

## 七、给新手的最终建议

如果你只想最快跑通，不要一上来就调完整链路。

按这个最稳：

1. 拿到 AppID 和 AppSecret
2. 写进 `~/.config/md2wechat/config.yaml`
3. 查当前执行机公网 IP
4. 把公网 IP 加到微信开发者平台白名单
5. 先测 `upload_image`
6. 再测 `test-draft`
7. 最后测 `convert --draft`

这样定位问题最快，也最不容易把“配置错”“白名单错”“图片错”混在一起。

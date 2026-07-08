# 安装指南

本文档详细说明 md2wechat 的各种安装方式。

## 目录

- [系统要求](#系统要求)
- [方式一：NPM 全局安装](#方式一npm-全局安装推荐)
- [方式二：Homebrew tap（macOS / Linux）](#方式二homebrew-tapmacos--linux)
- [方式三：Go install（已有 Go 环境时可选）](#方式三go-install已有-go-环境时可选)
- [方式四：安装脚本](#方式四安装脚本固定版本-release-资产)
- [方式五：预编译二进制](#方式五预编译二进制)
- [方式六：从源码编译](#方式六从源码编译)
- [验证安装](#验证安装)
- [卸载](#卸载)

---

## 系统要求

- **操作系统**：Linux / macOS / Windows
- **Node 版本**：18 或更高（如果使用 NPM 安装）
- **Go 版本**：1.26.1 或更高（如果从源码编译）
- **网络**：需要访问微信公众号 API

---

## 方式一：NPM 全局安装（推荐）

如果你的机器上已经有稳定可用的 Node/npm 环境，推荐直接执行：

```bash
npm install -g @geekjourneyx/md2wechat
```

这个路径适合：
- 已经用 npm 管理开发机上的 CLI
- 希望避免本地重新编译 Go 项目
- 希望 NPM、Homebrew、`install.sh` 复用同一批 GitHub Release 产物

这个包不会走 `latest` 漂移，也不会在安装时重新构建源码。它会下载与 `package.json` 版本同号的 GitHub Release 二进制，并校验同一个 `checksums.txt`。

当前 npm 安装目标矩阵：

- macOS: `amd64` / `arm64`
- Linux: `amd64` / `arm64`
- Windows: `amd64`

如果你的 npm 默认指向 `https://registry.npmmirror.com`，而新版本刚发布，镜像 tarball 可能会短暂 `404`。先用官方源安装：

```bash
npm install -g @geekjourneyx/md2wechat --registry=https://registry.npmjs.org/
```

维护者在 npm 发布新版本后，也应手动执行一次：

```bash
npx cnpm sync @geekjourneyx/md2wechat
```

这样可以尽快把新版本同步到 `npmmirror`。

如果你的全局 npm bin 目录还没加入 `PATH`，先修复 npm 的全局命令路径，再重新打开终端验证：

```bash
md2wechat version --json
md2wechat skills read md2wechat --json
```

---

## 方式二：Homebrew tap（macOS / Linux）

如果你的环境已经使用 Homebrew 管理 CLI，也可以执行：

```bash
brew install geekjourneyx/tap/md2wechat
```

升级：

```bash
brew upgrade geekjourneyx/tap/md2wechat
```

这个路径只安装 CLI 本身，不会运行远程 installer。Homebrew formula 会直接下载当前版本对应平台的预编译归档并安装 `md2wechat`。

---

## 方式三：Go install（已有 Go 环境时可选）

如果你的机器上已经有稳定可用的 Go 环境，也可以直接执行：

```bash
go install github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@v2.10.1
```

这是一个可选路径，不是默认推荐路径。

更适合：
- 已经长期使用 Go 工具链
- 希望直接用 Go 模块安装 CLI
- 不依赖 Homebrew

如果你只是普通用户，仍然优先使用 NPM。macOS 用户也可以按团队习惯选择 Homebrew。
如果你更关心固定版本 release 资产和 checksum，可继续使用下面的安装脚本。

---

## 方式四：安装脚本（固定版本 release 资产）

### macOS / Linux

```bash
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.10.1/install.sh | bash
```

### Windows PowerShell

```powershell
$env:MD2WECHAT_RELEASE_BASE_URL = "https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.10.1"
iex ((New-Object System.Net.WebClient).DownloadString("$env:MD2WECHAT_RELEASE_BASE_URL/install.ps1"))
```

安装脚本会下载对应 release 资产并校验 `checksums.txt`。主路径应始终指向固定版本的 release，不要使用 `latest` 或 `main` 作为安装入口。

默认安装路径：

- macOS / Linux: `~/.local/bin/md2wechat`
- Windows（普通用户）: `%USERPROFILE%\\AppData\\Local\\md2wechat\\md2wechat.exe`
- Windows（管理员）: `C:\\Program Files\\md2wechat\\md2wechat.exe`

如果 macOS / Linux 当前终端执行完脚本后仍然找不到命令，先运行：

```bash
export PATH="$HOME/.local/bin:$PATH"
md2wechat version --json
md2wechat skills read md2wechat --json
```

如果 Windows 当前 PowerShell 会话里仍然找不到命令，直接运行安装器输出的绝对路径验证命令即可。

---

## 方式五：预编译二进制

### 下载地址

访问 [Releases](https://github.com/geekjourneyx/md2wechat-skill/releases) 页面下载与你目标版本匹配的资产。

| 系统 | 文件名 |
|------|--------|
| Linux (amd64) | `md2wechat-linux-amd64` |
| Linux (arm64) | `md2wechat-linux-arm64` |
| macOS (Intel) | `md2wechat-darwin-amd64` |
| macOS (Apple Silicon) | `md2wechat-darwin-arm64` |
| Windows (64位) | `md2wechat-windows-amd64.exe` |
| Installer shell script | `install.sh` |
| Installer PowerShell script | `install.ps1` |

### 安装步骤

#### Linux / macOS

```bash
VERSION=v2.10.1
ASSET=md2wechat-linux-amd64
# macOS 请改成 md2wechat-darwin-amd64 或 md2wechat-darwin-arm64
curl -LO https://github.com/geekjourneyx/md2wechat-skill/releases/download/${VERSION}/${ASSET}
curl -LO https://github.com/geekjourneyx/md2wechat-skill/releases/download/${VERSION}/checksums.txt
sha256sum -c checksums.txt --ignore-missing

# 2. 添加执行权限
chmod +x "${ASSET}"

# 3. 移动到 PATH
sudo mv "${ASSET}" /usr/local/bin/md2wechat

# 4. 验证
md2wechat version --json
```

#### Windows

```powershell
# 1. 下载与你目标版本匹配的 release 资产
# 2. 同时下载 install.ps1 / install.sh 和 checksums.txt 并校验

# 3. 验证
md2wechat.exe version --json
```

---

## 方式六：从源码编译

```bash
git clone https://github.com/geekjourneyx/md2wechat-skill.git
cd md2wechat-skill
go build -o md2wechat ./cmd/md2wechat
```

如果你需要交叉编译，可以直接设置 `GOOS` / `GOARCH` 后再运行 `go build`。

---

## 关于 Docker

仓库当前不提供官方 Docker 镜像或 Dockerfile。若后续补齐容器化发布流程，再以 release 文档为准。

---

## 验证安装

运行以下命令验证安装成功：

```bash
# 查看帮助
md2wechat version --json
md2wechat --help

# 查看所有命令
md2wechat help

# 配置校验是独立动作，不是安装验证的一部分
```

如果当前 shell 还没刷新 PATH，也可以直接运行安装目录里的二进制：

```bash
~/.local/bin/md2wechat version --json
~/.local/bin/md2wechat --help
```

预期输出：

```
md2wechat converts Markdown articles to WeChat Official Account format
...
```

---

## 卸载

### Go 工具链安装

```bash
rm $(go env GOPATH)/bin/md2wechat
```

### 预编译二进制

```bash
# Linux/macOS
sudo rm /usr/local/bin/md2wechat

# Windows
# 删除安装目录下的 md2wechat.exe
```

---

## 下一步

安装完成后，请继续阅读 [配置指南](CONFIG.md) 和 [使用教程](USAGE.md)。

# Docs 目录说明

本目录为 **DevAgent** 的开源项目介绍页源码，用于 [GitHub Pages](https://pages.github.com/) 发布，**零运行成本**。

## 启用 GitHub Pages

1. 打开仓库 **Settings** → **Pages**
2. 在 **Build and deployment** 中：
   - **Source** 选择 **Deploy from a branch**
   - **Branch** 选择 `main`（或你的默认分支）
   - **Folder** 选择 **/docs**
3. 保存后等待构建，访问 `https://<你的用户名>.github.io/dev-agent/` 即可看到介绍页

## 目录结构

- `index.md` — 介绍页正文（中英双语，Markdown）
- `_config.yml` — Jekyll 配置（标题、描述、主题）
- `README.md` — 本说明

内容均使用 **Markdown** 编写，由 GitHub 提供的 Jekyll 自动构建为静态站点，无需自建服务器。

# 文档部署说明

本文档说明如何部署 VitePress 文档网站到 GitHub Pages。

## GitHub Pages 设置

### 1. 启用 GitHub Pages

1. 前往仓库的 **Settings** > **Pages**
2. 在 **Source** 部分，选择：
   - **Source**: `GitHub Actions`
3. 保存设置

### 2. 工作流说明

GitHub Actions 工作流文件位于 `.github/workflows/docs.yml`，会在以下情况自动触发：

- 推送到 `master` 或 `main` 分支的 `docs/` 目录更改
- 手动触发（workflow_dispatch）

### 3. 部署流程

工作流会自动：

1. 检出代码
2. 设置 Node.js 和 pnpm
3. 安装依赖
4. 构建 VitePress 文档
5. 部署到 GitHub Pages

### 4. 访问文档

部署完成后，文档将可通过以下 URL 访问：

```
https://go-zoox.github.io/ingress/
```

### 5. 本地测试

在部署前，可以在本地测试构建：

```bash
cd docs
pnpm install
pnpm run build
pnpm run preview
```

预览地址：`http://localhost:4173`

## 故障排除

### 构建失败

- 检查 Node.js 版本（需要 20+）
- 确保 `docs/package.json` 中的依赖正确
- 查看 GitHub Actions 日志获取详细错误信息

### 页面 404

- 确认 GitHub Pages 已启用
- 检查 `.vitepress/config.ts` 中的 `base` 路径是否正确
- 等待几分钟让 GitHub Pages 更新

### 缓存问题

如果遇到依赖问题，可以清除 GitHub Actions 缓存或使用 `--no-cache` 选项。

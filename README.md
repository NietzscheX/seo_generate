# SEO内容生成系统

基于Go语言的SEO内容生成系统，集成5118关键词API和AI模型生成优化内容。

## 功能特点

- 自动获取并处理养生/中医/修行行业关键词
- 基于DeepSeek/Ollama API生成高质量内容
- SEO优化组件（自动生成meta描述、sitemap、结构化数据等）
- 响应式Web展示界面

## 技术栈

- Go语言 + Gin框架
- PostgreSQL 15+
- Redis缓存
- DeepSeek/Ollama API集成

## 安装与使用

### 环境要求

- Go 1.21+
- PostgreSQL 15+
- Redis (可选，用于队列)

### 安装步骤

1. 克隆仓库
```
git clone https://github.com/NietzscheX/seo-generate.git
cd seo-generate
```

2. 安装依赖
```
go mod tidy
```

3. 配置环境变量
```
cp .env.example .env
# 编辑.env文件，填入必要的配置信息
```

4. 运行服务
```
go run cmd/server/main.go
```

## 配置说明

在`.env`文件中配置以下参数：

- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`: PostgreSQL数据库配置
- `5118_API_KEY`: 5118 API密钥
- `DEEPSEEK_API_KEY`: DeepSeek API密钥
- `OLLAMA_ENDPOINT`: Ollama API端点
- `PORT`: Web服务端口

## 项目结构

```
seo-generate/
├── cmd/                # 应用入口
├── config/             # 配置管理
├── internal/           # 内部包
│   ├── api/            # API处理器
│   ├── database/       # 数据库连接
│   ├── models/         # 数据模型
│   ├── services/       # 业务逻辑
│   └── utils/          # 工具函数
├── pkg/                # 可重用包
│   ├── ai/             # AI模型交互
│   └── seo/            # SEO工具
└── web/                # Web资源
    ├── templates/      # HTML模板
    └── static/         # 静态资源
```

## 许可证

MIT
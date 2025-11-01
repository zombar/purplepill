# DocuTag

**AI-Powered Web Content Processing Platform**

DocuTag is a microservices-based platform built in Go that scrapes web pages, extracts content using AI, and performs comprehensive text analysis. Perfect for building searchable knowledge bases and content repositories.

## 🚀 Main Repository

**[docutag/platform](https://github.com/docutag/platform)** - Complete platform with all services, Docker orchestration, and documentation

## 📦 Service Repositories

| Repository | Description | Language |
|------------|-------------|----------|
| **[web](https://github.com/docutag/web)** | React-based web interface for content ingestion and search | JavaScript/React |
| **[controller](https://github.com/docutag/controller)** | Orchestration service with unified API and SEO endpoints | Go |
| **[scraper](https://github.com/docutag/scraper)** | Web scraping with AI-powered content extraction | Go |
| **[textanalyzer](https://github.com/docutag/textanalyzer)** | Text analysis, sentiment, and NER | Go |
| **[scheduler](https://github.com/docutag/scheduler)** | Cron-based task scheduling for automation | Go |

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      Public Users                        │
│              (Search Engines, Web Browsers)              │
└────────────────────┬────────────────────────────────────┘
                     │ SEO-optimized HTML
                     ▼
            ┌─────────────────┐
            │   Controller    │ ◄──── Admin Interface
            │   Port 9080     │       (Web App)
            └────────┬────────┘
                     │
        ┌────────────┼────────────┐
        │            │            │
        ▼            ▼            ▼
  ┌─────────┐  ┌──────────┐  ┌──────────┐
  │ Scraper │  │TextAnalyz│  │Scheduler │
  │ :9081   │  │er :9082  │  │  :9083   │
  └─────────┘  └──────────┘  └──────────┘
        │            │            │
        └────────────┴────────────┘
                     │
              ┌──────┴──────┐
              │  PostgreSQL │
              │   + Redis   │
              └─────────────┘
```

## ✨ Key Features

- **AI-Powered**: Uses Ollama for intelligent content extraction and analysis
- **SEO-Friendly**: Server-rendered HTML pages with structured data
- **Async Processing**: Redis + Asynq for reliable task queues
- **Full Observability**: Prometheus metrics, Grafana dashboards, distributed tracing
- **Docker Ready**: Complete Docker Compose setup for easy deployment

## 📚 Getting Started

```bash
# Clone the main platform repository
git clone https://github.com/docutag/platform.git
cd platform

# Start all services with Docker
make docker-up

# Access the web interface
open http://localhost:3000
```

## 🔗 Links

- **Documentation**: See [platform README](https://github.com/docutag/platform#readme)
- **Issues**: [Report bugs](https://github.com/docutag/platform/issues)
- **License**: MIT

---

<p align="center">
  <strong>Built with ❤️ using Go and React</strong>
</p>

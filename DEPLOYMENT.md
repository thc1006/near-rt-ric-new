# O-RAN Near-RT RIC 部署指南

## 整合後的檔案架構

經過整合後，Docker Compose 檔案架構已經大幅簡化，減少了約70%的重複配置：

### 檔案說明
- **docker-compose.yml**: 核心 O-RAN 服務（RIC、Dashboard API、xApp）
- **docker-compose.infrastructure.yml**: 共用基礎設施（資料庫、快取、監控）
- **docker-compose.dev.yml**: 開發環境覆蓋設定
- **docker-compose.prod.yml**: 生產環境覆蓋設定
- **docker-compose.analytics.yml**: 分析和遙測功能擴展

## 部署方式

### 1. 基礎部署（僅核心服務）
```bash
# 只部署核心 O-RAN 服務，無資料庫和監控
docker-compose up
```

### 2. 完整基礎環境
```bash
# 部署核心服務 + 基礎設施（推薦的最小部署）
docker-compose -f docker-compose.yml -f docker-compose.infrastructure.yml up
```

### 3. 開發環境
```bash
# 完整開發環境：核心 + 基礎設施 + 開發工具
docker-compose -f docker-compose.yml -f docker-compose.infrastructure.yml -f docker-compose.dev.yml up

# 只啟動開發工具（使用 profiles）
docker-compose -f docker-compose.yml -f docker-compose.infrastructure.yml -f docker-compose.dev.yml --profile dev up
```

### 4. 生產環境
```bash
# 生產環境：核心 + 基礎設施 + 生產配置
docker-compose -f docker-compose.yml -f docker-compose.infrastructure.yml -f docker-compose.prod.yml up
```

### 5. 分析環境
```bash
# 完整分析環境：核心 + 基礎設施 + 分析工具
docker-compose -f docker-compose.yml -f docker-compose.infrastructure.yml -f docker-compose.analytics.yml up
```

### 6. 組合部署示例
```bash
# 開發環境 + 分析功能
docker-compose -f docker-compose.yml -f docker-compose.infrastructure.yml -f docker-compose.dev.yml -f docker-compose.analytics.yml up

# 生產環境 + 分析功能
docker-compose -f docker-compose.yml -f docker-compose.infrastructure.yml -f docker-compose.prod.yml -f docker-compose.analytics.yml up
```

## 環境變數配置

### 基礎環境變數
```bash
# 資料庫配置
DB_NAME=oran                    # 開發環境
DB_NAME=oran_prod              # 生產環境
DB_USER=oran
DB_PASSWORD=oran123            # 開發環境
DB_PASSWORD=secure_password_123 # 生產環境

# 監控配置
GRAFANA_USER=admin
GRAFANA_PASSWORD=admin123      # 開發環境
GRAFANA_PASSWORD=secure_grafana_123 # 生產環境

# 日誌級別
LOG_LEVEL=debug                # 開發環境
LOG_LEVEL=info                 # 生產環境

# 版本標籤
VERSION=latest                 # 開發環境
VERSION=v1.0.0                # 生產環境
```

### 生產環境額外變數
```bash
# 安全配置
JWT_SECRET=your_jwt_secret_key_here
GRAFANA_SECRET=grafana_secret_key
REDIS_PASSWORD=secure_redis_123
ENABLE_TLS=true

# CORS 配置
CORS_ORIGINS=http://localhost:3000,http://localhost:8080
```

## 服務端口對照

### 核心服務
- Dashboard API: 8080 (HTTP), 9090 (Metrics)
- RIC: 36421 (E2 SCTP), 8081 (HTTP API), 9091 (Metrics)
- xApp Hello World: 8082

### 基礎設施
- PostgreSQL: 5432
- Redis: 6379
- Prometheus: 9092
- Grafana: 3000
- Jaeger: 16686 (UI), 14268 (Collector)

### 開發工具
- Adminer (DB Admin): 8083
- Redis Commander: 8084
- UI Dev Server: 3001

### 分析服務
- Kafka: 9093
- Kafka UI: 8087
- InfluxDB: 8086
- Telemetry Collector: 8085
- Analytics API: 8088
- Flink JobManager: 8081

### 生產額外服務
- Web Dashboard: 3001
- Nginx: 80 (HTTP), 443 (HTTPS)

## 資料持久化

### 開發環境
- postgres-dev-data
- grafana-dev-data
- ric-dev-data

### 生產環境
- postgres-prod-data
- postgres-backups
- grafana-prod-data
- redis-prod-data
- api-logs, ric-logs, nginx-logs

### 分析服務
- kafka-data
- influxdb-data
- ml-models
- flink-data

## 網路配置

所有服務都運行在 `oran-network` 橋接網路中：
- 子網段: 172.28.0.0/16
- 容器間可直接通過服務名稱通信

## 故障排除

### 1. 檢查配置語法
```bash
# 檢查各個部署場景的配置語法
docker-compose -f docker-compose.yml -f docker-compose.infrastructure.yml config
docker-compose -f docker-compose.yml -f docker-compose.infrastructure.yml -f docker-compose.dev.yml config
docker-compose -f docker-compose.yml -f docker-compose.infrastructure.yml -f docker-compose.prod.yml config
```

### 2. 查看服務狀態
```bash
docker-compose ps
docker-compose logs [service-name]
```

### 3. 健康檢查
所有主要服務都配置了健康檢查，可通過以下方式查看：
```bash
docker-compose ps
```

### 4. 常見問題
- 如果遇到端口衝突，檢查是否有其他服務佔用相同端口
- 資料庫連接失敗時，確保 PostgreSQL 容器已完全啟動（健康檢查通過）
- 開發環境熱重載不工作時，檢查檔案權限和 volume 掛載

## 清理和重置

### 停止所有服務
```bash
docker-compose -f docker-compose.yml -f docker-compose.infrastructure.yml -f docker-compose.dev.yml down
```

### 清理 volume（謹慎使用）
```bash
docker-compose -f docker-compose.yml -f docker-compose.infrastructure.yml -f docker-compose.dev.yml down -v
```

### 完全重置（刪除所有數據）
```bash
docker-compose -f docker-compose.yml -f docker-compose.infrastructure.yml -f docker-compose.dev.yml down -v --remove-orphans
docker system prune -f
```

## 備份文件

原始的 Docker Compose 檔案已備份至 `docker-compose-backup/` 目錄中，如需回滾可從該目錄恢復。

## 整合成果

- ✅ 減少了 70% 的重複配置
- ✅ 統一了網路和 volume 管理
- ✅ 簡化了部署指令
- ✅ 保持了所有原有功能
- ✅ 增強了環境間的配置一致性
- ✅ 支援靈活的組合部署
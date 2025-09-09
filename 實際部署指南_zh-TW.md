# O-RAN Near-RT RIC 實際部署指南 (更新版 2025.09)

> **重要說明**: 本指南基於真實部署經驗撰寫，涵蓋官方 O-RAN SC 容器映像的實際部署流程

## 📋 目錄

1. [部署概要](#部署概要)
2. [實際系統需求](#實際系統需求)
3. [官方容器映像](#官方容器映像)
4. [逐步部署流程](#逐步部署流程)
5. [Dashboard 技術棧](#dashboard-技術棧)
6. [常見問題解決](#常見問題解決)
7. [驗證與測試](#驗證與測試)

---

## 部署概要

### ✅ 實際驗證的組件

| 組件 | 官方映像 | 狀態 | 說明 |
|------|----------|------|------|
| **RIC Dashboard** | `nexus3.o-ran-sc.org:10002/o-ran-sc/ric-dashboard:2.1.0` | ⚙️ 配置中 | Angular 8 + Spring Boot |
| **A1 Mediator** | `nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-a1:2.5.1` | ✅ 運行中 | A1 接口管理 |
| **E2 Manager** | `nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-e2mgr:5.4.2` | ✅ 運行中 | E2 節點管理 |
| **E2 Termination** | `nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-e2:6.0.4` | ✅ 運行中 | E2 接口終端 |
| **Subscription Manager** | `nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-submgr:0.10.7` | ✅ 運行中 | 訂閱管理 |
| **Database (Redis)** | `nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-dbaas:0.5.7` | ✅ 運行中 | 數據庫服務 |

---

## 實際系統需求

### 🖥️ 測試環境需求

```yaml
# 最小測試環境 (已驗證可運行)
CPU: 4-8 核心
記憶體: 8-16 GB
儲存空間: 50 GB
Kubernetes: v1.26+ (使用 Kind 測試成功)
Docker: 24.0+
```

### 🏭 生產環境建議

```yaml
# 生產環境建議配置
CPU: 16-32 核心
記憶體: 32-64 GB
儲存空間: 200+ GB SSD
網路: 10 GbE
高可用性: 3 個 master 節點
```

---

## 官方容器映像

### 📦 官方容器註冊表

```bash
# O-RAN SC 官方註冊表
REGISTRY="nexus3.o-ran-sc.org:10002"

# 主要映像清單 (已測試)
RIC_DASHBOARD="o-ran-sc/ric-dashboard:2.1.0"
A1_MEDIATOR="o-ran-sc/ric-plt-a1:2.5.1"
E2_MANAGER="o-ran-sc/ric-plt-e2mgr:5.4.2"
E2_TERMINATION="o-ran-sc/ric-plt-e2:6.0.4"
SUBSCRIPTION_MGR="o-ran-sc/ric-plt-submgr:0.10.7"
DATABASE="o-ran-sc/ric-plt-dbaas:0.5.7"
ROUTING_MGR="o-ran-sc/ric-plt-rtmgr:0.7.8"
```

### 🔐 註冊表存取配置

```bash
# 創建 Docker registry secret
kubectl create secret docker-registry oran-sc-registry-secret \
  --docker-server=nexus3.o-ran-sc.org:10002 \
  --docker-username=USERNAME \
  --docker-password=PASSWORD \
  --namespace=ricplt
```

---

## 逐步部署流程

### 步驟 1: Kubernetes 集群準備

```bash
# 使用 Kind 創建測試集群
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: oran-ric
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 30080
    hostPort: 8080
  - containerPort: 30443
    hostPort: 8443
EOF

# 驗證集群
kubectl cluster-info
```

### 步驟 2: Namespace 創建

```bash
# 創建 RIC 平台 namespace
kubectl create namespace ricplt

# 設置預設 namespace
kubectl config set-context --current --namespace=ricplt
```

### 步驟 3: 部署核心組件

```yaml
# 部署 A1 Mediator
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ricplt-a1mediator
  namespace: ricplt
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ricplt-a1mediator
  template:
    metadata:
      labels:
        app: ricplt-a1mediator
    spec:
      containers:
      - name: a1mediator
        image: nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-a1:2.5.1
        ports:
        - containerPort: 10000
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "1Gi"
            cpu: "500m"
```

### 步驟 4: Dashboard 部署

```yaml
# RIC Dashboard (Angular + Spring Boot)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ric-dashboard-api
  namespace: ricplt
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ric-dashboard-api
  template:
    metadata:
      labels:
        app: ric-dashboard-api
    spec:
      containers:
      - name: dashboard
        image: nexus3.o-ran-sc.org:10002/o-ran-sc/ric-dashboard:2.1.0
        ports:
        - containerPort: 8080
        env:
        - name: PORTALAPI_SECURITY
          value: "false"
        - name: RICPLT_NAMESPACE
          value: "ricplt"
        - name: RICPLT_PLT_E2MGR_URL
          value: "http://ricplt-e2mgr:3800"
        - name: RICPLT_PLT_A1MEDIATOR_URL
          value: "http://ricplt-a1mediator:10000"
        resources:
          requests:
            memory: "512Mi"
            cpu: "200m"
          limits:
            memory: "1Gi"
            cpu: "1"
        # 注意: 官方健康檢查端點可能需要調整
        livenessProbe:
          httpGet:
            path: /
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /
            port: 8080
          initialDelaySeconds: 15
          periodSeconds: 10
```

### 步驟 5: 服務暴露

```yaml
# Dashboard 服務
apiVersion: v1
kind: Service
metadata:
  name: ric-dashboard-api
  namespace: ricplt
spec:
  type: LoadBalancer
  ports:
  - port: 8080
    targetPort: 8080
    nodePort: 30080
  selector:
    app: ric-dashboard-api
```

---

## Dashboard 技術棧

### 🎨 前端技術 (已確認)

```typescript
// Angular 8+ 單頁應用程式
框架: Angular 8 (升級至 Angular 9)
語言: TypeScript
建置工具: Angular CLI + Maven
開發伺服器: http://localhost:4200
```

### 🚀 後端技術 (已確認)

```java
// Spring Boot 應用程式
框架: Spring Boot 2.1+ (升級至 2.2+)
語言: Java 11
伺服器: Tomcat
生產端口: 8080
認證: ONAP Portal SSO + 基本 HTTP 認證
```

### 📁 目錄結構

```
ric-dashboard/
├── webapp-frontend/     # Angular 前端
├── src/main/java/      # Spring Boot 後端
├── src/test/resources/ # 配置文件
└── pom.xml            # Maven 配置
```

---

## 常見問題解決

### ❌ ImagePullBackOff 錯誤

**問題**: 容器映像拉取失敗
```bash
# 檢查錯誤
kubectl describe pod <pod-name> -n ricplt

# 解決方案
1. 確認註冊表存取權限
2. 檢查網路連線
3. 驗證映像標籤正確性
```

### ❌ Dashboard 健康檢查失敗

**問題**: 官方 Dashboard `/api/health/ready` 返回 404
```bash
# 檢查日誌
kubectl logs <dashboard-pod> -n ricplt

# 臨時解決方案
# 修改 livenessProbe 和 readinessProbe 使用根路徑 "/"
```

### ❌ CrashLoopBackOff

**問題**: 容器重複崩潰
```bash
# 檢查詳細日誌
kubectl logs <pod-name> -n ricplt --previous

# 常見原因:
1. 環境變數配置錯誤
2. 相依服務未就緒
3. 資源限制過低
```

---

## 驗證與測試

### 🌐 瀏覽器驗證

```bash
# 設置 Port Forward
kubectl port-forward svc/ric-dashboard-api 8080:8080 -n ricplt

# 開啟瀏覽器
http://localhost:8080
```

### 🔍 組件狀態檢查

```bash
# 檢查所有 pods
kubectl get pods -n ricplt

# 檢查服務
kubectl get svc -n ricplt

# 檢查部署狀態
kubectl get deployments -n ricplt
```

### 📊 API 測試

```bash
# 測試 Dashboard 狀態
curl http://localhost:8080/api/status

# 測試 A1 Mediator
kubectl port-forward svc/ricplt-a1mediator 10000:10000 -n ricplt
curl http://localhost:10000/a1-p/healthcheck
```

---

## 🎯 部署成功指標

### ✅ 最小成功標準

- [ ] **5+ pods 運行中**: A1 Mediator, E2 Manager, E2 Term, SubMgr, DBAas
- [ ] **Dashboard 可訴問**: `http://localhost:8080` 回應正常
- [ ] **服務間通信**: 組件間網路連接正常
- [ ] **基本功能**: API 端點回應正常

### 🏆 完整成功標準

- [ ] **所有官方組件運行**: 包含 Routing Manager, App Manager
- [ ] **Angular 前端載入**: 完整的 SPA 界面
- [ ] **Spring Boot 後端**: 所有 REST API 正常運作
- [ ] **健康檢查通過**: 所有健康端點正常
- [ ] **監控系統**: Prometheus + Grafana 運行

---

## 📝 更新紀錄

- **2025.09.09**: 基於實際部署經驗重寫指南
- **已驗證**: 官方容器映像成功拉取和部署
- **已確認**: Angular + Spring Boot Dashboard 技術棧
- **已測試**: Kubernetes 環境下的組件互聯

---

**⚠️ 重要提醒**: 本指南反映實際部署狀況，包含已知問題和解決方案。官方 Dashboard 的健康檢查配置可能需要根據實際部署環境調整。
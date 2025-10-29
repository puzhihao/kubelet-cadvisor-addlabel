# Kubelet Demo - Kubernetes 指标收集代理

一个轻量级的 Kubernetes 指标收集代理，用于从集群节点收集 cAdvisor 指标并添加自定义标签。

## 功能特性

- 自动发现集群节点并收集指标
- 使用 ServiceAccount Token 认证访问 kubelet metrics 端点
- 支持为指标自动添加 Pod 标签
- 灵活的标签默认值配置
- Prometheus 兼容的指标格式输出
- 优雅的配置管理和错误处理

## 项目结构

```
├── cmd/                 # 应用程序入口点
│   ├── main.go         # 主程序入口，配置和启动逻辑
│   └── server.go       # HTTP 服务器实现
├── pkg/                # 核心业务逻辑包
│   └── services/       # 业务服务层
│       ├── kube_resource_service.go  # Kubernetes 资源管理服务
│       └── metrics_fetcher.go        # 指标抓取和处理服务
├── config/             # 配置管理
│   └── config.go       # 配置结构和环境变量解析
└── pkg/utils/          # 工具函数
    └── kubernetes_client.go  # Kubernetes 客户端工具
```

## 快速开始

### 前置条件

- Go 1.19+
- 容器运行时环境
- 对 Kubernetes 集群的访问权限

### 安装运行

1. 克隆项目：
```bash
git clone <repository-url>
cd kubelet-cadvisor-addlabel
```

2. 构建项目：
```bash
go build -o kubelet-demo ./cmd/
```

3. 运行应用：
```bash
./kubelet-demo
```

### 配置说明

通过环境变量配置应用：

| 环境变量 | 默认值 | 描述 |
|---------|-------|------|
| `PORT` | 9090 | HTTP 服务器监听端口 |
| `LOG_LEVEL` | info | 日志级别 (debug, info, warn, error) |
| `ADD_LABELS` | app | 要添加的标签列表，逗号分隔 |
| `LABEL_DEFAULTS` | test | 标签默认值，支持单值或键值对格式 |
| `FETCH_INTERVAL` | 30 | 指标抓取间隔（秒） |

**标签配置示例：**

```bash
# 添加单个标签，使用统一默认值
ADD_LABELS=app
LABEL_DEFAULTS=unknown

# 添加多个标签，使用统一默认值
ADD_LABELS=app,tier,env
LABEL_DEFAULTS=unknown

# 为不同标签指定不同默认值
ADD_LABELS=app,tier,env
LABEL_DEFAULTS="app=unknown,tier=backend,env=dev"
```

## API 接口

### 指标端点
- `GET /metrics` - 获取处理后的 Prometheus 指标数据
- `GET /health` - 健康检查接口

### 指标处理示例

输入指标（原始格式）：
```
container_cpu_load_average_10s{container="app",id="/path/to/container",namespace="default",pod="myapp-123"} 0
```

处理后指标（添加了标签）：
```
container_cpu_load_average_10s{container="app",id="/path/to/container",namespace="default",pod="myapp-123",app="myapp",tier="frontend"} 0
```

## 部署到 Kubernetes

### ServiceAccount 配置

应用需要以下权限：
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kubelet-demo
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kubelet-demo
rules:
- apiGroups: [""]
  resources: ["pods", "nodes"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubelet-demo
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kubelet-cadvisor-addlabel
subjects:
- kind: ServiceAccount
  name: kubelet-cadvisor-addlabel
  namespace: default
```

## 开发指南

### 代码结构说明

1. **配置模块 (config/)**
   - 管理应用程序配置
   - 支持环境变量覆盖

2. **服务模块 (pkg/services/)**
   - `kube_resource_service.go`: Kubernetes 资源监听和标签管理
   - `metrics_fetcher.go`: 指标抓取和处理流水线

3. **工具模块 (pkg/utils/)**
   - Kubernetes 客户端初始化和工具函数

### 扩展功能

要添加新的标签处理逻辑，可以修改 `AddLabelsToMetrics` 方法。
要支持新的指标源，可以扩展 `FetchAndProcessMetrics` 方法。

## 故障排除

### 常见问题

1. **无法连接 kubelet**
   - 检查 ServiceAccount Token 权限
   - 验证节点网络连通性

2. **指标标签未添加**
   - 检查 Pod 标签是否正确设置
   - 验证默认值配置格式

3. **内存使用过高**
   - 调整抓取间隔时间
   - 考虑指标过滤策略

## 许可证

[您的许可证信息]

## 贡献

欢迎提交 Issue 和 Pull Request！
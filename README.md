# 配置文件环境一致性检查工具 (Env-Check) 
## 1. 文档概述
### 1.1 目的
解决多环境（UAT/TEST/PROD）部署中，由于配置文件（YAML/Properties/TOML）不一致或 IP 跨环境污染导致的生产事故。

### 1.2 目标用户
开发人员（用于本地自测）、运维/DevOps（用于 CI/CD 流水线集成）。

---

## 2. 目录结构设计规范

### 2.1 工具自身目录
程序运行时会引用同级目录下的规则文件（envs/*.txt）：
```text
.
├── env-check              # 编译后的 Go 二进制程序
└── envs/                  # 规则定义目录（uat.txt 等）
    ├── uat.txt            # 存放 UAT 环境合法 IP (每行一个)
    ├── test.txt           # 存放 test 环境合法 IP
    └── prod.txt           # 存放 PROD 环境合法 IP
```

### 2.2 待测项目目录示例
程序需支持扫描如下结构，识别出 application 和 bootstrap 两组配置：
```text
my-service/
├── application-uat.yml
├── application-prod.yml
├── bootstrap-uat.properties
└── bootstrap-prod.properties
```

## 3. 核心功能需求
### 3.1 智能分组逻辑 (Grouping Logic)
程序必须能够识别同一个项目下的多组配置文件。

- 识别模式：{Prefix}-{Env}.{Extension}
- 分组策略：具有相同 Prefix 和 Extension 的文件归为一组。
- 示例：
    - application-uat.yml 与 application-prod.yml 归为 Group: application (yml)
    - log4j-uat.properties 与 log4j-prod.properties 归为 Group: log4j (properties)

### 3.2 配置一致性检查 (Structural Check)
- 平坦化处理：对比前需将嵌套结构（如 YAML）转换为 点分隔路径: 值 的格式。
- Key 集合对比：以 uat 为基准，检查 prod 是否缺失了对应的 Key。
- 判定标准：若 prod 缺少任一 Key，程序报 Critical 错误。

### 3.3 IP 合规性扫描 (IP Compliance)
1. 正则提取：从所有配置文件中提取符合 IPv4 格式的字符串。
2. 跨环境屏蔽：
    - *-prod.* 文件中若出现 uat.txt 或 test.txt 中的 IP，报 Critical。
3. 未知 IP 告警：
    - 若配置文件中的 IP 未在任何 .txt 规则文件中定义，报 Warning。

## 4. 核心执行流程
1. 加载规则：读取 rules/*.txt 到内存 Map。
2. 扫描文件：递归遍历目标目录，按“前缀+后缀”聚合文件组。
3. 循环校验：对每个文件组执行：
    - 解析文件内容 -> 扁平化 Key 列表。
    - 对比 Key 差异 -> 记录缺失项。
    - 提取 Value 中的 IP -> 匹配所属环境规则。
4. 生成报告：控制台输出着色报告，并根据扫描结果返回退出码（0/1/2）。

## 5. 开发技术规格
- 开发语言：Go 1.20+
- 核心库建议：
    - spf13/viper 或 test-go/ucl：处理多格式配置解析。
    - fatih/color：终端彩色输出。
    - regexp：IP 提取正则 \b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b
- 交付物：单个静态编译的二进制文件，支持 x86_64 和 ARM64 架构。

## 6. 验收标准 (Acceptance Criteria)
1. 多文件支持：能同时处理一个目录下的 application、bootstrap 等多组文件。
2. 高亮输出：错误信息（缺失 Key、跨环境 IP）必须用红色标记。
3. 退出策略：无问题返回 `0`；仅有警告（warnings）返回 `1`；出现严重错误（Critical）返回 `2`，便于在 CI 中区分阻断与告警场景。

## 7. 使用、构建与示例

### 7.1 快速构建
- 本项目使用 Go 编译：

```sh
go build -o env-check ./...
```

- 或使用发布脚本：

- Linux/macOS: `./build_release.sh`
- Windows PowerShell: `.\build_release.ps1`

### 7.2 运行示例
- 直接运行二进制扫描某个目录：

```sh
./env-check -dir ./my-service
```

- 开发时可使用 `go run`：

```sh
go run main.go -dir ./my-service
```

（实际 CLI 参数：`-dir` 用于指定待扫描目录，默认 `.`；`-envs` 用于指定环境规则目录，默认 `envs`。）

### 7.3 配置 / 规则文件位置
- 环境规则存放于 [envs](envs/) 下（例如 [envs/prod.txt](envs/prod.txt)）。
- 规则代码在 [rules](rules/) 目录。
- 示例待测服务配置示例位于 [my-service/app](my-service/app/)。

### 7.4 CI 集成要点
- 在 CI 中将项目编译后执行扫描，若有 Critical 错误需返回非 0 退出码，便于阻断流水线。
- 可在 CI 中使用 `./build_release.sh` 生成发行二进制并在流水线环境执行。

### 7.5 示例输出
下面给出若干示例输出，帮助在 CI/本地判断扫描结果类型：

- 无问题（退出 0）：

```sh
$ ./env-check -dir ./my-service -envs envs
No issues found
```

- 仅有警告（退出 1，需要关注）：

```sh
$ ./env-check -dir ./my-service -envs envs
Found 2 warnings
WARN: group=application missing key 'logging.level'
WARN: file=bootstrap-prod.properties contains unknown IP 10.1.2.3
```

- 出现严重错误（Critical，退出 2）：

```sh
$ ./env-check -dir ./my-service -envs envs
Found 1 critical issues
CRITICAL: group=application missing file for env 'prod'
CRITICAL: file=application-prod.yml contains prod IP from uat rules 192.168.1.10
```

### 7.6 真实运行输出（本仓库示例）
下面是基于本仓库直接运行得到的真实输出（命令在仓库根目录运行）：

```sh
$ go run main.go -dir ./my-service -envs ./envs
    CRITICAL: group=app/application.yml missing file for env 'test'
    CRITICAL: group=cmd/bootstrap.properties missing file for env 'test'

Checking group: app/application.yml
 - prod: my-service\app\application-prod.yml
    CRITICAL: group=app/application.yml prod missing key 'db.user' (baseline=uat) 
    CRITICAL: my-service\app\application-prod.yml contains IP 10.0.0.1 from test rules
 - uat: my-service\app\application-uat.yml

Checking group: cmd/bootstrap.properties
 - prod: my-service\cmd\bootstrap-prod.properties
 - uat: my-service\cmd\bootstrap-uat.properties
Found 4 critical issues
```

## 8. 项目快速参考
- 入口: [main.go](main.go)
- CLI/命令解析: [cmd/root.go](cmd/root.go)
- 规则加载: [rules/rules.go](rules/rules.go)
- 扫描器: [scan/scan.go](scan/scan.go)
- 解析器: [parse/parse.go](parse/parse.go)
- 环境规则: [envs](envs/)

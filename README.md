# IP 归属地公共服务

基于 [ip2region](https://github.com/lionsoul2014/ip2region) 官方 Go binding 和 Gin，提供离线 IPv4 / IPv6 查询。`pkg/ipregion` 可以独立被其他 Go 项目使用，HTTP 服务提供 JSON 接口。

## 启动

需要 Go 1.25 或更高版本。项目已包含官方 IPv4、IPv6 xdb 文件，启动和查询无需下载数据；首次构建需要下载 Go 依赖。

```bash
cd /Users/shuishui.kuang/Documents/work/git/box-im/ip-region-service
go run ./cmd/server
```

默认监听 `:8080`。编译运行：

```bash
make build
./bin/ip-region-service
```

以上命令在项目根目录执行，默认数据路径相对于当前工作目录。支持 SIGINT / SIGTERM 优雅停止。

| 环境变量 | 默认值 | 说明 |
| --- | --- | --- |
| HTTP_ADDR | :8080 | HTTP 监听地址 |
| IP2REGION_V4_PATH | data/ip2region_v4.xdb | IPv4 数据路径，空字符串禁用 IPv4 |
| IP2REGION_V6_PATH | data/ip2region_v6.xdb | IPv6 数据路径，空字符串禁用 IPv6 |

至少启用一个版本。例如只启用 IPv4：

```bash
HTTP_ADDR=:8081 IP2REGION_V6_PATH='' go run ./cmd/server
```

## HTTP 接口

公开接口，无需登录。允许浏览器跨域访问（`Access-Control-Allow-Origin: *`），不支持携带跨域凭证。若业务需要鉴权或流量限制，可在网关配置。

### GET /api/v1/ip

```bash
curl 'http://localhost:8080/api/v1/ip?ip=113.92.157.29'
curl 'http://localhost:8080/api/v1/ip?ip=240e:3b7:3272:d8d0:db09:c067:8d59:539e'
```

### POST /api/v1/ip

```bash
curl -X POST 'http://localhost:8080/api/v1/ip' \
  -H 'Content-Type: application/json' \
  -d '{"ip":"113.92.157.29"}'
```

成功返回（以随附数据文件为准）：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "ip": "113.92.157.29",
    "version": "IPv4",
    "country": "中国",
    "province": "广东省",
    "city": "深圳市",
    "isp": "电信",
    "country_code": "CN",
    "raw": "中国|广东省|深圳市|电信|CN"
  }
}
```

- `ip` 为规范化地址，自动去掉两端空白，IPv4-mapped IPv6 转为 IPv4。
- `version` 为 `IPv4` 或 `IPv6`。
- `country_code` 是 ISO alpha-2 国家代码。
- 官方数据的 `0` 占位字段转换为空字符串，`raw` 保留原始信息。
- 只接受纯 IP，不接受域名、端口、CIDR、IPv6 zone。内网地址等按数据库原样查询，可能返回保留地址描述，不代表真实位置。
- 仅支持当前官方五字段数据格式（国家、省份、城市、运营商、国家代码）；旧版“国家、区域、省份、城市、运营商”数据不可直接替换。

| HTTP 状态 | code | 含义 |
| --- | --- | --- |
| 200 | 0 | 查询成功 |
| 400 | 400 | 缺少 IP、参数错误、JSON 错误或请求体超过 4 KiB |
| 404 | 404 | 数据中无此 IP；或访问了不存在的接口 |
| 415 | 415 | POST 未使用 application/json |
| 422 | 422 | 此 IP 版本未配置数据文件 |
| 500 | 500 | 查询或服务内部错误 |

错误结构统一为 `{"code":400,"message":"IP 地址格式不正确","data":null}`。内部错误细节仅写到服务日志。健康检查：`GET /healthz`。

### 前端调用示例

```js
export async function lookupIP(ip) {
  const response = await fetch('http://localhost:8080/api/v1/ip', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ip }),
  })
  const result = await response.json()
  if (!response.ok || result.code !== 0) {
    throw new Error(result.message || 'IP 查询失败')
  }
  return result.data
}

const region = await lookupIP('113.92.157.29')
console.log(region.country, region.province, region.city, region.isp)
```

## 在其他 Go 项目中复用

当前 module path 为 `github.com/box-im/ip-region-service`，作为本地模块标识，尚未发布到远程仓库。本地接入可先执行：

```bash
go mod edit -replace=github.com/box-im/ip-region-service=/Users/shuishui.kuang/Documents/work/git/box-im/ip-region-service
go mod edit -require=github.com/box-im/ip-region-service@v0.0.0
```

然后在项目中引入：

```go
import "github.com/box-im/ip-region-service/pkg/ipregion"

client, err := ipregion.New(ipregion.Config{
    IPv4Path: "/absolute/path/ip2region_v4.xdb",
    IPv6Path: "/absolute/path/ip2region_v6.xdb",
})
if err != nil {
    return err
}
defer client.Close() // 在应用退出时调用，不要在每次 HTTP 请求后关闭

region, err := client.Lookup("113.92.157.29")
if err != nil {
    return err
}
fmt.Println(region.City)
```

引入后执行 `go mod tidy`。`New` 应在启动时调用一次；客户端可被多个 goroutine 共享，不可值拷贝。`Close` 幂等并等待正在执行的查询。错误可使用 `errors.Is` 判断 `ErrInvalidIP`、`ErrNotFound`、`ErrUnsupported`、`ErrClosed`。发布到真实仓库时统一修改 module path 及内部 import，再通过版本标签分发。

## 结构和实现

```text
cmd/server/                         启动、配置和优雅停止
pkg/ipregion/                       独立查询库，不依赖 Gin
internal/api/public/location/ip/     控制器接口、实现、DTO、VO
internal/response/                  统一 JSON 响应
router/public/location/            模块和领域路由
router/public/                     公开接口汇总
initialize/                        Gin 引擎、中间件和装配
data/                              IPv4、IPv6 数据及上游许可
```

启动时验证 xdb 格式及 IP 版本，并将数据全量加载到内存（数据文件合计约 47 MiB，另有运行时内存开销）。启动失败直接退出，不提供空数据服务。查询加互斥锁保护上游可变的 IO 计数状态，因此单客户端内查询串行执行；避免全内存搜索器共享导致的数据竞争。默认配置无文件查询 IO。

## 数据来源和更新

随附数据来自官方发布 `v3.18.0`，查询 binding 固定在提交 `6a276cc7c4f9`。来源、SHA-256 和许可见 [data/README.md](data/README.md)。官方数据为不定期更新的离线快照，结果精度和覆盖取决于数据文件。

更新时从官方获取相同五字段格式的 `ip2region_v4.xdb`、`ip2region_v6.xdb`，保留备份后替换，更新数据来源和校验和，执行测试并重启进程。运行中的服务仍使用启动时加载的快照。

## 验证

```bash
go test -race ./...
go vet ./...
make build
```

测试使用随附真实数据库，覆盖字段映射、IPv4 / IPv6、IPv4-mapped 地址、非法输入、数据版本不匹配、并发查询和关闭，以及 HTTP 成功、错误、请求大小限制、跨域预检。无外部数据库或网络服务依赖。

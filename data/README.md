# ip2region 数据来源

- 上游：https://github.com/lionsoul2014/ip2region
- 数据版本：`v3.18.0`（Go module `v3.18.0+incompatible`）
- IPv4：https://github.com/lionsoul2014/ip2region/blob/v3.18.0/data/ip2region_v4.xdb
- IPv6：https://github.com/lionsoul2014/ip2region/blob/v3.18.0/data/ip2region_v6.xdb
- 随附上游许可：[IP2REGION-LICENSE.md](IP2REGION-LICENSE.md)
- 格式：`Country|Province|City|ISP|iso-alpha2-code`

SHA-256：

```text
8e31bbdccb5bf21028af10592d4312ec975da0bffa108c0c5d862a12190f9ad3  ip2region_v4.xdb
939f6b46bd2b8bec3cf7c5ceb8ba782266ae9b1f35b5ba7916700dec0b7506ed  ip2region_v6.xdb
```

在项目根目录运行 `shasum -a 256 data/*.xdb` 核对。数据是离线快照，不代表实时位置；未知字段以 `0` 占位。替换数据后需重启服务。

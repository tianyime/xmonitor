#xmonitor

### 说明
 * 投资品价格监控软件，目前已支持比特币，黄金。
 * 每隔三分钟抓取价格，超过设定价格会自动发送邮件（每天最多一封）。
 * 发送邮件需要SMTP授权码或者密码，需要在 xmonitor.toml 中自行配置。

### 使用
 * go build -o xmonitor main.go
 * nohup ./xmonitor &

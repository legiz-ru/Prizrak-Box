package constant

import _ "embed"

const (
	DefaultWorkDir     = "Prizrak-Box-V3"
	DefaultCrawlDir    = "crawl"
	DefaultTemplateDir = "template"
	DefaultServerDB    = "px-server.db"
	DefaultClientDB    = "px-client.db"
	DefaultDownload    = "Download_0.yaml"
	PrefixProfile      = "Profile_"
	ProfileOrder       = "ProfileOrder"
	PrefixWebTest      = "WebTest_"
	WebTestOrder       = "WebTestOrder"
	PrefixGetter       = "Getter_"
	PrefixTemplate     = "Template_"
	TemplateSwitch     = "TemplateSwitch"
	RealIpHeader       = "RealIp_"
	SecretKey          = "SecretKey_pb"
	RecoverTmp         = "RecoverTmp"
	QuitSignal         = "QuitSignal"
	Dns                = "DNS"
	Mihomo             = "Mihomo"
)

const (
	CollectLocal     = "local"
	CollectBatch     = "batch"
	CollectClash     = "clash"
	CollectV2ray     = "v2ray"
	CollectSharelink = "share"
	CollectFuzzy     = "fuzzy"
	CollectAuto      = "auto"
	CollectSingBox   = "sing"
)

const PrizrakVersionUrl = "https://raw.githubusercontent.com/legiz-ru/Prizrak-Box/main/backend/constant/version.txt"
const PrizrakDownloadUrl = "https://github.com/legiz-ru/Prizrak-Box/releases/download/%s/%s-%s.zip"

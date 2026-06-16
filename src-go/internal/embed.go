package internal

import _ "embed"

//go:embed em/Template_0.yaml
var Template_0 []byte

//go:embed em/Template_1.yaml
var Template_1 []byte

//go:embed em/Template_2.yaml
var Template_2 []byte

//go:embed em/config_download.yaml
var PrizrakDefaultDownloadConfig []byte

//go:embed em/geoip.metadb
var GeoIp []byte

//go:embed em/GeoSite.dat
var GeoSite []byte

//go:embed em/GeoLite2-ASN.mmdb
var ASN []byte

//go:embed em/webtest.json
var DefaultWebTest []byte

//go:embed em/dns.yaml
var DefaultDNS string

//go:embed em/Model.bin
var ModelBin []byte

// BundleMRS is the bundled .mrs rule-set archive (BundleMRS.7z). mihomo's
// rules/bundle loads rule-providers declared with `path-in-bundle` from
// C.Path.BundleMRS(), which resolves to <homeDir>/BundleMRS.7z — so we only need
// to drop this file into the home dir (see releaseGeoData).
//
//go:embed em/BundleMRS.7z
var BundleMRS []byte

// ZashboardZip is the bundled zashboard web panel (gh-pages-misans-only build).
// It is unpacked at startup into <home>/ui/zashboard (see releaseZashboard) and
// served by the controller at /ui/ via route.SetUIPath (external-ui).
//
//go:embed em/zashboard.zip
var ZashboardZip []byte

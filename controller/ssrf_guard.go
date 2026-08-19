package controller

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// isPrivateOrLocalIP 判断 IP 是否属于不应被服务端主动访问的内网/本地地址段。
// 覆盖: loopback / link-local / 未指定 / RFC1918 / CGNAT / 云元数据段。
func isPrivateOrLocalIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		switch {
		case ip4[0] == 10: // 10.0.0.0/8
			return true
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31: // 172.16.0.0/12
			return true
		case ip4[0] == 192 && ip4[1] == 168: // 192.168.0.0/16
			return true
		case ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127: // 100.64.0.0/10 CGNAT
			return true
		case ip4[0] == 169 && ip4[1] == 254: // 169.254.0.0/16 (含云元数据 169.254.169.254)
			return true
		case ip4[0] == 0: // 0.0.0.0/8
			return true
		}
	}
	return false
}

// validateOutboundBaseURL 对渠道 base_url 做 SSRF 防护校验。
// 拒绝 localhost / 内网 / 链路本地 / 云元数据地址(含 DNS 解析后二次校验,
// 防止攻击者用指向内网的域名绕过字面黑名单)。
func validateOutboundBaseURL(rawURL string) error {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid base url")
	}
	host := u.Hostname()
	if strings.EqualFold(host, "localhost") {
		return fmt.Errorf("base url points to local address")
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("cannot resolve base url host")
	}
	for _, ip := range ips {
		if isPrivateOrLocalIP(ip) {
			return fmt.Errorf("base url resolves to internal address")
		}
	}
	return nil
}

package varycontrol

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// AcceptEncoding 表示Accept-Encoding头中的一个编码项
//
// 字段说明:
//   - Value: 编码名称，如 "gzip", "br", "deflate"
//   - Q: 权重值（q值），范围0.0-1.0，默认1.0
//
// 示例:
//   - "gzip" → {Value: "gzip", Q: 1.0}
//   - "gzip;q=1.0" → {Value: "gzip", Q: 1.0}
//   - "br;q=0.9" → {Value: "br", Q: 0.9}
type AcceptEncoding struct {
	Value string
	Q     float64
}

// AcceptEncodingList Accept-Encoding列表，按q值降序排序
//
// 排序规则: q值高的在前，q值相同的按出现顺序
//
// 示例:
//
//	输入: "gzip,br;q=0.9,deflate;q=0.5"
//	输出: [{gzip,1.0}, {br,0.9}, {deflate,0.5}]
type AcceptEncodingList []AcceptEncoding

func (a AcceptEncodingList) Len() int           { return len(a) }
func (a AcceptEncodingList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a AcceptEncodingList) Less(i, j int) bool { return a[i].Q > a[j].Q }

// ParseAcceptEncoding 解析Accept-Encoding头
//
// 功能: 将Accept-Encoding头字符串解析为AcceptEncodingList
//
// 支持的格式:
//   - "gzip,br,deflate" → q值默认为1.0
//   - "gzip;q=1.0,br;q=0.9,deflate;q=0.5" → 指定q值
//   - "gzip;q=0" → 明确不支持gzip
//
// 处理逻辑:
//  1. 按逗号分割多个编码
//  2. 对每个编码解析q值（如果有）
//  3. 按q值降序排序
//
// 示例:
//
//	输入: "gzip,br;q=0.9,deflate;q=0.5"
//	输出: [{gzip,1.0}, {br,0.9}, {deflate,0.5}]
func ParseAcceptEncoding(header string) AcceptEncodingList {
	if header == "" {
		return nil
	}

	result := make(AcceptEncodingList, 0)
	parts := strings.Split(header, ",")

	for _, part := range parts {
		encoding := strings.TrimSpace(part)
		if encoding == "" {
			continue
		}

		q := 1.0
		if strings.Contains(encoding, ";") {
			segments := strings.Split(encoding, ";")
			encoding = strings.TrimSpace(segments[0])
			for _, seg := range segments[1:] {
				seg = strings.TrimSpace(seg)
				if strings.HasPrefix(seg, "q=") {
					qValue := strings.TrimPrefix(seg, "q=")
					qValue = strings.TrimSpace(qValue)
					if parsedQ, err := strconv.ParseFloat(qValue, 64); err == nil {
						q = parsedQ
					}
				}
			}
		}

		result = append(result, AcceptEncoding{
			Value: encoding,
			Q:     q,
		})
	}

	sort.Sort(result)
	return result
}

// SupportsEncoding 检查客户端是否支持指定的编码
//
// 功能: 根据Accept-Encoding列表判断客户端是否支持某个编码
//
// 匹配规则（兼容性匹配，非精确字符串匹配）:
//  1. 如果Accept-Encoding列表为空，只支持空编码或identity
//  2. 跳过q<=0的编码（明确不支持）
//  3. 如果有"*"，支持所有编码
//  4. 如果有"identity"，只支持空编码或identity
//  5. 精确匹配编码名称（不区分大小写）
//
// 特殊值处理:
//   - "*": 通配符，支持所有编码
//   - "identity": 不接受压缩，只支持未压缩内容
//   - q=0: 明确不支持该编码
//
// 示例:
//
//   - Accept-Encoding: "gzip,br,deflate"
//
//   - SupportsEncoding(list, "br") → true
//
//   - SupportsEncoding(list, "gzip") → true
//
//   - SupportsEncoding(list, "compress") → false
//
//   - Accept-Encoding: "gzip;q=0,br;q=0.9"
//
//   - SupportsEncoding(list, "gzip") → false（q=0）
//
//   - SupportsEncoding(list, "br") → true
//
//   - Accept-Encoding: "identity"
//
//   - SupportsEncoding(list, "") → true
//
//   - SupportsEncoding(list, "gzip") → false
func SupportsEncoding(list AcceptEncodingList, encoding string) bool {
	if len(list) == 0 {
		return encoding == "" || encoding == "identity"
	}

	for _, item := range list {
		if item.Q <= 0 {
			continue
		}

		if item.Value == "*" {
			return true
		}

		if item.Value == "identity" {
			if encoding == "" || encoding == "identity" {
				return true
			}
			continue
		}

		if strings.EqualFold(item.Value, encoding) {
			return true
		}
	}

	return false
}

// GetBestSupportedEncoding 从客户端支持的编码中选择最优的服务端编码
//
// 功能: 根据客户端的Accept-Encoding列表，从服务端支持的编码中选择最优的
//
// 选择规则:
//  1. 如果客户端没有Accept-Encoding，优先选择未压缩（identity）
//  2. 按客户端的q值降序遍历
//  3. 跳过q<=0的编码
//  4. 选择第一个匹配的服务端编码
//
// 示例:
//
//   - 客户端: "gzip,br,deflate"
//
//   - 服务端: ["br", "gzip", "deflate"]
//
//   - 返回: "br"（客户端优先级最高）
//
//   - 客户端: "gzip;q=0.9,br;q=1.0"
//
//   - 服务端: ["br", "gzip"]
//
//   - 返回: "br"（q值最高）
func GetBestSupportedEncoding(list AcceptEncodingList, serverEncodings []string) string {
	if len(list) == 0 {
		if contains(serverEncodings, "") || contains(serverEncodings, "identity") {
			return ""
		}
		if len(serverEncodings) > 0 {
			return serverEncodings[0]
		}
		return ""
	}

	for _, item := range list {
		if item.Q <= 0 {
			continue
		}

		if item.Value == "*" {
			if len(serverEncodings) > 0 {
				return serverEncodings[0]
			}
			return ""
		}

		if item.Value == "identity" {
			if contains(serverEncodings, "") || contains(serverEncodings, "identity") {
				return ""
			}
			continue
		}

		for _, serverEnc := range serverEncodings {
			if strings.EqualFold(item.Value, serverEnc) {
				return serverEnc
			}
		}
	}

	return ""
}

// contains 检查字符串是否在切片中（不区分大小写）
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

// GetRequestAcceptEncoding 从请求中获取Accept-Encoding列表
//
// 功能: 解析请求头中的Accept-Encoding
//
// 示例:
//
//   - 请求头: "Accept-Encoding: gzip,br,deflate"
//
//   - 返回: [{gzip,1.0}, {br,1.0}, {deflate,1.0}]
//
//   - 请求头: "Accept-Encoding: gzip;q=1.0,br;q=0.9"
//
//   - 返回: [{gzip,1.0}, {br,0.9}]
func GetRequestAcceptEncoding(req *http.Request) AcceptEncodingList {
	acceptEncoding := req.Header.Get("Accept-Encoding")
	return ParseAcceptEncoding(acceptEncoding)
}

// GetResponseContentEncoding 从响应中获取Content-Encoding
//
// 功能: 获取响应的Content-Encoding头（第一个编码）
//
// 注意: HTTP允许多个Content-Encoding，但这里只返回第一个
//
// 示例:
//
//   - 响应头: "Content-Encoding: br"
//
//   - 返回: "br"
//
//   - 响应头: "Content-Encoding: gzip,br"
//
//   - 返回: "gzip"
func GetResponseContentEncoding(resp *http.Response) string {
	encoding := resp.Header.Get("Content-Encoding")
	if encoding == "" {
		return ""
	}

	encodings := strings.Split(encoding, ",")
	if len(encodings) > 0 {
		return strings.TrimSpace(encodings[0])
	}
	return ""
}

// NormalizeContentEncoding 规范化Content-Encoding
//
// 功能: 将Content-Encoding转换为小写并去除空格
//
// 特殊处理:
//   - 空字符串 → 返回空
//   - "identity" → 返回空（identity表示未压缩）
//
// 示例:
//   - "GZIP" → "gzip"
//   - " Br " → "br"
//   - "identity" → ""
//   - "" → ""
func NormalizeContentEncoding(encoding string) string {
	if encoding == "" || encoding == "identity" {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(encoding))
}

// ShouldUseVaryCache 判断是否应该使用vary缓存
//
// 功能: 检查响应头中是否包含Vary: Accept-Encoding且有Content-Encoding
//
// 判断规则:
//  1. 响应头中必须有Vary头
//  2. Vary头中必须包含Accept-Encoding
//  3. 响应头中必须有Content-Encoding（非空）
//
// 注意: 这是判断是否需要启用vary缓存的关键函数
//
// 示例:
//   - 响应: Vary: Accept-Encoding + Content-Encoding: br → true
//   - 响应: Vary: Accept-Encoding + Content-Encoding: "" → false
//   - 响应: Vary: User-Agent + Content-Encoding: br → false（没有Accept-Encoding）
func ShouldUseVaryCache(req *http.Request, resp *http.Response) bool {
	varyHeaders := resp.Header.Values("Vary")
	if len(varyHeaders) == 0 {
		return false
	}

	varyKeys := Clean(varyHeaders...)
	for _, key := range varyKeys {
		if strings.EqualFold(key, "Accept-Encoding") {
			contentEncoding := GetResponseContentEncoding(resp)
			if contentEncoding != "" {
				return true
			}
		}
	}

	return false
}

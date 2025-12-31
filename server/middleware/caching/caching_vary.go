package caching

import (
	"net/http"

	"github.com/omalloc/tavern/api/defined/v1/storage/object"
	"github.com/omalloc/tavern/pkg/x/http/varycontrol"
)

var _ Processor = (*VaryProcessor)(nil)

type VaryOption func(r *VaryProcessor)

type VaryProcessor struct {
	maxLimit      int
	varyIgnoreKey map[string]struct{}
}

// Lookup implements Processor.
//
// Vary缓存设计（三层架构）:
//  1. FlagCache (Flags=0): 普通缓存，没有vary头
//  2. FlagVaryIndex (Flags=1): Vary索引，URL的完整hash-key，VirtualKey存储所有vary组合
//  3. FlagVaryCache (Flags=2): Vary实际缓存，每个VirtualKey对应的实际缓存数据
//
// 查找流程:
//   - FlagCache: 直接命中，返回true
//   - FlagVaryIndex: 查找VirtualKey列表，找到匹配的VaryCache
//   - FlagVaryCache: 直接命中（已经找到了），返回true
func (v *VaryProcessor) Lookup(caching *Caching, req *http.Request) (bool, error) {
	if caching.hasNoCache() {
		return false, nil
	}

	switch caching.md.Flags {
	case object.FlagCache:
		// 普通缓存，直接命中
		return true, nil

	case object.FlagVaryIndex:
		// Vary索引，需要查找对应的VaryCache
		return v.lookupVaryIndex(caching, req)

	case object.FlagVaryCache:
		// Vary实际缓存，直接命中（已经找到了）
		return true, nil

	default:
		return false, nil
	}
}

// PreRequest implements Processor.
func (v *VaryProcessor) PreRequest(caching *Caching, req *http.Request) (*http.Request, error) {
	return req, nil
}

// PostRequest implements Processor.
//
// 根据响应头中的Vary信息，更新缓存结构:
//   - 如果响应没有Vary头，将VaryIndex降级为普通缓存
//   - 如果响应有Vary头，根据当前缓存类型进行相应处理:
//   - FlagCache: 升级为VaryIndex + 创建VaryCache
//   - FlagVaryIndex: 更新VaryIndex的VirtualKey列表
//   - FlagVaryCache: 更新VaryIndex的VirtualKey列表（通过rootmd）
func (v *VaryProcessor) PostRequest(caching *Caching, req *http.Request, resp *http.Response) (*http.Response, error) {
	if caching.md == nil {
		return resp, nil
	}

	varyHeaders := varycontrol.Clean(resp.Header.Values("Vary")...)

	// 响应没有Vary头，如果当前是VaryIndex则降级为普通缓存
	if len(varyHeaders) == 0 {
		if caching.md.Flags == object.FlagVaryIndex {
			v.downgradeToNormal(caching, req)
		}
		return resp, nil
	}

	// 响应有Vary头，根据当前缓存类型进行处理
	switch caching.md.Flags {
	case object.FlagCache:
		// 普通缓存升级为VaryIndex
		v.upgradeToVaryIndex(caching, req, resp, varyHeaders)

	case object.FlagVaryIndex:
		// 更新VaryIndex的VirtualKey列表
		v.updateVaryIndex(caching, req, resp, varyHeaders)

	case object.FlagVaryCache:
		// 更新VaryCache（通过rootmd更新VaryIndex）
		v.updateVaryCache(caching, req, resp, varyHeaders)
	}

	return resp, nil
}

// lookupVaryIndex 查找VaryIndex对应的VaryCache
//
// 查找流程:
//  1. 获取VaryIndex的VirtualKey列表（所有vary组合）
//  2. 检查Vary头中是否包含Accept-Encoding
//  3. 如果包含Accept-Encoding，使用兼容性匹配（非精确字符串匹配）
//  4. 如果不包含Accept-Encoding，使用精确匹配
//
// 兼容性匹配示例:
//   - 客户端: Accept-Encoding: gzip,br,deflate
//   - VaryCache: Content-Encoding: br → 命中（客户端支持br）
//   - VaryCache: Content-Encoding: gzip → 命中（客户端支持gzip）
//   - VaryCache: Content-Encoding: compress → 不命中（客户端不支持compress）
func (v *VaryProcessor) lookupVaryIndex(caching *Caching, req *http.Request) (bool, error) {
	virtualKeys := caching.md.VirtualKey
	if len(virtualKeys) == 0 {
		return false, nil
	}

	varyHeaders := varycontrol.Clean(caching.md.Headers.Values("Vary")...)
	if len(varyHeaders) == 0 {
		return false, nil
	}

	// 检查Vary头中是否包含Accept-Encoding
	hasAcceptEncoding := false
	for _, key := range varyHeaders {
		if key == "accept-encoding" {
			hasAcceptEncoding = true
			break
		}
	}

	// 根据是否包含Accept-Encoding选择匹配策略
	if !hasAcceptEncoding {
		// 不包含Accept-Encoding，使用精确匹配
		return v.findExactVaryCache(caching, req, virtualKeys)
	}

	// 包含Accept-Encoding，使用兼容性匹配
	return v.findAcceptEncodingCompatibleCache(caching, req, virtualKeys)
}

// findExactVaryCache 精确匹配查找VaryCache
//
// 遍历VirtualKey列表，查找完全匹配的VaryCache:
//  1. 根据VirtualKey生成object ID
//  2. 查找对应的metadata
//  3. 检查是否为VaryCache（Flags=2）
//  4. 如果是VaryCache，设置caching.rootmd和caching.md
//
// 注意: 精确匹配适用于非Accept-Encoding的vary头（如User-Agent）
func (v *VaryProcessor) findExactVaryCache(caching *Caching, req *http.Request, virtualKeys []string) (bool, error) {
	for _, vkey := range virtualKeys {
		// 根据VirtualKey生成object ID
		vid, err := newObjectIDFromRequest(req, vkey, caching.opt.IncludeQueryInCacheKey)
		if err != nil {
			continue
		}

		// 查找对应的metadata
		vmd, err := caching.bucket.Lookup(req.Context(), vid)
		if err != nil {
			continue
		}

		// 检查是否为VaryCache（Flags=2）
		if vmd.Flags == object.FlagVaryCache {
			// 设置rootmd为VaryIndex（用于后续更新）
			caching.rootmd = caching.md
			// 设置id为VaryCache的ID
			caching.id = vmd.ID
			// 设置md为VaryCache（用于返回数据）
			caching.md = vmd
			return true, nil
		}
	}

	return false, nil
}

// findAcceptEncodingCompatibleCache Accept-Encoding兼容性匹配查找VaryCache
//
// 核心逻辑: 不是精确字符串匹配，而是检查客户端是否支持缓存条目的Content-Encoding
//
// 匹配规则:
//  1. 解析客户端的Accept-Encoding（支持q值，如: gzip;q=1.0,br;q=0.9）
//  2. 遍历所有VirtualKey，查找对应的VaryCache
//  3. 检查VaryCache的Content-Encoding是否被客户端支持
//  4. 如果支持，返回该VaryCache
//
// 兼容性匹配示例:
//   - 客户端: Accept-Encoding: gzip,br,deflate
//   - VaryCache1: Content-Encoding: br → 命中（客户端支持br）
//   - VaryCache2: Content-Encoding: gzip → 命中（客户端支持gzip）
//   - VaryCache3: Content-Encoding: deflate → 命中（客户端支持deflate）
//   - VaryCache4: Content-Encoding: compress → 不命中（客户端不支持compress）
//
// 特殊值处理:
//   - *: 支持所有编码
//   - identity: 不接受压缩
//   - q=0: 明确不支持该编码
func (v *VaryProcessor) findAcceptEncodingCompatibleCache(caching *Caching, req *http.Request, virtualKeys []string) (bool, error) {
	// 解析客户端的Accept-Encoding（支持q值）
	acceptEncodingList := varycontrol.GetRequestAcceptEncoding(req)
	if len(acceptEncodingList) == 0 {
		// 客户端没有Accept-Encoding，使用精确匹配
		return v.findExactVaryCache(caching, req, virtualKeys)
	}

	// 遍历所有VirtualKey，查找兼容的VaryCache
	for _, vkey := range virtualKeys {
		// 根据VirtualKey生成object ID
		vid, err := newObjectIDFromRequest(req, vkey, caching.opt.IncludeQueryInCacheKey)
		if err != nil {
			continue
		}

		// 查找对应的metadata
		vmd, err := caching.bucket.Lookup(req.Context(), vid)
		if err != nil {
			continue
		}

		// 检查是否为VaryCache（Flags=2）
		if vmd.Flags != object.FlagVaryCache {
			continue
		}

		// 获取VaryCache的Content-Encoding
		contentEncoding := vmd.Headers.Get("Content-Encoding")

		// 检查客户端是否支持这个Content-Encoding（兼容性匹配）
		if varycontrol.SupportsEncoding(acceptEncodingList, contentEncoding) {
			// 命中！
			// 设置rootmd为VaryIndex（用于后续更新）
			caching.rootmd = caching.md
			// 设置id为VaryCache的ID
			caching.id = vmd.ID
			// 设置md为VaryCache（用于返回数据）
			caching.md = vmd
			return true, nil
		}
	}

	return false, nil
}

// upgradeToVaryIndex 将普通缓存升级为VaryIndex
//
// 场景: 首次请求，响应有Vary头，当前缓存为普通缓存
//
// 处理流程:
//  1. 根据Vary头和请求/响应头构建VirtualKey
//     - 对于Accept-Encoding: 使用响应的Content-Encoding作为key
//     - 对于其他vary头: 使用请求头的值作为key
//  2. 将当前metadata升级为VaryIndex（Flags=1）
//  3. 设置VirtualKey为构建的key
//  4. 存储VaryIndex
//
// 示例:
//   - 请求: GET /api/data + Accept-Encoding: gzip,br
//   - 响应: Content-Encoding: br + Vary: Accept-Encoding
//   - VirtualKey: "accept-encoding=br"
//   - VaryIndex: Flags=1, VirtualKey=["accept-encoding=br"]
//   - VaryCache: Flags=2, Content-Encoding=br（在doProxy中自动创建）
func (v *VaryProcessor) upgradeToVaryIndex(caching *Caching, req *http.Request, resp *http.Response, varyHeaders varycontrol.Key) {
	// 根据Vary头和请求/响应头构建VirtualKey
	varyKey := varycontrol.BuildVaryKeyForCache(varyHeaders, req.Header, resp.Header)
	if varyKey == "" {
		return
	}

	// 将当前metadata升级为VaryIndex（Flags=1）
	caching.md.Flags = object.FlagVaryIndex
	// 设置VirtualKey为构建的key
	caching.md.VirtualKey = []string{varyKey}

	// 存储VaryIndex
	_ = caching.bucket.Store(req.Context(), caching.md)
}

// updateVaryIndex 更新VaryIndex的VirtualKey列表
//
// 场景: 后续请求，响应有Vary头，发现新的vary组合
//
// 处理流程:
//  1. 根据Vary头和请求/响应头构建VirtualKey
//  2. 检查VirtualKey是否已存在于VaryIndex的VirtualKey列表中
//  3. 如果不存在，添加到列表中
//  4. 如果超过maxLimit，删除最旧的VirtualKey
//  5. 更新VaryIndex
//
// 示例:
//   - 当前VaryIndex: VirtualKey=["accept-encoding=br"]
//   - 新请求: Accept-Encoding: gzip
//   - 新响应: Content-Encoding: gzip + Vary: Accept-Encoding
//   - 新VirtualKey: "accept-encoding=gzip"
//   - 更新后: VirtualKey=["accept-encoding=br", "accept-encoding=gzip"]
func (v *VaryProcessor) updateVaryIndex(caching *Caching, req *http.Request, resp *http.Response, varyHeaders varycontrol.Key) {
	// 根据Vary头和请求/响应头构建VirtualKey
	varyKey := varycontrol.BuildVaryKeyForCache(varyHeaders, req.Header, resp.Header)
	if varyKey == "" {
		return
	}

	// 检查VirtualKey是否已存在
	if v.containsVirtualKey(caching.md.VirtualKey, varyKey) {
		return // 已存在，不重复添加
	}

	// 添加新的VirtualKey
	virtualKeys := append(caching.md.VirtualKey, varyKey)
	// 限制VirtualKey数量（默认100）
	if len(virtualKeys) > v.maxLimit {
		virtualKeys = virtualKeys[len(virtualKeys)-v.maxLimit:]
	}

	// 更新VaryIndex的VirtualKey列表
	caching.md.VirtualKey = virtualKeys
	_ = caching.bucket.Store(req.Context(), caching.md)
}

// updateVaryCache 更新VaryCache（通过rootmd更新VaryIndex）
//
// 场景: 当前是VaryCache，响应有Vary头，发现新的vary组合
//
// 处理流程:
//  1. 根据Vary头和请求/响应头构建VirtualKey
//  2. 通过rootmd访问VaryIndex
//  3. 检查VirtualKey是否已存在于VaryIndex的VirtualKey列表中
//  4. 如果不存在，添加到列表中
//  5. 如果超过maxLimit，删除最旧的VirtualKey
//  6. 更新VaryIndex（通过rootmd）
//
// 注意: rootmd是VaryIndex的引用，通过它来更新VaryIndex的VirtualKey列表
func (v *VaryProcessor) updateVaryCache(caching *Caching, req *http.Request, resp *http.Response, varyHeaders varycontrol.Key) {
	if caching.rootmd == nil {
		return
	}

	// 根据Vary头和请求/响应头构建VirtualKey
	varyKey := varycontrol.BuildVaryKeyForCache(varyHeaders, req.Header, resp.Header)
	if varyKey == "" {
		return
	}

	// 检查VirtualKey是否已存在
	if !v.containsVirtualKey(caching.rootmd.VirtualKey, varyKey) {
		// 添加新的VirtualKey
		virtualKeys := append(caching.rootmd.VirtualKey, varyKey)
		// 限制VirtualKey数量（默认100）
		if len(virtualKeys) > v.maxLimit {
			virtualKeys = virtualKeys[len(virtualKeys)-v.maxLimit:]
		}

		// 更新VaryIndex的VirtualKey列表（通过rootmd）
		caching.rootmd.VirtualKey = virtualKeys
		_ = caching.bucket.Store(req.Context(), caching.rootmd)
	}
}

// downgradeToNormal 将VaryIndex降级为普通缓存
//
// 场景: 响应没有Vary头，但当前缓存是VaryIndex
//
// 处理流程:
//  1. 将metadata降级为普通缓存（Flags=0）
//  2. 清空VirtualKey列表
//  3. 存储更新
//
// 示例:
//   - 当前: VaryIndex (Flags=1, VirtualKey=["accept-encoding=br"])
//   - 响应: 没有Vary头
//   - 降级后: 普通缓存 (Flags=0, VirtualKey=[])
func (v *VaryProcessor) downgradeToNormal(caching *Caching, req *http.Request) {
	// 将metadata降级为普通缓存（Flags=0）
	caching.md.Flags = object.FlagCache
	// 清空VirtualKey列表
	caching.md.VirtualKey = nil
	// 存储更新
	_ = caching.bucket.Store(req.Context(), caching.md)
}

// containsVirtualKey 检查VirtualKey是否已存在于列表中
func (v *VaryProcessor) containsVirtualKey(keys []string, key string) bool {
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}

func NewVaryProcessor(opts ...VaryOption) *VaryProcessor {
	v := &VaryProcessor{
		maxLimit:      100,
		varyIgnoreKey: make(map[string]struct{}),
	}

	for _, opt := range opts {
		opt(v)
	}
	return v
}

// WithVaryMaxLimit 设置VaryIndex的VirtualKey最大数量
func WithVaryMaxLimit(limit int) VaryOption {
	return func(r *VaryProcessor) {
		r.maxLimit = limit
	}
}

// WithVaryIgnoreKeys 设置需要忽略的vary头
func WithVaryIgnoreKeys(keys ...string) VaryOption {
	return func(r *VaryProcessor) {
		for _, key := range keys {
			r.varyIgnoreKey[key] = struct{}{}
		}
	}
}

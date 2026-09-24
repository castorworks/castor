// Package rediskey 为共享 Redis 中的每套部署划出独占的 key 空间。
//
// 多套 Castor 可以共用同一个 Redis：每个实例的 key 都以 General.InstanceID 为前缀，
// RSA 密钥、字典缓存、限流计数、验证码、token 黑名单互不可见。所有访问 Redis 的代码
// 都必须经 Namespace.Key 构造 key（internal/architecture_test.go 把关）。
package rediskey

// Namespace 是一个实例在共享 Redis 中的 key 前缀（即 General.InstanceID）。
type Namespace string

// Key 返回 name 在本命名空间下的完整 key：`{namespace}:{name}`。
func (n Namespace) Key(name string) string {
	return string(n) + ":" + name
}

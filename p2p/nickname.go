package p2p

// 昵称 → 地址
var NicknameToAddress = map[string]string{
	// 这里填你们约定好的昵称
	"qlh": "2c04dcbaf58a0ed895381a26976569e74bbeba656f318c36ad6906301ae2edc0",
	// 举例：你的地址
	"yyf":  "97488901d0b481b2bcb537fdc6babeb9e48aca3031326ca3b9d6e73d13ab7a7e",
	"991":  "b46763518857692729d92fb0bed81d82713fa583ceb22a1634f9c0d4bf9e8736",
	"rain": "4ce8585ed3b5f5df39fdd968e0669fcdbed6a1eda6e5893cabf20f0fdf505000",
	"lzl":  "a60c4dfd8ae8e16d0fa4708d929b5bc016a3c125326523018aa2b8e1e5a95b22",
}

// 根据昵称或地址，解析出真正的地址
// - 如果传进来的是昵称，就查表返回地址
// - 如果传进来本来就是 64 位 hex 地址，就原样返回
func ResolveAddress(nameOrAddr string) string {
	if addr, ok := NicknameToAddress[nameOrAddr]; ok {
		return addr
	}
	return nameOrAddr
}

// 把地址转换成展示用的「昵称/地址」字符串
// 如果有昵称：返回 "qlh (2c04dcba...edc0)"
// 如果没有昵称：直接返回地址
func DisplayName(addr string) string {
	for name, a := range NicknameToAddress {
		if a == addr {
			// 做个短一点的展示
			if len(addr) > 10 {
				short := addr[:8] + "..." + addr[len(addr)-4:]
				return name + " (" + short + ")"
			}
			return name + " (" + addr + ")"
		}
	}
	return addr
}

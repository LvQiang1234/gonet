package base

import (
	"math/rand"
	"time"
)

type (
	Rand struct {
		*rand.Rand
	}
)

//随机一个从i~n的整数
func (this *Rand) RandI(i int, n int) int {
	if i > n {
		Assert(false, "Rand::RandI: inverted range")
		return i
	}

	return int(i + this.Int()%(n-i+1))
}

//随机一个从i~n的整数
func (this *Rand) RandF(i float32, n float32) float32 {
	if i > n {
		Assert(false, "Rand::RandF: inverted range")
		return i
	}

	return (i + (n-i)*this.Float32())
}

//全部的随机生成器
var RAND = Rand{rand.New(rand.NewSource(time.Now().UnixNano()))}

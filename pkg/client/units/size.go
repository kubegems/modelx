package units

import "fmt"

const (
	KB = 1000
	MB = 1000 * KB
	GB = 1000 * MB
	TB = 1000 * GB
	PB = 1000 * TB

	KiB = 1024
	MiB = 1024 * KiB
	GiB = 1024 * MiB
	TiB = 1024 * GiB
	PiB = 1024 * TiB
)

type unitMap map[byte]int64

var (
	decimalMap = unitMap{'k': KB, 'm': MB, 'g': GB, 't': TB, 'p': PB}
	binaryMap  = unitMap{'k': KiB, 'm': MiB, 'g': GiB, 't': TiB, 'p': PiB}
)

var (
	decimapAbbrs = []string{"B", "kB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}
	binaryAbbrs  = []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"}
)

func getSizeAndUnit(size float64, base float64, _map []string) (float64, string) {
	i := 0
	unitsLimit := len(_map) - 1
	for size >= base && i < unitsLimit {
		size = size / base
		i++
	}
	return size, _map[i]
}

func HumanSize(size float64) string {
	return HumanSizeWithPrecision(size, 3)
}

func HumanSizeWithPrecision(size float64, precision int) string {
	size, unit := getSizeAndUnit(size, 1000.0, decimapAbbrs)
	return fmt.Sprintf("%.*g%s", precision, size, unit)
}

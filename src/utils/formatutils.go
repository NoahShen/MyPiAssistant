package utils

import (
	"fmt"
	"strconv"
	"time"
)

var unitArr = []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}

func FormatSizeString(size string) string {
	sizeInt, _ := strconv.Atoi(size)
	return FormatSize(int64(sizeInt))
}

func FormatSize(size int64) string {
	rank := 0
	size2 := float64(size)
	for size2 >= 1024 && rank < len(unitArr) {
		size2 = size2 / 1024
		rank++
	}
	sizeStr := fmt.Sprintf("%.2f", size2)
	unit := unitArr[rank]
	return sizeStr + unit
}

func FormatFloatString(f string, decimal int) string {
	a, _ := strconv.ParseFloat(f, 64)
	return fmt.Sprintf("%."+string(decimal)+"f", a)
}

func FormatFloat(f float64, decimal int) string {
	return fmt.Sprintf("%."+string(decimal)+"f", f)
}

func FormatTime(sec int64) string {
	return (time.Duration(sec) * time.Second).String()
}

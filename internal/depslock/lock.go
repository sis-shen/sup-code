// Package depslock 用于锁定当前暂未被直接引用、但其他 Agent 即将使用的依赖版本。
// 此文件不包含业务逻辑，仅通过空白导入确保 go.mod 保留这些依赖。
package depslock

import (
	_ "github.com/bmatcuk/doublestar/v4"
	_ "github.com/mattn/go-sqlite3"
)

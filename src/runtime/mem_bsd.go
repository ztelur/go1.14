// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build dragonfly freebsd netbsd openbsd solaris

package runtime

import (
	"unsafe"
)

/**
运行时使用 Linux 提供的 mmap、munmap 和 madvise 等系统调用实现了操作系统的内存管理抽象层，抹平了不同操作系统的差异，为运行时提供了更加方便的接口，除了 Linux 之外，运行时还实现了 BSD、Darwin、Plan9 以及 Windows 等平台上抽象层。
*/

// Don't split the stack as this function may be invoked without a valid G,
// which prevents us from allocating more stack.
// 会从操作系统中获取一大块可用的内存空间，可能为几百 KB 或者几 MB；
//go:nosplit
func sysAlloc(n uintptr, sysStat *uint64) unsafe.Pointer {
	v, err := mmap(nil, n, _PROT_READ|_PROT_WRITE, _MAP_ANON|_MAP_PRIVATE, -1, 0)
	if err != 0 {
		return nil
	}
	mSysStatInc(sysStat, n)
	return v
}

// 通知操作系统虚拟内存对应的物理内存已经不再需要了，它可以重用物理内存；
func sysUnused(v unsafe.Pointer, n uintptr) {
	madvise(v, n, _MADV_FREE)
}

// 通知操作系统应用程序需要使用该内存区域，需要保证内存区域可以安全访问；
func sysUsed(v unsafe.Pointer, n uintptr) {
}

func sysHugePage(v unsafe.Pointer, n uintptr) {
}

// Don't split the stack as this function may be invoked without a valid G,
// which prevents us from allocating more stack.
// 会在程序发生内存不足（Out-of Memory，OOM）时调用并无条件地返回内存；
//go:nosplit
func sysFree(v unsafe.Pointer, n uintptr, sysStat *uint64) {
	mSysStatDec(sysStat, n)
	munmap(v, n)
}

// 内存区域转换成保留状态，主要用于运行时的调试
func sysFault(v unsafe.Pointer, n uintptr) {
	mmap(v, n, _PROT_NONE, _MAP_ANON|_MAP_PRIVATE|_MAP_FIXED, -1, 0)
}

// Indicates not to reserve swap space for the mapping.
const _sunosMAP_NORESERVE = 0x40

// 保留操作系统中的一片内存区域，对这片内存的访问会触发异常；
func sysReserve(v unsafe.Pointer, n uintptr) unsafe.Pointer {
	flags := int32(_MAP_ANON | _MAP_PRIVATE)
	if GOOS == "solaris" || GOOS == "illumos" {
		// Be explicit that we don't want to reserve swap space
		// for PROT_NONE anonymous mappings. This avoids an issue
		// wherein large mappings can cause fork to fail.
		flags |= _sunosMAP_NORESERVE
	}
	p, err := mmap(v, n, _PROT_NONE, flags, -1, 0)
	if err != 0 {
		return nil
	}
	return p
}

const _sunosEAGAIN = 11
const _ENOMEM = 12

// 保证内存区域可以快速转换至准备就绪；
func sysMap(v unsafe.Pointer, n uintptr, sysStat *uint64) {
	mSysStatInc(sysStat, n)

	p, err := mmap(v, n, _PROT_READ|_PROT_WRITE, _MAP_ANON|_MAP_FIXED|_MAP_PRIVATE, -1, 0)
	if err == _ENOMEM || ((GOOS == "solaris" || GOOS == "illumos") && err == _sunosEAGAIN) {
		throw("runtime: out of memory")
	}
	if p != v || err != 0 {
		throw("runtime: cannot map pages in arena address space")
	}
}

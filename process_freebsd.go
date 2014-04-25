// +build freebsd

package gopsutil

import (
	"bytes"
	"encoding/binary"
	"syscall"
	"unsafe"
)

// Memory_info_ex is different between OSes
type Memory_info_exStat struct {
}

type Memory_mapsStat struct {
}

func Pids() ([]int32, error) {
	ret := make([]int32, 0)
	procs, err := processes()
	if err != nil {
		return ret, nil
	}

	for _, p := range procs {
		ret = append(ret, p.Pid)
	}

	return ret, nil
}

func (p *Process) Memory_Maps() (*[]Memory_mapsStat, error) {
	ret := make([]Memory_mapsStat, 0)
	return &ret, nil
}

func copy_params(k *Kinfo_proc, p *Process) error {
	p.Exe = byteToString(k.Ki_comm[:])
	p.Ppid = k.Ki_ppid

	return nil
}

func processes() ([]Process, error) {
	results := make([]Process, 0, 50)

	mib := []int32{CTL_KERN, KERN_PROC, KERN_PROC_PROC, 0}
	buf, length, err := call_syscall(mib)
	if err != nil {
		return results, err
	}

	// get kinfo_proc size
	k := Kinfo_proc{}
	procinfo_len := int(unsafe.Sizeof(k))
	count := int(length / uint64(procinfo_len))

	// parse buf to procs
	for i := 0; i < count; i++ {
		b := buf[i*procinfo_len : i*procinfo_len+procinfo_len]
		k, err := parse_kinfo_proc(b)
		if err != nil {
			continue
		}
		p, err := NewProcess(int32(k.Ki_pid))
		if err != nil {
			continue
		}
		copy_params(&k, p)

		results = append(results, *p)
	}

	return results, nil
}

func parse_kinfo_proc(buf []byte) (Kinfo_proc, error) {
	var k Kinfo_proc
	br := bytes.NewReader(buf)
	err := binary.Read(br, binary.LittleEndian, &k)
	if err != nil {
		return k, err
	}

	return k, nil
}

func call_syscall(mib []int32) ([]byte, uint64, error) {
	miblen := uint64(len(mib))

	// get required buffer size
	length := uint64(0)
	_, _, err := syscall.Syscall6(
		syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		uintptr(miblen),
		0,
		uintptr(unsafe.Pointer(&length)),
		0,
		0)
	if err != 0 {
		b := make([]byte, 0)
		return b, length, err
	}
	if length == 0 {
		b := make([]byte, 0)
		return b, length, err
	}
	// get proc info itself
	buf := make([]byte, length)
	_, _, err = syscall.Syscall6(
		syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		uintptr(miblen),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&length)),
		0,
		0)
	if err != 0 {
		return buf, length, err
	}

	return buf, length, nil
}

func NewProcess(pid int32) (*Process, error) {
	p := &Process{Pid: pid}
	mib := []int32{CTL_KERN, KERN_PROC, KERN_PROC_PID, p.Pid}

	buf, length, err := call_syscall(mib)
	if err != nil {
		return nil, err
	}
	proc_k := Kinfo_proc{}
	if length != uint64(unsafe.Sizeof(proc_k)) {
		return nil, err
	}

	k, err := parse_kinfo_proc(buf)
	if err != nil {
		return nil, err
	}

	copy_params(&k, p)
	return p, nil
}

package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"git.dolansoft.org/lorenz/go-zfs/nvlist"

	"git.dolansoft.org/lorenz/go-zfs/ioctl"
	"github.com/lunixbochs/struc"
)

func delimitedBufToString(buf []byte) string {
	i := 0
	for ; i < len(buf); i++ {
		if buf[i] == 0x00 {
			break
		}
	}
	return string(buf[:i])
}

func main() {
	var regs syscall.PtraceRegs

	fmt.Printf("Run %v\n", os.Args[1:])

	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Ptrace: true,
	}

	cmd.Start()
	err := cmd.Wait()
	if err != nil {
		fmt.Printf("Wait returned: %v\n", err)
	}

	pid := cmd.Process.Pid
	exit := true

	for {
		if exit {
			err = syscall.PtraceGetRegs(pid, &regs)
			if err != nil {
				break
			}

			if regs.Orig_rax == syscall.SYS_IOCTL {
				cmdSize, err := struc.Sizeof(&ioctl.Cmd{})
				if err != nil {
					panic(err)
				}
				data := make([]byte, cmdSize)
				if _, err := syscall.PtracePeekData(pid, uintptr(regs.Rdx), data); err != nil {
					panic(err)
				}
				fmt.Println(regs.Rsi)
				cmd := &ioctl.Cmd{}
				if err := struc.UnpackWithOrder(bytes.NewReader(data), cmd, binary.LittleEndian); err != nil {
					panic(err)
				}
				name := delimitedBufToString(cmd.Name[:])
				if len(name) > 0 {
					fmt.Printf("name: %v\n", name)
				}
				if cmd.Cookie != 0 {
					fmt.Printf("cookie: %v\n", cmd.Cookie)
				}
				/*cmdJSON, err := json.Marshal(cmd)
				if err != nil {
					panic(err)
				}
				fmt.Printf("cmd\n---------\n%v\n", string(cmdJSON))*/
				if cmd.Nvlist_src != 0 {
					rawSrc := make([]byte, cmd.Nvlist_src_size)
					if _, err := syscall.PtracePeekData(pid, uintptr(cmd.Nvlist_src), rawSrc); err != nil {
						panic(err)
					}
					src := new(interface{})
					if err := nvlist.Unmarshal(rawSrc, src); err != nil {
						panic(err)
					}
					srcJSON, err := json.MarshalIndent(src, "", "\t")
					if err != nil {
						panic(err)
					}
					fmt.Printf("src\n%v\n", string(srcJSON))
					//spew.Dump(src)
				}
				if cmd.Nvlist_dst != 0 {
					rawDst := make([]byte, cmd.Nvlist_dst_size)
					if _, err := syscall.PtracePeekData(pid, uintptr(cmd.Nvlist_dst), rawDst); err != nil {
						panic(err)
					}
					dst := new(interface{})
					if err := nvlist.Unmarshal(rawDst, dst); err != nil {
						panic(err)
					}
					dstJSON, err := json.MarshalIndent(dst, "", "\t")
					if err != nil {
						panic(err)
					}
					fmt.Printf("dst\n%v\n", string(dstJSON))
				}
				if cmd.Nvlist_conf != 0 {
					rawConf := make([]byte, cmd.Nvlist_conf_size)
					if _, err := syscall.PtracePeekData(pid, uintptr(cmd.Nvlist_conf), rawConf); err != nil {
						panic(err)
					}
					conf := new(interface{})
					if err := nvlist.Unmarshal(rawConf, conf); err != nil {
						panic(err)
					}
					confJSON, err := json.MarshalIndent(conf, "", "\t")
					if err != nil {
						panic(err)
					}
					fmt.Printf("conf\n%v\n", string(confJSON))
				}
				fmt.Println("----------------------------")
			}
		}

		err = syscall.PtraceSyscall(pid, 0)
		if err != nil {
			panic(err)
		}

		_, err = syscall.Wait4(pid, nil, 0, nil)
		if err != nil {
			panic(err)
		}

		exit = !exit
	}
}

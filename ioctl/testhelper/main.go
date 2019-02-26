package main

import (
	"fmt"
	"os"
	"os/exec"

	"golang.org/x/sys/unix"
)

func MountSys(fsType, path string) {
	if err := os.MkdirAll(path, 0755); err != nil {
		fmt.Printf("Failed to create directory to mount %v: %v\n", fsType, err)
		unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF)
	}
	if err := unix.Mount(fsType, path, fsType, unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC, ""); err != nil {
		fmt.Printf("Failed to mount %v: %v\n", fsType, err)
		unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF)
	}
}

func main() {
	if os.Getpid() == 1 { // Running as Init
		os.MkdirAll("/dev", 0755)
		err := unix.Mount("none", "/dev", "devtmpfs", unix.MS_NOSUID, "")
		if err != nil {
			fmt.Printf("Failed to mount /dev: %v\n", err)
			return
		}
		MountSys("tmpfs", "/dev/shm")
		MountSys("sysfs", "/sys")
		cmd := exec.Command("/ioctl.test", "-v")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err == nil {
			f, err := os.Create("/successful")
			if err != nil {
				fmt.Printf("Failed to write test status: %v", err)
				return
			}
			f.Close()
		}
	}
}
